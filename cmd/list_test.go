package cmd

import (
	"testing"
	"time"

	"hmans.dev/beans/internal/bean"
)

func TestFilterBeans(t *testing.T) {
	// Create test beans
	beans := []*bean.Bean{
		{ID: "a1", Status: "open"},
		{ID: "b2", Status: "in-progress"},
		{ID: "c3", Status: "done"},
		{ID: "d4", Status: "open"},
		{ID: "e5", Status: "in-progress"},
	}

	tests := []struct {
		name      string
		statuses  []string
		wantCount int
		wantIDs   []string
	}{
		{
			name:      "no filter",
			statuses:  nil,
			wantCount: 5,
		},
		{
			name:      "empty filter",
			statuses:  []string{},
			wantCount: 5,
		},
		{
			name:      "filter open",
			statuses:  []string{"open"},
			wantCount: 2,
			wantIDs:   []string{"a1", "d4"},
		},
		{
			name:      "filter in-progress",
			statuses:  []string{"in-progress"},
			wantCount: 2,
			wantIDs:   []string{"b2", "e5"},
		},
		{
			name:      "filter done",
			statuses:  []string{"done"},
			wantCount: 1,
			wantIDs:   []string{"c3"},
		},
		{
			name:      "multiple statuses",
			statuses:  []string{"open", "done"},
			wantCount: 3,
			wantIDs:   []string{"a1", "c3", "d4"},
		},
		{
			name:      "non-existent status",
			statuses:  []string{"invalid"},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterBeans(beans, tt.statuses)

			if len(got) != tt.wantCount {
				t.Errorf("filterBeans() count = %d, want %d", len(got), tt.wantCount)
			}

			if tt.wantIDs != nil {
				gotIDs := make([]string, len(got))
				for i, b := range got {
					gotIDs[i] = b.ID
				}
				for _, wantID := range tt.wantIDs {
					found := false
					for _, gotID := range gotIDs {
						if gotID == wantID {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("filterBeans() missing expected ID %q", wantID)
					}
				}
			}
		})
	}
}

func TestSortBeans(t *testing.T) {
	now := time.Now()
	earlier := now.Add(-1 * time.Hour)
	evenEarlier := now.Add(-2 * time.Hour)

	statusNames := []string{"open", "in-progress", "done"}

	t.Run("sort by id", func(t *testing.T) {
		beans := []*bean.Bean{
			{ID: "c3"},
			{ID: "a1"},
			{ID: "b2"},
		}
		sortBeans(beans, "id", statusNames)

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
		sortBeans(beans, "created", statusNames)

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
		sortBeans(beans, "created", statusNames)

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
		sortBeans(beans, "updated", statusNames)

		// Should be newest first
		if beans[0].ID != "new" || beans[1].ID != "mid" || beans[2].ID != "old" {
			t.Errorf("sort by updated: got [%s, %s, %s], want [new, mid, old]",
				beans[0].ID, beans[1].ID, beans[2].ID)
		}
	})

	t.Run("sort by status", func(t *testing.T) {
		beans := []*bean.Bean{
			{ID: "d1", Status: "done"},
			{ID: "o1", Status: "open"},
			{ID: "i1", Status: "in-progress"},
			{ID: "o2", Status: "open"},
		}
		sortBeans(beans, "status", statusNames)

		// Should be ordered by status config order, then by ID within same status
		expected := []string{"o1", "o2", "i1", "d1"}
		for i, want := range expected {
			if beans[i].ID != want {
				t.Errorf("sort by status[%d]: got %q, want %q", i, beans[i].ID, want)
			}
		}
	})

	t.Run("default sort (id)", func(t *testing.T) {
		beans := []*bean.Bean{
			{ID: "c3"},
			{ID: "a1"},
			{ID: "b2"},
		}
		sortBeans(beans, "unknown", statusNames)

		if beans[0].ID != "a1" || beans[1].ID != "b2" || beans[2].ID != "c3" {
			t.Errorf("default sort: got [%s, %s, %s], want [a1, b2, c3]",
				beans[0].ID, beans[1].ID, beans[2].ID)
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

func TestFilterByLinks(t *testing.T) {
	// Create test beans with various link configurations
	beans := []*bean.Bean{
		{ID: "a1", Links: bean.Links{{Type: "blocks", Target: "b2"}}},
		{ID: "b2", Links: bean.Links{{Type: "parent", Target: "epic1"}}},
		{ID: "c3", Links: bean.Links{{Type: "blocks", Target: "a1"}, {Type: "blocks", Target: "b2"}}},
		{ID: "d4", Links: nil}, // no links
		{ID: "e5", Links: bean.Links{{Type: "blocks", Target: "b2"}, {Type: "parent", Target: "epic1"}}},
	}

	tests := []struct {
		name    string
		filter  []string
		wantIDs []string
	}{
		{
			name:    "no filter",
			filter:  nil,
			wantIDs: []string{"a1", "b2", "c3", "d4", "e5"},
		},
		{
			name:    "filter by type only - blocks",
			filter:  []string{"blocks"},
			wantIDs: []string{"a1", "c3", "e5"},
		},
		{
			name:    "filter by type only - parent",
			filter:  []string{"parent"},
			wantIDs: []string{"b2", "e5"},
		},
		{
			name:    "filter by type:id - blocks:b2",
			filter:  []string{"blocks:b2"},
			wantIDs: []string{"a1", "c3", "e5"},
		},
		{
			name:    "filter by type:id - blocks:a1",
			filter:  []string{"blocks:a1"},
			wantIDs: []string{"c3"},
		},
		{
			name:    "multiple filters (OR logic)",
			filter:  []string{"blocks", "parent"},
			wantIDs: []string{"a1", "b2", "c3", "e5"},
		},
		{
			name:    "non-existent link type",
			filter:  []string{"nonexistent"},
			wantIDs: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterByLinks(beans, parseLinkFilters(tt.filter))
			gotIDs := extractIDs(got)

			if !equalStringSlices(gotIDs, tt.wantIDs) {
				t.Errorf("filterByLinks() = %v, want %v", gotIDs, tt.wantIDs)
			}
		})
	}
}

func TestFilterByLinkedAs(t *testing.T) {
	// Create test beans where some beans link to others
	beans := []*bean.Bean{
		{ID: "a1", Links: bean.Links{{Type: "blocks", Target: "b2"}}},
		{ID: "b2", Links: bean.Links{{Type: "blocks", Target: "c3"}}},
		{ID: "c3", Links: nil},
		{ID: "epic1", Links: bean.Links{{Type: "parent", Target: "a1"}, {Type: "parent", Target: "b2"}}}, // epic1 is parent of a1 and b2
	}

	// Build index once for all tests
	idx := buildLinkIndex(beans)

	tests := []struct {
		name    string
		filter  []string
		wantIDs []string
	}{
		{
			name:    "no filter",
			filter:  nil,
			wantIDs: []string{"a1", "b2", "c3", "epic1"},
		},
		{
			name:    "filter by type only - blocks (beans that are blocked)",
			filter:  []string{"blocks"},
			wantIDs: []string{"b2", "c3"}, // b2 is blocked by a1, c3 is blocked by b2
		},
		{
			name:    "filter by type:id - blocks:a1 (beans that a1 blocks)",
			filter:  []string{"blocks:a1"},
			wantIDs: []string{"b2"},
		},
		{
			name:    "filter by type only - parent (beans that have a parent)",
			filter:  []string{"parent"},
			wantIDs: []string{"a1", "b2"}, // epic1 is parent of a1 and b2
		},
		{
			name:    "non-existent source bean",
			filter:  []string{"blocks:nonexistent"},
			wantIDs: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterByLinkedAs(beans, parseLinkFilters(tt.filter), idx)
			gotIDs := extractIDs(got)

			if !equalStringSlices(gotIDs, tt.wantIDs) {
				t.Errorf("filterByLinkedAs() = %v, want %v", gotIDs, tt.wantIDs)
			}
		})
	}
}

func TestExcludeByLinks(t *testing.T) {
	// Create test beans with various link configurations
	beans := []*bean.Bean{
		{ID: "a1", Links: bean.Links{{Type: "blocks", Target: "b2"}}},
		{ID: "b2", Links: bean.Links{{Type: "parent", Target: "epic1"}}},
		{ID: "c3", Links: bean.Links{{Type: "blocks", Target: "a1"}, {Type: "blocks", Target: "b2"}}},
		{ID: "d4", Links: nil}, // no links
		{ID: "e5", Links: bean.Links{{Type: "blocks", Target: "b2"}, {Type: "parent", Target: "epic1"}}},
	}

	tests := []struct {
		name    string
		exclude []string
		wantIDs []string
	}{
		{
			name:    "no exclusion",
			exclude: nil,
			wantIDs: []string{"a1", "b2", "c3", "d4", "e5"},
		},
		{
			name:    "exclude by type - blocks (exclude beans that block something)",
			exclude: []string{"blocks"},
			wantIDs: []string{"b2", "d4"}, // only b2 and d4 don't have blocks links
		},
		{
			name:    "exclude by type - parent",
			exclude: []string{"parent"},
			wantIDs: []string{"a1", "c3", "d4"}, // these don't have parent links
		},
		{
			name:    "exclude by type:id - blocks:b2",
			exclude: []string{"blocks:b2"},
			wantIDs: []string{"b2", "d4"}, // a1, c3, e5 all block b2
		},
		{
			name:    "multiple exclusions",
			exclude: []string{"blocks", "parent"},
			wantIDs: []string{"d4"}, // only d4 has neither blocks nor parent
		},
		{
			name:    "non-existent link type",
			exclude: []string{"nonexistent"},
			wantIDs: []string{"a1", "b2", "c3", "d4", "e5"}, // nothing excluded
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := excludeByLinks(beans, parseLinkFilters(tt.exclude))
			gotIDs := extractIDs(got)

			if !equalStringSlices(gotIDs, tt.wantIDs) {
				t.Errorf("excludeByLinks() = %v, want %v", gotIDs, tt.wantIDs)
			}
		})
	}
}

func TestExcludeByLinkedAs(t *testing.T) {
	// Create test beans where some beans link to others
	beans := []*bean.Bean{
		{ID: "a1", Links: bean.Links{{Type: "blocks", Target: "b2"}}},
		{ID: "b2", Links: bean.Links{{Type: "blocks", Target: "c3"}}},
		{ID: "c3", Links: nil},
		{ID: "d4", Links: nil},
		{ID: "epic1", Links: bean.Links{{Type: "parent", Target: "a1"}, {Type: "parent", Target: "b2"}}},
	}

	// Build index once for all tests
	idx := buildLinkIndex(beans)

	tests := []struct {
		name    string
		exclude []string
		wantIDs []string
	}{
		{
			name:    "no exclusion",
			exclude: nil,
			wantIDs: []string{"a1", "b2", "c3", "d4", "epic1"},
		},
		{
			name:    "exclude blocked beans (actionable work)",
			exclude: []string{"blocks"},
			wantIDs: []string{"a1", "d4", "epic1"}, // b2 and c3 are blocked
		},
		{
			name:    "exclude by type:id - blocks:a1 (exclude beans blocked by a1)",
			exclude: []string{"blocks:a1"},
			wantIDs: []string{"a1", "c3", "d4", "epic1"}, // only b2 is blocked by a1
		},
		{
			name:    "exclude beans with parent",
			exclude: []string{"parent"},
			wantIDs: []string{"c3", "d4", "epic1"}, // a1 and b2 have epic1 as parent
		},
		{
			name:    "non-existent source bean",
			exclude: []string{"blocks:nonexistent"},
			wantIDs: []string{"a1", "b2", "c3", "d4", "epic1"}, // nothing excluded
		},
		{
			name:    "multiple exclusions",
			exclude: []string{"blocks", "parent"},
			wantIDs: []string{"d4", "epic1"}, // d4 and epic1 are neither blocked nor children
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := excludeByLinkedAs(beans, parseLinkFilters(tt.exclude), idx)
			gotIDs := extractIDs(got)

			if !equalStringSlices(gotIDs, tt.wantIDs) {
				t.Errorf("excludeByLinkedAs() = %v, want %v", gotIDs, tt.wantIDs)
			}
		})
	}
}

// Helper function to extract IDs from beans slice
func extractIDs(beans []*bean.Bean) []string {
	ids := make([]string, len(beans))
	for i, b := range beans {
		ids[i] = b.ID
	}
	return ids
}

// Helper function to compare string slices (order-independent)
func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	aMap := make(map[string]int)
	for _, s := range a {
		aMap[s]++
	}
	for _, s := range b {
		aMap[s]--
		if aMap[s] < 0 {
			return false
		}
	}
	return true
}
