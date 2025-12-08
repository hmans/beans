package cmd

import (
	"testing"
	"time"

	"hmans.dev/beans/internal/bean"
	"hmans.dev/beans/internal/config"
)

// mockConfig implements the StatusNames interface for testing.
type mockConfig struct {
	statuses []string
	archive  map[string]bool
}

func (m *mockConfig) StatusNames() []string {
	return m.statuses
}

func (m *mockConfig) IsArchiveStatus(s string) bool {
	return m.archive[s]
}

func TestBuildRoadmap(t *testing.T) {
	// Save and restore global cfg
	oldCfg := cfg
	defer func() { cfg = oldCfg }()

	cfg = &config.Config{}
	cfg.Statuses = []config.StatusConfig{
		{Name: "in-progress"},
		{Name: "open"},
		{Name: "done", Archive: true},
	}

	now := time.Now()

	tests := []struct {
		name           string
		beans          []*bean.Bean
		includeDone    bool
		wantMilestones int
		wantUnscheduled int
	}{
		{
			name:           "empty beans",
			beans:          []*bean.Bean{},
			wantMilestones: 0,
		},
		{
			name: "milestone with epic and items",
			beans: []*bean.Bean{
				{ID: "m1", Type: "milestone", Title: "v1.0", Status: "open", CreatedAt: &now},
				{ID: "e1", Type: "epic", Title: "Auth", Status: "open", Links: bean.Links{{Type: "parent", Target: "m1"}}},
				{ID: "t1", Type: "task", Title: "Login", Status: "open", Links: bean.Links{{Type: "parent", Target: "e1"}}},
			},
			wantMilestones: 1,
		},
		{
			name: "milestone with direct children (no epic)",
			beans: []*bean.Bean{
				{ID: "m1", Type: "milestone", Title: "v1.0", Status: "open", CreatedAt: &now},
				{ID: "t1", Type: "task", Title: "Docs", Status: "open", Links: bean.Links{{Type: "parent", Target: "m1"}}},
			},
			wantMilestones: 1,
		},
		{
			name: "unscheduled epic",
			beans: []*bean.Bean{
				{ID: "e1", Type: "epic", Title: "Future", Status: "open"},
				{ID: "t1", Type: "task", Title: "Nice to have", Status: "open", Links: bean.Links{{Type: "parent", Target: "e1"}}},
			},
			wantMilestones:  0,
			wantUnscheduled: 1,
		},
		{
			name: "done items excluded by default",
			beans: []*bean.Bean{
				{ID: "m1", Type: "milestone", Title: "v1.0", Status: "open", CreatedAt: &now},
				{ID: "t1", Type: "task", Title: "Done task", Status: "done", Links: bean.Links{{Type: "parent", Target: "m1"}}},
			},
			includeDone:    false,
			wantMilestones: 0, // milestone has no visible children
		},
		{
			name: "done items included when requested",
			beans: []*bean.Bean{
				{ID: "m1", Type: "milestone", Title: "v1.0", Status: "open", CreatedAt: &now},
				{ID: "t1", Type: "task", Title: "Done task", Status: "done", Links: bean.Links{{Type: "parent", Target: "m1"}}},
			},
			includeDone:    true,
			wantMilestones: 1,
		},
		{
			name: "bean without parent not included",
			beans: []*bean.Bean{
				{ID: "m1", Type: "milestone", Title: "v1.0", Status: "open", CreatedAt: &now},
				{ID: "t1", Type: "task", Title: "Orphan", Status: "open"}, // no parent link
			},
			wantMilestones: 0, // milestone has no children
		},
		{
			name: "item with both epic and milestone parent appears once under epic",
			beans: []*bean.Bean{
				{ID: "m1", Type: "milestone", Title: "v1.0", Status: "open", CreatedAt: &now},
				{ID: "e1", Type: "epic", Title: "Auth", Status: "open", Links: bean.Links{{Type: "parent", Target: "m1"}}},
				{ID: "t1", Type: "task", Title: "Login", Status: "open", Links: bean.Links{
					{Type: "parent", Target: "e1"},
					{Type: "parent", Target: "m1"},
				}},
			},
			wantMilestones: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildRoadmap(tt.beans, tt.includeDone, nil, nil)

			if got := len(result.Milestones); got != tt.wantMilestones {
				t.Errorf("got %d milestones, want %d", got, tt.wantMilestones)
			}

			if got := len(result.Unscheduled); got != tt.wantUnscheduled {
				t.Errorf("got %d unscheduled, want %d", got, tt.wantUnscheduled)
			}
		})
	}
}

func TestBuildRoadmap_DuplicateAvoidance(t *testing.T) {
	// Verify that an item with both epic and milestone parent appears only under epic
	oldCfg := cfg
	defer func() { cfg = oldCfg }()

	cfg = &config.Config{}
	cfg.Statuses = []config.StatusConfig{
		{Name: "open"},
		{Name: "done", Archive: true},
	}

	now := time.Now()
	beans := []*bean.Bean{
		{ID: "m1", Type: "milestone", Title: "v1.0", Status: "open", CreatedAt: &now},
		{ID: "e1", Type: "epic", Title: "Auth", Status: "open", Links: bean.Links{{Type: "parent", Target: "m1"}}},
		{ID: "t1", Type: "task", Title: "Login", Status: "open", Links: bean.Links{
			{Type: "parent", Target: "e1"},
			{Type: "parent", Target: "m1"},
		}},
	}

	result := buildRoadmap(beans, false, nil, nil)

	if len(result.Milestones) != 1 {
		t.Fatalf("expected 1 milestone, got %d", len(result.Milestones))
	}

	mg := result.Milestones[0]
	if len(mg.Epics) != 1 {
		t.Fatalf("expected 1 epic, got %d", len(mg.Epics))
	}

	if len(mg.Epics[0].Items) != 1 {
		t.Errorf("expected 1 item under epic, got %d", len(mg.Epics[0].Items))
	}

	if len(mg.Other) != 0 {
		t.Errorf("expected 0 items in Other, got %d", len(mg.Other))
	}
}

func TestFirstParagraph(t *testing.T) {
	tests := []struct {
		name  string
		body  string
		want  string
	}{
		{
			name: "empty body",
			body: "",
			want: "",
		},
		{
			name: "single line",
			body: "This is a description.",
			want: "This is a description.",
		},
		{
			name: "multiple paragraphs",
			body: "First paragraph.\n\nSecond paragraph.",
			want: "First paragraph.",
		},
		{
			name: "multiline first paragraph",
			body: "Line one\nLine two\n\nSecond para.",
			want: "Line one Line two",
		},
		{
			name: "skips headers at start",
			body: "## Checklist\n- item one",
			want: "- item one",
		},
		{
			name: "truncates long text",
			body: "This is a very long paragraph that exceeds two hundred characters and needs to be truncated so it does not take up too much space in the roadmap output. Lorem ipsum dolor sit amet consectetur adipiscing elit.",
			want: "This is a very long paragraph that exceeds two hundred characters and needs to be truncated so it does not take up too much space in the roadmap output. Lorem ipsum dolor sit amet consectetur adipi...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := firstParagraph(tt.body)
			if got != tt.want {
				t.Errorf("firstParagraph() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderItem(t *testing.T) {
	tests := []struct {
		name       string
		bean       *bean.Bean
		asLink     bool
		linkPrefix string
		want       string
	}{
		{
			name:   "no links",
			bean:   &bean.Bean{ID: "abc", Path: "abc--do-something.md", Type: "task", Title: "Do something", Status: "open"},
			asLink: false,
			want:   "- [#abc] Do something\n",
		},
		{
			name:       "with links, no prefix",
			bean:       &bean.Bean{ID: "abc", Path: "abc--do-something.md", Type: "task", Title: "Do something", Status: "open"},
			asLink:     true,
			linkPrefix: "",
			want:       "- [[#abc](abc--do-something.md)] Do something\n",
		},
		{
			name:       "with links and prefix",
			bean:       &bean.Bean{ID: "def", Path: "def--fix-issue.md", Type: "bug", Title: "Fix issue", Status: "open"},
			asLink:     true,
			linkPrefix: "https://github.com/user/repo/blob/main/.beans/",
			want:       "- [[#def](https://github.com/user/repo/blob/main/.beans/def--fix-issue.md)] Fix issue\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderItem(tt.bean, tt.asLink, tt.linkPrefix)
			if got != tt.want {
				t.Errorf("renderItem() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderBeanRef(t *testing.T) {
	tests := []struct {
		name       string
		bean       *bean.Bean
		asLink     bool
		linkPrefix string
		want       string
	}{
		{
			name:   "no link - just ID",
			bean:   &bean.Bean{ID: "abc", Path: "abc--milestone.md"},
			asLink: false,
			want:   "[#abc]",
		},
		{
			name:       "link without prefix",
			bean:       &bean.Bean{ID: "abc", Path: "abc--milestone.md"},
			asLink:     true,
			linkPrefix: "",
			want:       "[[#abc](abc--milestone.md)]",
		},
		{
			name:       "link with prefix",
			bean:       &bean.Bean{ID: "abc", Path: "abc--milestone.md"},
			asLink:     true,
			linkPrefix: "https://example.com/beans/",
			want:       "[[#abc](https://example.com/beans/abc--milestone.md)]",
		},
		{
			name:       "link with prefix without trailing slash",
			bean:       &bean.Bean{ID: "abc", Path: "abc--milestone.md"},
			asLink:     true,
			linkPrefix: ".beans",
			want:       "[[#abc](.beans/abc--milestone.md)]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderBeanRef(tt.bean, tt.asLink, tt.linkPrefix)
			if got != tt.want {
				t.Errorf("renderBeanRef() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStatusFiltering(t *testing.T) {
	oldCfg := cfg
	defer func() { cfg = oldCfg }()

	cfg = &config.Config{}
	cfg.Statuses = []config.StatusConfig{
		{Name: "open"},
		{Name: "in-progress"},
		{Name: "done", Archive: true},
	}

	now := time.Now()
	beans := []*bean.Bean{
		{ID: "m1", Type: "milestone", Title: "Open Milestone", Status: "open", CreatedAt: &now},
		{ID: "m2", Type: "milestone", Title: "In Progress Milestone", Status: "in-progress", CreatedAt: &now},
		{ID: "t1", Type: "task", Title: "Task 1", Status: "open", Links: bean.Links{{Type: "parent", Target: "m1"}}},
		{ID: "t2", Type: "task", Title: "Task 2", Status: "open", Links: bean.Links{{Type: "parent", Target: "m2"}}},
	}

	t.Run("filter by status", func(t *testing.T) {
		result := buildRoadmap(beans, false, []string{"open"}, nil)
		if len(result.Milestones) != 1 {
			t.Errorf("expected 1 milestone, got %d", len(result.Milestones))
		}
		if result.Milestones[0].Milestone.Status != "open" {
			t.Errorf("expected open milestone, got %s", result.Milestones[0].Milestone.Status)
		}
	})

	t.Run("exclude by status", func(t *testing.T) {
		result := buildRoadmap(beans, false, nil, []string{"in-progress"})
		if len(result.Milestones) != 1 {
			t.Errorf("expected 1 milestone, got %d", len(result.Milestones))
		}
		if result.Milestones[0].Milestone.Status != "open" {
			t.Errorf("expected open milestone, got %s", result.Milestones[0].Milestone.Status)
		}
	})
}
