package cmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/beancore"
	"github.com/hmans/beans/internal/config"
	"github.com/hmans/beans/internal/graph"
	"github.com/hmans/beans/internal/graph/model"
)

// setupTestCore creates a temporary beans directory and initializes core/cfg.
func setupTestCore(t *testing.T) (*beancore.Core, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	beansDir := filepath.Join(tmpDir, ".beans")
	if err := os.MkdirAll(beansDir, 0755); err != nil {
		t.Fatalf("failed to create test .beans dir: %v", err)
	}

	testCfg := config.Default()
	testCore := beancore.New(beansDir, testCfg)
	if err := testCore.Load(); err != nil {
		t.Fatalf("failed to load core: %v", err)
	}

	oldCore := core
	oldCfg := cfg
	core = testCore
	cfg = testCfg

	cleanup := func() {
		core = oldCore
		cfg = oldCfg
	}

	return testCore, cleanup
}

func createBean(t *testing.T, c *beancore.Core, id, title, status, beanType string) *bean.Bean {
	t.Helper()
	b := &bean.Bean{
		ID:     id,
		Slug:   bean.Slugify(title),
		Title:  title,
		Status: status,
		Type:   beanType,
	}
	if err := c.Create(b); err != nil {
		t.Fatalf("failed to create test bean: %v", err)
	}
	return b
}

func createBeanWithPriority(t *testing.T, c *beancore.Core, id, title, status, beanType, priority string) *bean.Bean {
	t.Helper()
	b := &bean.Bean{
		ID:       id,
		Slug:     bean.Slugify(title),
		Title:    title,
		Status:   status,
		Type:     beanType,
		Priority: priority,
	}
	if err := c.Create(b); err != nil {
		t.Fatalf("failed to create test bean: %v", err)
	}
	return b
}

func createBeanWithParent(t *testing.T, c *beancore.Core, id, title, status, beanType, parent string) *bean.Bean {
	t.Helper()
	b := &bean.Bean{
		ID:     id,
		Slug:   bean.Slugify(title),
		Title:  title,
		Status: status,
		Type:   beanType,
		Parent: parent,
	}
	if err := c.Create(b); err != nil {
		t.Fatalf("failed to create test bean: %v", err)
	}
	return b
}

// --- resolveBean tests ---

func TestResolveBean(t *testing.T) {
	testCore, cleanup := setupTestCore(t)
	defer cleanup()

	createBean(t, testCore, "test-1", "Test Bean", "todo", "task")

	t.Run("found", func(t *testing.T) {
		resolver := &graph.Resolver{Core: testCore}
		b, wasArchived, err := resolveBean(resolver, "test-1", false)
		if err != nil {
			t.Fatalf("resolveBean() error = %v", err)
		}
		if b == nil {
			t.Fatal("resolveBean() returned nil")
		}
		if b.ID != "test-1" {
			t.Errorf("resolveBean().ID = %q, want %q", b.ID, "test-1")
		}
		if wasArchived {
			t.Error("resolveBean() wasArchived = true, want false")
		}
	})

	t.Run("not found", func(t *testing.T) {
		resolver := &graph.Resolver{Core: testCore}
		_, _, err := resolveBean(resolver, "nonexistent", false)
		if err == nil {
			t.Fatal("resolveBean() expected error for nonexistent bean")
		}
	})
}

// --- outputResults tests ---

func TestOutputResults(t *testing.T) {
	t.Run("single success JSON", func(t *testing.T) {
		b := &bean.Bean{ID: "test-1", Title: "Test"}
		results := []*result{{bean: b}}
		err := outputResults(results, nil, true, "completed", "Completed")
		if err != nil {
			t.Errorf("outputResults() error = %v", err)
		}
	})

	t.Run("with warnings JSON", func(t *testing.T) {
		b := &bean.Bean{ID: "test-1", Title: "Test"}
		results := []*result{{bean: b, warning: "already completed"}}
		err := outputResults(results, nil, true, "completed", "Completed")
		if err != nil {
			t.Errorf("outputResults() error = %v", err)
		}
	})

	t.Run("multiple beans JSON", func(t *testing.T) {
		results := []*result{
			{bean: &bean.Bean{ID: "test-1", Title: "First"}},
			{bean: &bean.Bean{ID: "test-2", Title: "Second"}},
		}
		err := outputResults(results, nil, true, "completed", "Completed")
		if err != nil {
			t.Errorf("outputResults() error = %v", err)
		}
	})

	t.Run("human output", func(t *testing.T) {
		b := &bean.Bean{ID: "test-1", Title: "Test"}
		results := []*result{{bean: b}}
		err := outputResults(results, nil, false, "completed", "Completed")
		if err != nil {
			t.Errorf("outputResults() error = %v", err)
		}
	})
}

// --- formatBlockerMessage tests ---

func TestFormatBlockerMessage(t *testing.T) {
	t.Run("single blocker", func(t *testing.T) {
		blockers := []*bean.Bean{{ID: "b-1", Title: "Blocker"}}
		msg := formatBlockerMessage("target", blockers)
		if !strings.Contains(msg, "b-1") || !strings.Contains(msg, "Blocker") {
			t.Errorf("formatBlockerMessage() = %q, expected to contain blocker info", msg)
		}
	})

	t.Run("multiple blockers", func(t *testing.T) {
		blockers := []*bean.Bean{
			{ID: "b-1", Title: "First"},
			{ID: "b-2", Title: "Second"},
		}
		msg := formatBlockerMessage("target", blockers)
		if !strings.Contains(msg, "b-1") || !strings.Contains(msg, "b-2") {
			t.Errorf("formatBlockerMessage() = %q, expected to contain both blockers", msg)
		}
	})
}

// --- Complete command logic tests ---

func TestCompleteCommand(t *testing.T) {
	testCore, cleanup := setupTestCore(t)
	defer cleanup()

	resolver := &graph.Resolver{Core: testCore}
	ctx := context.Background()

	t.Run("complete a todo bean", func(t *testing.T) {
		createBean(t, testCore, "comp-1", "Complete Me", "todo", "task")

		status := "completed"
		b, err := resolver.Mutation().UpdateBean(ctx, "comp-1", model.UpdateBeanInput{Status: &status})
		if err != nil {
			t.Fatalf("UpdateBean() error = %v", err)
		}
		if b.Status != "completed" {
			t.Errorf("status = %q, want %q", b.Status, "completed")
		}
	})

	t.Run("complete with summary appended", func(t *testing.T) {
		createBean(t, testCore, "comp-2", "With Summary", "todo", "task")

		status := "completed"
		appendText := "## Summary of Changes\n\nDid the thing"
		b, err := resolver.Mutation().UpdateBean(ctx, "comp-2", model.UpdateBeanInput{
			Status:  &status,
			BodyMod: &model.BodyModification{Append: &appendText},
		})
		if err != nil {
			t.Fatalf("UpdateBean() error = %v", err)
		}
		if !strings.Contains(b.Body, "## Summary of Changes") {
			t.Errorf("body = %q, expected to contain summary section", b.Body)
		}
		if !strings.Contains(b.Body, "Did the thing") {
			t.Errorf("body = %q, expected to contain summary text", b.Body)
		}
	})

	t.Run("already completed bean", func(t *testing.T) {
		createBean(t, testCore, "comp-3", "Already Done", "completed", "task")
		b, err := resolver.Query().Bean(ctx, "comp-3")
		if err != nil {
			t.Fatalf("Bean() error = %v", err)
		}
		if b.Status != "completed" {
			t.Errorf("expected bean to already be completed")
		}
	})
}

// --- Scrap command logic tests ---

func TestScrapCommand(t *testing.T) {
	testCore, cleanup := setupTestCore(t)
	defer cleanup()

	resolver := &graph.Resolver{Core: testCore}
	ctx := context.Background()

	t.Run("scrap a todo bean with reason", func(t *testing.T) {
		createBean(t, testCore, "scrap-1", "Scrap Me", "todo", "task")

		status := "scrapped"
		appendText := "## Reasons for Scrapping\n\nNo longer needed"
		b, err := resolver.Mutation().UpdateBean(ctx, "scrap-1", model.UpdateBeanInput{
			Status:  &status,
			BodyMod: &model.BodyModification{Append: &appendText},
		})
		if err != nil {
			t.Fatalf("UpdateBean() error = %v", err)
		}
		if b.Status != "scrapped" {
			t.Errorf("status = %q, want %q", b.Status, "scrapped")
		}
		if !strings.Contains(b.Body, "Reasons for Scrapping") {
			t.Errorf("body = %q, expected to contain reason section", b.Body)
		}
	})
}

// --- Start command logic tests ---

func TestStartCommand(t *testing.T) {
	testCore, cleanup := setupTestCore(t)
	defer cleanup()

	resolver := &graph.Resolver{Core: testCore}
	ctx := context.Background()

	t.Run("start a todo bean", func(t *testing.T) {
		createBean(t, testCore, "start-1", "Start Me", "todo", "task")

		status := "in-progress"
		b, err := resolver.Mutation().UpdateBean(ctx, "start-1", model.UpdateBeanInput{Status: &status})
		if err != nil {
			t.Fatalf("UpdateBean() error = %v", err)
		}
		if b.Status != "in-progress" {
			t.Errorf("status = %q, want %q", b.Status, "in-progress")
		}
	})

	t.Run("blocked bean detected", func(t *testing.T) {
		blocker := createBean(t, testCore, "blocker-1", "Blocker", "todo", "task")
		blocked := createBean(t, testCore, "blocked-1", "Blocked", "todo", "task")
		blocked.BlockedBy = []string{blocker.ID}
		if err := testCore.Update(blocked, nil); err != nil {
			t.Fatalf("Save() error = %v", err)
		}

		blockers := testCore.FindActiveBlockers("blocked-1")
		if len(blockers) == 0 {
			t.Fatal("expected bean to be blocked")
		}
		if blockers[0].ID != "blocker-1" {
			t.Errorf("blocker ID = %q, want %q", blockers[0].ID, "blocker-1")
		}
	})

	t.Run("blocked with completed blocker is not blocked", func(t *testing.T) {
		createBean(t, testCore, "done-blocker", "Done Blocker", "completed", "task")
		target := createBean(t, testCore, "target-1", "Target", "todo", "task")
		target.BlockedBy = []string{"done-blocker"}
		if err := testCore.Update(target, nil); err != nil {
			t.Fatalf("Save() error = %v", err)
		}

		blockers := testCore.FindActiveBlockers("target-1")
		if len(blockers) != 0 {
			t.Errorf("expected no active blockers, got %d", len(blockers))
		}
	})
}

// --- Ready/Next filter tests ---

func TestReadyFilter(t *testing.T) {
	testCore, cleanup := setupTestCore(t)
	defer cleanup()

	resolver := &graph.Resolver{Core: testCore}
	ctx := context.Background()

	createBean(t, testCore, "ready-1", "Ready Todo", "todo", "task")
	createBean(t, testCore, "ready-2", "In Progress", "in-progress", "task")
	createBean(t, testCore, "ready-3", "Completed", "completed", "task")
	createBean(t, testCore, "ready-4", "Draft", "draft", "task")
	createBean(t, testCore, "ready-5", "Another Todo", "todo", "feature")

	isBlocked := false
	filter := &model.BeanFilter{
		IsBlocked:     &isBlocked,
		ExcludeStatus: []string{"in-progress", "completed", "scrapped", "draft"},
	}

	beans, err := resolver.Query().Beans(ctx, filter)
	if err != nil {
		t.Fatalf("Beans() error = %v", err)
	}

	// Should only get ready-1 and ready-5
	if len(beans) != 2 {
		t.Fatalf("expected 2 ready beans, got %d", len(beans))
	}

	ids := make(map[string]bool)
	for _, b := range beans {
		ids[b.ID] = true
	}
	if !ids["ready-1"] || !ids["ready-5"] {
		t.Errorf("expected ready-1 and ready-5, got %v", ids)
	}
}

func TestNextPrioritySorting(t *testing.T) {
	testCore, cleanup := setupTestCore(t)
	defer cleanup()

	resolver := &graph.Resolver{Core: testCore}
	ctx := context.Background()

	createBeanWithPriority(t, testCore, "next-1", "Low Priority", "todo", "task", "low")
	createBeanWithPriority(t, testCore, "next-2", "Critical", "todo", "task", "critical")
	createBeanWithPriority(t, testCore, "next-3", "Normal", "todo", "task", "")

	isBlocked := false
	filter := &model.BeanFilter{
		IsBlocked:     &isBlocked,
		ExcludeStatus: []string{"in-progress", "completed", "scrapped", "draft"},
	}

	beans, err := resolver.Query().Beans(ctx, filter)
	if err != nil {
		t.Fatalf("Beans() error = %v", err)
	}

	sortBeans(beans, "priority", cfg)

	if len(beans) == 0 {
		t.Fatal("expected some beans")
	}
	if beans[0].ID != "next-2" {
		t.Errorf("first bean should be critical priority (next-2), got %q", beans[0].ID)
	}
}

func TestNextEmptyResult(t *testing.T) {
	_, cleanup := setupTestCore(t)
	defer cleanup()

	resolver := &graph.Resolver{Core: core}
	ctx := context.Background()

	isBlocked := false
	filter := &model.BeanFilter{
		IsBlocked:     &isBlocked,
		ExcludeStatus: []string{"in-progress", "completed", "scrapped", "draft"},
	}

	beans, err := resolver.Query().Beans(ctx, filter)
	if err != nil {
		t.Fatalf("Beans() error = %v", err)
	}
	if len(beans) != 0 {
		t.Errorf("expected 0 beans, got %d", len(beans))
	}
}

// --- Blocked filter tests ---

func TestBlockedFilter(t *testing.T) {
	testCore, cleanup := setupTestCore(t)
	defer cleanup()

	resolver := &graph.Resolver{Core: testCore}
	ctx := context.Background()

	createBean(t, testCore, "bl-blocker", "The Blocker", "todo", "task")
	blocked := createBean(t, testCore, "bl-target", "Blocked Bean", "todo", "task")
	blocked.BlockedBy = []string{"bl-blocker"}
	if err := testCore.Update(blocked, nil); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	createBean(t, testCore, "bl-free", "Free Bean", "todo", "task")

	isBlocked := true
	filter := &model.BeanFilter{
		IsBlocked:     &isBlocked,
		ExcludeStatus: []string{"completed", "scrapped"},
	}

	beans, err := resolver.Query().Beans(ctx, filter)
	if err != nil {
		t.Fatalf("Beans() error = %v", err)
	}

	if len(beans) != 1 {
		t.Fatalf("expected 1 blocked bean, got %d", len(beans))
	}
	if beans[0].ID != "bl-target" {
		t.Errorf("blocked bean ID = %q, want %q", beans[0].ID, "bl-target")
	}

	blockers := testCore.FindActiveBlockers(beans[0].ID)
	if len(blockers) != 1 {
		t.Fatalf("expected 1 blocker, got %d", len(blockers))
	}
	if blockers[0].ID != "bl-blocker" {
		t.Errorf("blocker ID = %q, want %q", blockers[0].ID, "bl-blocker")
	}
}

// --- Milestones tests ---

func TestMilestonesAggregation(t *testing.T) {
	testCore, cleanup := setupTestCore(t)
	defer cleanup()

	createBean(t, testCore, "ms-1", "Milestone 1", "in-progress", "milestone")
	createBeanWithParent(t, testCore, "ms-child-1", "Task 1", "completed", "task", "ms-1")
	createBeanWithParent(t, testCore, "ms-child-2", "Task 2", "todo", "task", "ms-1")
	createBeanWithParent(t, testCore, "ms-child-3", "Task 3", "completed", "task", "ms-1")

	resolver := &graph.Resolver{Core: testCore}
	ctx := context.Background()
	allBeans, err := resolver.Query().Beans(ctx, nil)
	if err != nil {
		t.Fatalf("Beans() error = %v", err)
	}

	// Build children index
	children := make(map[string][]*bean.Bean)
	for _, b := range allBeans {
		if b.Parent != "" {
			children[b.Parent] = append(children[b.Parent], b)
		}
	}

	kids := children["ms-1"]
	if len(kids) != 3 {
		t.Fatalf("expected 3 children, got %d", len(kids))
	}

	byStatus := make(map[string]int)
	for _, k := range kids {
		byStatus[k.Status]++
	}

	if byStatus["completed"] != 2 {
		t.Errorf("completed = %d, want 2", byStatus["completed"])
	}
	if byStatus["todo"] != 1 {
		t.Errorf("todo = %d, want 1", byStatus["todo"])
	}
}

func TestMilestonesExcludeArchived(t *testing.T) {
	testCore, cleanup := setupTestCore(t)
	defer cleanup()

	createBean(t, testCore, "ms-active", "Active MS", "in-progress", "milestone")
	createBean(t, testCore, "ms-done", "Done MS", "completed", "milestone")

	resolver := &graph.Resolver{Core: testCore}
	ctx := context.Background()
	allBeans, err := resolver.Query().Beans(ctx, nil)
	if err != nil {
		t.Fatalf("Beans() error = %v", err)
	}

	var active, done int
	for _, b := range allBeans {
		if b.Type != "milestone" {
			continue
		}
		if cfg.IsArchiveStatus(b.Status) {
			done++
		} else {
			active++
		}
	}

	if active != 1 {
		t.Errorf("active milestones = %d, want 1", active)
	}
	if done != 1 {
		t.Errorf("done milestones = %d, want 1", done)
	}
}

// --- Progress tests ---

func TestProgressAggregation(t *testing.T) {
	testCore, cleanup := setupTestCore(t)
	defer cleanup()

	createBean(t, testCore, "pg-1", "Task 1", "todo", "task")
	createBean(t, testCore, "pg-2", "Task 2", "in-progress", "task")
	createBean(t, testCore, "pg-3", "Bug 1", "todo", "bug")
	createBean(t, testCore, "pg-4", "Feature 1", "completed", "feature")

	resolver := &graph.Resolver{Core: testCore}
	ctx := context.Background()
	allBeans, err := resolver.Query().Beans(ctx, nil)
	if err != nil {
		t.Fatalf("Beans() error = %v", err)
	}

	byStatus := make(map[string]int)
	byType := make(map[string]int)
	for _, b := range allBeans {
		byStatus[b.Status]++
		byType[b.Type]++
	}

	if len(allBeans) != 4 {
		t.Errorf("total = %d, want 4", len(allBeans))
	}
	if byStatus["todo"] != 2 {
		t.Errorf("todo = %d, want 2", byStatus["todo"])
	}
	if byStatus["in-progress"] != 1 {
		t.Errorf("in-progress = %d, want 1", byStatus["in-progress"])
	}
	if byStatus["completed"] != 1 {
		t.Errorf("completed = %d, want 1", byStatus["completed"])
	}
	if byType["task"] != 2 {
		t.Errorf("task = %d, want 2", byType["task"])
	}
	if byType["bug"] != 1 {
		t.Errorf("bug = %d, want 1", byType["bug"])
	}
	if byType["feature"] != 1 {
		t.Errorf("feature = %d, want 1", byType["feature"])
	}
}
