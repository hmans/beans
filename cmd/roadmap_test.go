package cmd

import (
	"testing"
	"time"

	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/config"
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

	// Statuses are now hardcoded
	cfg = config.Default()

	now := time.Now()

	tests := []struct {
		name                  string
		beans                 []*bean.Bean
		includeDone           bool
		wantMilestones        int
		wantUnscheduledEpics  int
		wantUnscheduledOther  int
	}{
		{
			name:           "empty beans",
			beans:          []*bean.Bean{},
			wantMilestones: 0,
		},
		{
			name: "milestone with epic and items",
			beans: []*bean.Bean{
				{ID: "m1", Type: "milestone", Title: "v1.0", Status: "todo", CreatedAt: &now},
				{ID: "e1", Type: "epic", Title: "Auth", Status: "todo", Milestone: "m1"},
				{ID: "t1", Type: "task", Title: "Login", Status: "todo", Epic: "e1"},
			},
			wantMilestones: 1,
		},
		{
			name: "milestone with direct children (no epic)",
			beans: []*bean.Bean{
				{ID: "m1", Type: "milestone", Title: "v1.0", Status: "todo", CreatedAt: &now},
				{ID: "t1", Type: "task", Title: "Docs", Status: "todo", Milestone: "m1"},
			},
			wantMilestones: 1,
		},
		{
			name: "unscheduled epic",
			beans: []*bean.Bean{
				{ID: "e1", Type: "epic", Title: "Future", Status: "todo"},
				{ID: "t1", Type: "task", Title: "Nice to have", Status: "todo", Epic: "e1"},
			},
			wantMilestones:       0,
			wantUnscheduledEpics: 1,
		},
		{
			name: "done items excluded by default",
			beans: []*bean.Bean{
				{ID: "m1", Type: "milestone", Title: "v1.0", Status: "todo", CreatedAt: &now},
				{ID: "t1", Type: "task", Title: "Done task", Status: "completed", Milestone: "m1"},
			},
			includeDone:    false,
			wantMilestones: 0, // milestone has no visible children
		},
		{
			name: "done items included when requested",
			beans: []*bean.Bean{
				{ID: "m1", Type: "milestone", Title: "v1.0", Status: "todo", CreatedAt: &now},
				{ID: "t1", Type: "task", Title: "Done task", Status: "completed", Milestone: "m1"},
			},
			includeDone:    true,
			wantMilestones: 1,
		},
		{
			name: "orphan bean appears in unscheduled other",
			beans: []*bean.Bean{
				{ID: "m1", Type: "milestone", Title: "v1.0", Status: "todo", CreatedAt: &now},
				{ID: "t1", Type: "task", Title: "Orphan", Status: "todo"}, // no parent link
			},
			wantMilestones:       0, // milestone has no children
			wantUnscheduledOther: 1, // orphan appears in unscheduled
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildRoadmap(tt.beans, tt.includeDone, nil, nil)

			if got := len(result.Milestones); got != tt.wantMilestones {
				t.Errorf("got %d milestones, want %d", got, tt.wantMilestones)
			}

			gotUnscheduledEpics := 0
			gotUnscheduledOther := 0
			if result.Unscheduled != nil {
				gotUnscheduledEpics = len(result.Unscheduled.Epics)
				gotUnscheduledOther = len(result.Unscheduled.Other)
			}
			if gotUnscheduledEpics != tt.wantUnscheduledEpics {
				t.Errorf("got %d unscheduled epics, want %d", gotUnscheduledEpics, tt.wantUnscheduledEpics)
			}
			if gotUnscheduledOther != tt.wantUnscheduledOther {
				t.Errorf("got %d unscheduled other, want %d", gotUnscheduledOther, tt.wantUnscheduledOther)
			}
		})
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
			want:   "(abc)",
		},
		{
			name:       "link without prefix",
			bean:       &bean.Bean{ID: "abc", Path: "abc--milestone.md"},
			asLink:     true,
			linkPrefix: "",
			want:       "([abc](abc--milestone.md))",
		},
		{
			name:       "link with prefix",
			bean:       &bean.Bean{ID: "abc", Path: "abc--milestone.md"},
			asLink:     true,
			linkPrefix: "https://example.com/beans/",
			want:       "([abc](https://example.com/beans/abc--milestone.md))",
		},
		{
			name:       "link with prefix without trailing slash",
			bean:       &bean.Bean{ID: "abc", Path: "abc--milestone.md"},
			asLink:     true,
			linkPrefix: ".beans",
			want:       "([abc](.beans/abc--milestone.md))",
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

	// Statuses are now hardcoded
	cfg = config.Default()

	now := time.Now()
	beans := []*bean.Bean{
		{ID: "m1", Type: "milestone", Title: "Todo Milestone", Status: "todo", CreatedAt: &now},
		{ID: "m2", Type: "milestone", Title: "In Progress Milestone", Status: "in-progress", CreatedAt: &now},
		{ID: "t1", Type: "task", Title: "Task 1", Status: "todo", Milestone: "m1"},
		{ID: "t2", Type: "task", Title: "Task 2", Status: "todo", Milestone: "m2"},
	}

	t.Run("filter by status", func(t *testing.T) {
		result := buildRoadmap(beans, false, []string{"todo"}, nil)
		if len(result.Milestones) != 1 {
			t.Errorf("expected 1 milestone, got %d", len(result.Milestones))
		}
		if result.Milestones[0].Milestone.Status != "todo" {
			t.Errorf("expected todo milestone, got %s", result.Milestones[0].Milestone.Status)
		}
	})

	t.Run("exclude by status", func(t *testing.T) {
		result := buildRoadmap(beans, false, nil, []string{"in-progress"})
		if len(result.Milestones) != 1 {
			t.Errorf("expected 1 milestone, got %d", len(result.Milestones))
		}
		if result.Milestones[0].Milestone.Status != "todo" {
			t.Errorf("expected todo milestone, got %s", result.Milestones[0].Milestone.Status)
		}
	})
}
