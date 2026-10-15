package tui

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hmans/beans/pkg/bean"
	"github.com/hmans/beans/pkg/beancore"
	"github.com/hmans/beans/pkg/beangraph"
	"github.com/hmans/beans/pkg/config"
)

func newViewModeTestList(t *testing.T) listModel {
	t.Helper()
	beansDir := filepath.Join(t.TempDir(), ".beans")
	if err := os.MkdirAll(beansDir, 0755); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	core := beancore.New(beansDir, cfg)
	if err := core.Load(); err != nil {
		t.Fatal(err)
	}
	for _, b := range []*bean.Bean{
		{ID: "open", Title: "Open", Status: "todo", Type: "task", Tags: []string{"x"}},
		{ID: "done", Title: "Done", Status: "completed", Type: "task", Tags: []string{"x"}},
		{ID: "dropped", Title: "Dropped", Status: "scrapped", Type: "task"},
		{ID: "done-epic", Title: "Done epic", Status: "completed", Type: "epic"},
		{ID: "orphan", Title: "Under done epic", Status: "todo", Type: "task", Parent: "done-epic"},
		{ID: "blocked", Title: "Blocked", Status: "todo", Type: "task", BlockedBy: []string{"open"}},
		{ID: "wip", Title: "In progress", Status: "in-progress", Type: "task"},
		{ID: "draft", Title: "Draft", Status: "draft", Type: "task"},
	} {
		b.Slug = bean.Slugify(b.Title)
		if err := core.Create(b); err != nil {
			t.Fatal(err)
		}
	}
	return newListModel(&beangraph.CoreResolver{Core: core}, cfg)
}

func loadedIDs(t *testing.T, m listModel) []string {
	t.Helper()
	msg, ok := m.loadBeans().(beansLoadedMsg)
	if !ok {
		t.Fatalf("loadBeans did not return beansLoadedMsg")
	}
	var ids []string
	for _, it := range msg.items {
		ids = append(ids, it.Bean.ID)
	}
	slices.Sort(ids)
	return ids
}

func TestListViewMode(t *testing.T) {
	tests := []struct {
		mode      viewMode
		tagFilter string
		want      []string
	}{
		{viewAll, "", []string{"blocked", "done", "done-epic", "draft", "dropped", "open", "orphan", "wip"}},
		{viewActive, "", []string{"blocked", "draft", "open", "wip"}},
		{viewUnblocked, "", []string{"draft", "open", "wip"}},
		{viewReady, "", []string{"open"}},
		{viewActive, "x", []string{"open"}},
	}
	for _, tt := range tests {
		t.Run(tt.mode.String()+"/tag="+tt.tagFilter, func(t *testing.T) {
			m := newViewModeTestList(t)
			m.viewMode = tt.mode
			m.tagFilter = tt.tagFilter
			if got := loadedIDs(t, m); !slices.Equal(got, tt.want) {
				t.Errorf("ids = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestListViewModeCycle(t *testing.T) {
	m := newListModel(nil, config.Default())
	m.tagFilter = "x"

	for _, want := range []string{"Beans (active) [tag: x]", "Beans (unblocked) [tag: x]", "Beans (ready) [tag: x]", "Beans [tag: x]"} {
		var cmd tea.Cmd
		m, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
		if cmd == nil {
			t.Fatalf("h must trigger a reload")
		}
		if got := m.title(); got != want {
			t.Errorf("title = %q, want %q", got, want)
		}
	}

	m.viewMode = viewReady
	m.clearFilter()
	if m.viewMode != viewReady {
		t.Error("clearFilter must not reset the view mode")
	}
	if got, want := m.viewModeHelp(), "show all"; got != want {
		t.Errorf("help = %q, want %q", got, want)
	}
}
