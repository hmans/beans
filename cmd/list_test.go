package cmd

import (
	"testing"
	"time"

	"hmans.dev/beans/internal/bean"
	"hmans.dev/beans/internal/config"
	"hmans.dev/beans/internal/graph/model"
)

func TestSortBeans(t *testing.T) {
	now := time.Now()
	earlier := now.Add(-1 * time.Hour)
	evenEarlier := now.Add(-2 * time.Hour)

	// Statuses are now hardcoded, so we just use default config
	testCfg := config.Default()

	t.Run("sort by id", func(t *testing.T) {
		beans := []*bean.Bean{
			{ID: "c3"},
			{ID: "a1"},
			{ID: "b2"},
		}
		sortBeans(beans, "id", testCfg)

		if beans[0].ID != "a1" || beans[1].ID != "b2" || beans[2].ID != "c3" {
			t.Errorf("sort by id: got [%s, %s, %s], want [a1, b2, c3]",
				beans[0].ID, beans[1].ID, beans[2].ID)
		}
	})

	t.Run("sort by created", func(t *testing.T) {
		beans := []*bean.Bean{
			{ID: "old", CreatedAt: &evenEarlier},
			{ID: "new", CreatedAt: &now},
			{ID: "mid", CreatedAt: &earlier},
		}
		sortBeans(beans, "created", testCfg)

		// Should be newest first
		if beans[0].ID != "new" || beans[1].ID != "mid" || beans[2].ID != "old" {
			t.Errorf("sort by created: got [%s, %s, %s], want [new, mid, old]",
				beans[0].ID, beans[1].ID, beans[2].ID)
		}
	})

	t.Run("sort by created with nil", func(t *testing.T) {
		beans := []*bean.Bean{
			{ID: "nil1", CreatedAt: nil},
			{ID: "has", CreatedAt: &now},
			{ID: "nil2", CreatedAt: nil},
		}
		sortBeans(beans, "created", testCfg)

		// Non-nil should come first, then nil sorted by ID
		if beans[0].ID != "has" {
			t.Errorf("sort by created with nil: first should be \"has\", got %q", beans[0].ID)
		}
	})

	t.Run("sort by updated", func(t *testing.T) {
		beans := []*bean.Bean{
			{ID: "old", UpdatedAt: &evenEarlier},
			{ID: "new", UpdatedAt: &now},
			{ID: "mid", UpdatedAt: &earlier},
		}
		sortBeans(beans, "updated", testCfg)

		// Should be newest first
		if beans[0].ID != "new" || beans[1].ID != "mid" || beans[2].ID != "old" {
			t.Errorf("sort by updated: got [%s, %s, %s], want [new, mid, old]",
				beans[0].ID, beans[1].ID, beans[2].ID)
		}
	})

	t.Run("sort by status", func(t *testing.T) {
		beans := []*bean.Bean{
			{ID: "c1", Status: "completed"},
			{ID: "t1", Status: "todo"},
			{ID: "i1", Status: "in-progress"},
			{ID: "t2", Status: "todo"},
		}
		sortBeans(beans, "status", testCfg)

		// Should be ordered by status config order (not-ready, ready, in-progress, completed, scrapped), then by ID within same status
		expected := []string{"t1", "t2", "i1", "c1"}
		for i, want := range expected {
			if beans[i].ID != want {
				t.Errorf("sort by status[%d]: got %q, want %q", i, beans[i].ID, want)
			}
		}
	})

	t.Run("default sort (archive status then type)", func(t *testing.T) {
		beans := []*bean.Bean{
			{ID: "completed-bug", Status: "completed", Type: "bug"},
			{ID: "todo-feature", Status: "todo", Type: "feature"},
			{ID: "todo-task", Status: "todo", Type: "task"},
			{ID: "completed-task", Status: "completed", Type: "task"},
			{ID: "todo-bug", Status: "todo", Type: "bug"},
		}
		sortBeans(beans, "", testCfg)

		// Should be: non-archive first (sorted by type order from DefaultTypes: milestone, epic, bug, feature, task),
		// then archive (sorted by type)
		// DefaultTypes order: milestone, epic, bug, feature, task
		expected := []string{"todo-bug", "todo-feature", "todo-task", "completed-bug", "completed-task"}
		for i, want := range expected {
			if beans[i].ID != want {
				t.Errorf("default sort[%d]: got %q, want %q", i, beans[i].ID, want)
			}
		}
	})
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{"short string", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"needs truncation", "hello world", 8, "hello..."},
		{"very short max", "hello", 4, "h..."},
		{"empty string", "", 10, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestParseLinkFilters(t *testing.T) {
	tests := []struct {
		name    string
		input   []string
		want    []*model.LinkFilter
	}{
		{
			name:  "empty input",
			input: nil,
			want:  nil,
		},
		{
			name:  "type only",
			input: []string{"blocks"},
			want:  []*model.LinkFilter{{Type: "blocks"}},
		},
		{
			name:  "type with target",
			input: []string{"blocks:abc123"},
			want:  []*model.LinkFilter{{Type: "blocks", Target: strPtr("abc123")}},
		},
		{
			name:  "multiple filters",
			input: []string{"blocks", "parent:epic1"},
			want:  []*model.LinkFilter{{Type: "blocks"}, {Type: "parent", Target: strPtr("epic1")}},
		},
		{
			name:  "target with colons",
			input: []string{"blocks:id:with:colons"},
			want:  []*model.LinkFilter{{Type: "blocks", Target: strPtr("id:with:colons")}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseLinkFilters(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("parseLinkFilters() returned %d filters, want %d", len(got), len(tt.want))
				return
			}
			for i, f := range got {
				if f.Type != tt.want[i].Type {
					t.Errorf("parseLinkFilters()[%d].Type = %q, want %q", i, f.Type, tt.want[i].Type)
				}
				if (f.Target == nil) != (tt.want[i].Target == nil) {
					t.Errorf("parseLinkFilters()[%d].Target nil mismatch", i)
				} else if f.Target != nil && *f.Target != *tt.want[i].Target {
					t.Errorf("parseLinkFilters()[%d].Target = %q, want %q", i, *f.Target, *tt.want[i].Target)
				}
			}
		})
	}
}

// Helper function to create string pointer
func strPtr(s string) *string {
	return &s
}
