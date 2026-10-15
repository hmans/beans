package commands

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/hmans/beans/pkg/bean"
	"github.com/hmans/beans/pkg/beancore"
	"github.com/hmans/beans/pkg/config"
	"github.com/spf13/cobra"
)

// setupNextTest gives core and cfg a fresh project in a temp dir and
// restores the next flags afterwards.
func setupNextTest(t *testing.T) (*beancore.Core, string) {
	t.Helper()
	dir := t.TempDir()
	beansDir := filepath.Join(dir, ".beans")
	if err := os.MkdirAll(beansDir, 0755); err != nil {
		t.Fatal(err)
	}
	testCfg := config.Default()
	testCfg.SetConfigDir(dir)
	testCore := beancore.New(beansDir, testCfg)
	if err := testCore.Load(); err != nil {
		t.Fatal(err)
	}
	oldCore, oldCfg := core, cfg
	oldJSON, oldAll, oldIP, oldLimit := nextJSON, nextAll, nextInProgress, nextLimit
	core, cfg = testCore, testCfg
	nextJSON, nextAll, nextInProgress, nextLimit = false, false, false, 10
	t.Cleanup(func() {
		core, cfg = oldCore, oldCfg
		nextJSON, nextAll, nextInProgress, nextLimit = oldJSON, oldAll, oldIP, oldLimit
	})
	return testCore, dir
}

func mustCreate(t *testing.T, c *beancore.Core, b *bean.Bean) {
	t.Helper()
	b.Slug = bean.Slugify(b.Title)
	if err := c.Create(b); err != nil {
		t.Fatal(err)
	}
}

// decodedNext mirrors nextJSONResult; output.Bean cannot be decoded.
type decodedNext struct {
	Scope   nextScope `json:"scope"`
	Hint    string    `json:"hint"`
	Warning string    `json:"warning"`
	Rows    []struct {
		ID     string        `json:"id"`
		Lane   bean.NextLane `json:"lane"`
		Reason string        `json:"reason"`
		Front  string        `json:"front"`
	} `json:"rows"`
}

func runNextJSON(t *testing.T) decodedNext {
	t.Helper()
	nextJSON = true
	var buf bytes.Buffer
	nextCmd.SetOut(&buf)
	defer nextCmd.SetOut(nil)
	if err := nextCmd.RunE(nextCmd, nil); err != nil {
		t.Fatal(err)
	}
	var got decodedNext
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("%v\n%s", err, buf.String())
	}
	return got
}

func jsonRowIDs(r decodedNext) []string {
	ids := make([]string, len(r.Rows))
	for i, row := range r.Rows {
		ids[i] = row.ID
	}
	return ids
}

func seedNextProject(t *testing.T, c *beancore.Core) {
	t.Helper()
	mustCreate(t, c, &bean.Bean{ID: "e1", Title: "Epic one", Type: "epic", Status: "in-progress"})
	mustCreate(t, c, &bean.Bean{ID: "e2", Title: "Epic two", Type: "epic", Status: "in-progress"})
	mustCreate(t, c, &bean.Bean{ID: "t1", Title: "Task", Type: "task", Status: "todo", Parent: "e1"})
	mustCreate(t, c, &bean.Bean{ID: "d2", Title: "Draft", Type: "task", Status: "draft", Parent: "e2"})
	// Create stamps CreatedAt to the second, so a manual order fixes the
	// rank of the two rest-lane tasks.
	mustCreate(t, c, &bean.Bean{ID: "loose", Title: "Loose", Type: "task", Status: "todo", Order: "b"})
	mustCreate(t, c, &bean.Bean{ID: "doing", Title: "Doing", Type: "task", Status: "in-progress", Parent: "e1"})
	mustCreate(t, c, &bean.Bean{ID: "blocker", Title: "Blocker", Type: "task", Status: "todo", Order: "a"})
	mustCreate(t, c, &bean.Bean{ID: "blocked", Title: "Blocked draft", Type: "task", Status: "draft", Parent: "e1", BlockedBy: []string{"blocker"}})
}

func TestNextCommand(t *testing.T) {
	t.Run("default hides rest and blocked drafts, hints at fronts", func(t *testing.T) {
		c, _ := setupNextTest(t)
		seedNextProject(t, c)
		got := runNextJSON(t)
		if ids := jsonRowIDs(got); !slices.Equal(ids, []string{"t1", "d2"}) {
			t.Errorf("rows = %v, want [t1 d2]", ids)
		}
		if got.Scope.Kind != "in-progress-fronts" {
			t.Errorf("scope = %+v", got.Scope)
		}
		if !strings.Contains(got.Hint, "2 active fronts") {
			t.Errorf("hint = %q", got.Hint)
		}
		if got.Rows[1].Lane != bean.LaneRefine || got.Rows[1].Front != "e2" {
			t.Errorf("draft row = %+v", got.Rows[1])
		}
	})

	t.Run("--all adds rest, --limit cuts", func(t *testing.T) {
		c, _ := setupNextTest(t)
		seedNextProject(t, c)
		nextAll = true
		if ids := jsonRowIDs(runNextJSON(t)); !slices.Equal(ids, []string{"t1", "d2", "blocker", "loose"}) {
			t.Errorf("--all rows = %v", ids)
		}
		nextLimit = 1
		if ids := jsonRowIDs(runNextJSON(t)); !slices.Equal(ids, []string{"t1"}) {
			t.Errorf("--limit 1 rows = %v", ids)
		}
	})

	t.Run("--in-progress lists beans to resume", func(t *testing.T) {
		c, _ := setupNextTest(t)
		seedNextProject(t, c)
		nextInProgress = true
		got := runNextJSON(t)
		if ids := jsonRowIDs(got); !slices.Equal(ids, []string{"doing"}) {
			t.Errorf("rows = %v, want [doing]", ids)
		}
		nextAll = true
		if err := nextCmd.RunE(nextCmd, nil); err == nil {
			t.Error("--all with --in-progress: want error")
		}
	})

	t.Run("without any front the rest lane is shown", func(t *testing.T) {
		c, _ := setupNextTest(t)
		mustCreate(t, c, &bean.Bean{ID: "e", Title: "Idle epic", Type: "epic", Status: "todo"})
		mustCreate(t, c, &bean.Bean{ID: "a", Title: "A", Type: "task", Status: "todo", Parent: "e"})
		mustCreate(t, c, &bean.Bean{ID: "b", Title: "B", Type: "task", Status: "todo"})
		got := runNextJSON(t)
		if got.Scope.Kind != "no-fronts" {
			t.Errorf("scope = %+v, want no-fronts", got.Scope)
		}
		if ids := jsonRowIDs(got); len(ids) != 2 {
			t.Errorf("rows = %v, want a and b", ids)
		}
		for _, r := range got.Rows {
			if r.Lane != bean.LaneRest {
				t.Errorf("%s: lane = %q, want rest", r.ID, r.Lane)
			}
		}
	})

	t.Run("valid focus replaces in-progress fronts", func(t *testing.T) {
		c, _ := setupNextTest(t)
		seedNextProject(t, c)
		if err := cfg.SetLocalFocus("e2"); err != nil {
			t.Fatal(err)
		}
		got := runNextJSON(t)
		if got.Scope.Kind != "focus" || got.Scope.Focus != "e2" || got.Scope.File != config.LocalConfigFileName {
			t.Errorf("scope = %+v", got.Scope)
		}
		if got.Hint != "" || got.Warning != "" {
			t.Errorf("hint = %q, warning = %q, want both empty", got.Hint, got.Warning)
		}
		if ids := jsonRowIDs(got); !slices.Equal(ids, []string{"d2"}) {
			t.Errorf("rows = %v, want [d2]", ids)
		}
	})

	t.Run("invalid focus falls back with a warning", func(t *testing.T) {
		c, _ := setupNextTest(t)
		seedNextProject(t, c)
		if err := cfg.SetLocalFocus("t1"); err != nil {
			t.Fatal(err)
		}
		got := runNextJSON(t)
		if got.Scope.Kind != "in-progress-fronts" {
			t.Errorf("scope = %+v, want fallback", got.Scope)
		}
		if got.Warning != "focus t1 invalid: is a task, not a milestone or epic" {
			t.Errorf("warning = %q", got.Warning)
		}
	})
}

func TestFocusCommands(t *testing.T) {
	c, dir := setupNextTest(t)
	seedNextProject(t, c)
	subcommands := map[string]*cobra.Command{"set": focusSetCmd, "clear": focusClearCmd}
	run := func(args ...string) (string, error) {
		var buf bytes.Buffer
		cmd := focusCmd
		if len(args) > 0 {
			cmd, args = subcommands[args[0]], args[1:]
		}
		cmd.SetOut(&buf)
		defer cmd.SetOut(nil)
		err := cmd.RunE(cmd, args)
		return buf.String(), err
	}

	if _, err := run("set", "t1"); err == nil || !strings.Contains(err.Error(), "not a milestone or epic") {
		t.Errorf("set on a task: err = %v", err)
	}
	if _, err := run("set", "missing"); err == nil {
		t.Error("set on a missing bean: want error")
	}
	out, err := run("set", "e1")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "e1") || !strings.Contains(out, config.LocalConfigFileName) {
		t.Errorf("set output = %q", out)
	}
	if _, err := os.Stat(filepath.Join(dir, config.LocalConfigFileName)); err != nil {
		t.Errorf("local file not written: %v", err)
	}
	if out, _ := run(); !strings.Contains(out, "focus: e1") {
		t.Errorf("show output = %q", out)
	}
	if out, err := run("clear"); err != nil || !strings.Contains(out, "no focus set") {
		t.Errorf("clear: out = %q, err = %v", out, err)
	}
	if _, err := os.Stat(filepath.Join(dir, config.LocalConfigFileName)); !os.IsNotExist(err) {
		t.Errorf("local file still present after clear: %v", err)
	}

	focusJSON = true
	defer func() { focusJSON = false }()
	var got focusJSONResult
	decode := func(out string) {
		t.Helper()
		got = focusJSONResult{}
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatalf("%v\n%s", err, out)
		}
	}
	out, err = run("set", "e2")
	if err != nil {
		t.Fatal(err)
	}
	decode(out)
	if got != (focusJSONResult{Focus: "e2", File: config.LocalConfigFileName}) {
		t.Errorf("set --json = %+v", got)
	}
	stored, err := c.Get("e2")
	if err != nil {
		t.Fatal(err)
	}
	done := stored.Clone()
	done.Status = "completed"
	if err := c.Update(done, stored.ETag()); err != nil {
		t.Fatal(err)
	}
	out, _ = run()
	decode(out)
	if got.Focus != "e2" || got.Problem != "completed" {
		t.Errorf("focus --json with stale focus = %+v", got)
	}
	out, _ = run("clear")
	decode(out)
	if got != (focusJSONResult{}) {
		t.Errorf("clear --json = %+v", got)
	}
}
