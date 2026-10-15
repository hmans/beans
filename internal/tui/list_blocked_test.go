package tui

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/hmans/beans/pkg/bean"
	"github.com/hmans/beans/pkg/beancore"
	"github.com/hmans/beans/pkg/beangraph"
	"github.com/hmans/beans/pkg/config"
)

func TestListLoadBeansBlocked(t *testing.T) {
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
		{ID: "blocker", Title: "Blocker", Status: "todo", Type: "task"},
		{ID: "done-blocker", Title: "Done blocker", Status: "completed", Type: "task"},
		{ID: "direct", Title: "Blocked by field", Status: "todo", Type: "task", BlockedBy: []string{"blocker"}},
		{ID: "via-link", Title: "Blocked by link", Status: "todo", Type: "task"},
		{ID: "linker", Title: "Linker", Status: "todo", Type: "task", Blocking: []string{"via-link"}},
		{ID: "resolved", Title: "Blocker done", Status: "todo", Type: "task", BlockedBy: []string{"done-blocker"}},
		{ID: "epic", Title: "Blocked epic", Status: "todo", Type: "epic", BlockedBy: []string{"blocker"}},
		{ID: "child", Title: "Child of blocked epic", Status: "todo", Type: "task", Parent: "epic"},
		{ID: "closed", Title: "Closed but blocked", Status: "completed", Type: "task", BlockedBy: []string{"blocker"}},
	} {
		b.Slug = bean.Slugify(b.Title)
		if err := core.Create(b); err != nil {
			t.Fatal(err)
		}
	}

	m := newListModel(&beangraph.CoreResolver{Core: core}, cfg)
	msg, ok := m.loadBeans().(beansLoadedMsg)
	if !ok {
		t.Fatalf("loadBeans did not return beansLoadedMsg")
	}

	var got []string
	for id := range msg.blocked {
		got = append(got, id)
	}
	slices.Sort(got)
	if want := []string{"child", "direct", "epic", "via-link"}; !slices.Equal(got, want) {
		t.Errorf("blocked = %v, want %v", got, want)
	}

	m, _ = m.Update(msg)
	for _, it := range m.list.Items() {
		item := it.(beanItem)
		if item.blocked != msg.blocked[item.bean.ID] {
			t.Errorf("item %s: blocked = %v, want %v", item.bean.ID, item.blocked, msg.blocked[item.bean.ID])
		}
	}
}
