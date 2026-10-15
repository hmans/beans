package bean

import (
	"slices"
	"testing"
	"time"
)

var (
	testPriorities = []string{"critical", "high", "normal", "low", "deferred"}
	testTypes      = []string{"milestone", "epic", "bug", "feature", "task"}
)

func at(day int) *time.Time {
	t := time.Date(2026, 1, day, 0, 0, 0, 0, time.UTC)
	return &t
}

func rowIDs(rows []NextRow) []string {
	ids := make([]string, len(rows))
	for i, r := range rows {
		ids[i] = r.Bean.ID
	}
	return ids
}

func rowLanes(rows []NextRow) []NextLane {
	lanes := make([]NextLane, len(rows))
	for i, r := range rows {
		lanes[i] = r.Lane
	}
	return lanes
}

// rank uses every non-closed, non-in-progress bean as a candidate, standing
// in for the --ready set plus drafts.
func rank(all []*Bean, focus *Bean) NextResult {
	var candidates []*Bean
	for _, b := range all {
		if !isClosedStatus(b.Status) && b.Status != "in-progress" {
			candidates = append(candidates, b)
		}
	}
	return RankNext(candidates, all, NextOptions{Focus: focus, PriorityNames: testPriorities, TypeNames: testTypes})
}

func TestRankNextLanes(t *testing.T) {
	epic := &Bean{ID: "e1", Type: "epic", Status: "in-progress"}
	idle := &Bean{ID: "e2", Type: "epic", Status: "todo"}
	all := []*Bean{
		epic, idle,
		{ID: "bug-high-front", Type: "bug", Status: "todo", Priority: "high", Parent: "e1"},
		{ID: "bug-crit-nofront", Type: "bug", Status: "todo", Priority: "critical", Parent: "e2"},
		{ID: "bug-normal-front", Type: "bug", Status: "todo", Priority: "normal", Parent: "e1"},
		{ID: "task-front", Type: "task", Status: "todo", Parent: "e1"},
		{ID: "draft-front", Type: "task", Status: "draft", Parent: "e1"},
		{ID: "draft-bug-high", Type: "bug", Status: "draft", Priority: "high", Parent: "e1"},
		{ID: "task-nofront", Type: "task", Status: "todo"},
		{ID: "draft-nofront", Type: "task", Status: "draft"},
	}

	got := rank(all, nil)

	wantIDs := []string{
		"bug-crit-nofront", "bug-high-front",
		"bug-normal-front", "task-front",
		"draft-bug-high", "draft-front",
		"draft-nofront", "task-nofront",
	}
	wantLanes := []NextLane{
		LaneInterrupt, LaneInterrupt,
		LaneFocus, LaneFocus,
		LaneRefine, LaneRefine,
		LaneRest, LaneRest,
	}
	if ids := rowIDs(got.Rows); !slices.Equal(ids, wantIDs) {
		t.Errorf("order = %v, want %v", ids, wantIDs)
	}
	if lanes := rowLanes(got.Rows); !slices.Equal(lanes, wantLanes) {
		t.Errorf("lanes = %v, want %v", lanes, wantLanes)
	}
	if len(got.Fronts) != 1 || got.Fronts[0].ID != "e1" {
		t.Errorf("fronts = %v, want [e1]", got.Fronts)
	}
}

func TestRankNextCandidateExclusion(t *testing.T) {
	tests := []struct {
		name string
		all  []*Bean
		want []string
	}{
		{
			name: "milestones and epics are never rows",
			all: []*Bean{
				{ID: "m", Type: "milestone", Status: "todo"},
				{ID: "e", Type: "epic", Status: "todo"},
				{ID: "t", Type: "task", Status: "todo"},
			},
			want: []string{"t"},
		},
		{
			name: "parent with todo children is replaced by them",
			all: []*Bean{
				{ID: "a", Type: "feature", Status: "todo"},
				{ID: "b", Type: "task", Status: "todo", Parent: "a", CreatedAt: at(1)},
				{ID: "c", Type: "task", Status: "todo", Parent: "a", CreatedAt: at(2)},
			},
			want: []string{"b", "c"},
		},
		{
			name: "parent with only closed children is a row",
			all: []*Bean{
				{ID: "a", Type: "feature", Status: "todo"},
				{ID: "b", Type: "task", Status: "completed", Parent: "a"},
				{ID: "c", Type: "task", Status: "scrapped", Parent: "a"},
			},
			want: []string{"a"},
		},
		{
			name: "draft child keeps its parent hidden and is itself a row",
			all: []*Bean{
				{ID: "a", Type: "feature", Status: "todo"},
				{ID: "b", Type: "task", Status: "completed", Parent: "a"},
				{ID: "d", Type: "task", Status: "draft", Parent: "a"},
			},
			want: []string{"d"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := rowIDs(rank(tt.all, nil).Rows); !slices.Equal(got, tt.want) {
				t.Errorf("rows = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRankNextFronts(t *testing.T) {
	milestone := &Bean{ID: "m", Type: "milestone", Status: "in-progress"}
	epic := &Bean{ID: "e", Type: "epic", Status: "in-progress", Parent: "m"}
	other := &Bean{ID: "o", Type: "epic", Status: "in-progress"}
	all := []*Bean{
		milestone, epic, other,
		{ID: "under-epic", Type: "task", Status: "todo", Parent: "e"},
		{ID: "under-milestone", Type: "task", Status: "todo", Parent: "m"},
		{ID: "under-other", Type: "task", Status: "todo", Parent: "o"},
	}

	t.Run("nearest in-progress front decides", func(t *testing.T) {
		got := rank(all, nil)
		fronts := map[string]string{}
		for _, r := range got.Rows {
			fronts[r.Bean.ID] = r.Front.ID
		}
		want := map[string]string{"under-epic": "e", "under-milestone": "m", "under-other": "o"}
		for id, f := range want {
			if fronts[id] != f {
				t.Errorf("%s: front = %q, want %q", id, fronts[id], f)
			}
		}
		if len(got.Fronts) != 3 {
			t.Errorf("got %d fronts, want 3", len(got.Fronts))
		}
	})

	t.Run("explicit focus replaces in-progress fronts", func(t *testing.T) {
		got := rank(all, milestone)
		lanes := map[string]NextLane{}
		for _, r := range got.Rows {
			lanes[r.Bean.ID] = r.Lane
		}
		want := map[string]NextLane{"under-epic": LaneFocus, "under-milestone": LaneFocus, "under-other": LaneRest}
		for id, l := range want {
			if lanes[id] != l {
				t.Errorf("%s: lane = %q, want %q", id, lanes[id], l)
			}
		}
		if len(got.Fronts) != 1 || got.Fronts[0].ID != "m" {
			t.Errorf("fronts = %v, want [m]", got.Fronts)
		}
	})
}

func TestRankNextOrdering(t *testing.T) {
	tests := []struct {
		name string
		all  []*Bean
		want []string
	}{
		{
			name: "front priority beats bean priority",
			all: []*Bean{
				{ID: "fa", Type: "epic", Status: "in-progress", Priority: "normal"},
				{ID: "fb", Type: "epic", Status: "in-progress", Priority: "high"},
				{ID: "a-high", Type: "task", Status: "todo", Priority: "high", Parent: "fa"},
				{ID: "b-low", Type: "task", Status: "todo", Priority: "low", Parent: "fb"},
			},
			want: []string{"b-low", "a-high"},
		},
		{
			name: "front manual order, then front updated_at newest first",
			all: []*Bean{
				{ID: "fa", Type: "epic", Status: "in-progress", UpdatedAt: at(9)},
				{ID: "fb", Type: "epic", Status: "in-progress", Order: "V", UpdatedAt: at(1)},
				{ID: "fc", Type: "epic", Status: "in-progress", UpdatedAt: at(5)},
				{ID: "a", Type: "task", Status: "todo", Parent: "fa"},
				{ID: "b", Type: "task", Status: "todo", Parent: "fb"},
				{ID: "c", Type: "task", Status: "todo", Parent: "fc"},
			},
			want: []string{"b", "a", "c"},
		},
		{
			name: "within a front: priority, type, order, created_at",
			all: []*Bean{
				{ID: "f", Type: "epic", Status: "in-progress"},
				{ID: "task-old", Type: "task", Status: "todo", Parent: "f", CreatedAt: at(1)},
				{ID: "task-new", Type: "task", Status: "todo", Parent: "f", CreatedAt: at(2)},
				{ID: "task-ordered", Type: "task", Status: "todo", Parent: "f", Order: "V", CreatedAt: at(3)},
				{ID: "feature", Type: "feature", Status: "todo", Parent: "f"},
				{ID: "bug-normal", Type: "bug", Status: "todo", Parent: "f"},
				{ID: "task-high", Type: "task", Status: "todo", Priority: "high", Parent: "f"},
				{ID: "task-unset", Type: "task", Status: "todo", Priority: "", Parent: "f", CreatedAt: at(4)},
			},
			want: []string{"task-high", "bug-normal", "feature", "task-ordered", "task-old", "task-new", "task-unset"},
		},
		{
			name: "interrupt: bean priority first, then front",
			all: []*Bean{
				{ID: "fa", Type: "epic", Status: "in-progress", Priority: "high"},
				{ID: "fb", Type: "epic", Status: "in-progress", Priority: "low"},
				{ID: "high-in-a", Type: "bug", Status: "todo", Priority: "high", Parent: "fa"},
				{ID: "crit-nofront", Type: "bug", Status: "todo", Priority: "critical"},
				{ID: "high-nofront", Type: "bug", Status: "todo", Priority: "high"},
				{ID: "high-in-b", Type: "bug", Status: "todo", Priority: "high", Parent: "fb"},
			},
			want: []string{"crit-nofront", "high-in-a", "high-in-b", "high-nofront"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := rowIDs(rank(tt.all, nil).Rows); !slices.Equal(got, tt.want) {
				t.Errorf("rows = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRankNextInProgress(t *testing.T) {
	front := &Bean{ID: "f", Type: "epic", Status: "in-progress"}
	all := []*Bean{
		front,
		{ID: "a", Type: "task", Status: "in-progress", Parent: "f"},
		{ID: "b", Type: "bug", Status: "in-progress", Priority: "high"},
	}
	candidates := []*Bean{all[1], all[2]}
	got := RankNext(candidates, all, NextOptions{InProgress: true, PriorityNames: testPriorities, TypeNames: testTypes})
	if ids := rowIDs(got.Rows); !slices.Equal(ids, []string{"a", "b"}) {
		t.Errorf("rows = %v, want [a b]", ids)
	}
	for _, r := range got.Rows {
		if r.Lane != LaneInProgress {
			t.Errorf("%s: lane = %q, want in-progress", r.Bean.ID, r.Lane)
		}
	}
}

func TestRankNextReason(t *testing.T) {
	front := &Bean{ID: "e", Type: "epic", Status: "in-progress"}
	all := []*Bean{
		front,
		{ID: "t", Type: "task", Status: "todo", Parent: "e"},
		{ID: "b", Type: "bug", Status: "todo", Priority: "high", Parent: "e"},
		{ID: "d", Type: "task", Status: "draft", Parent: "e"},
		{ID: "r", Type: "task", Status: "todo"},
	}
	want := map[string]string{
		"t": "under epic e (in-progress)",
		"b": "high bug under epic e (in-progress)",
		"d": "draft to refine under epic e (in-progress)",
		"r": "no front",
	}
	for _, r := range rank(all, nil).Rows {
		if r.Reason != want[r.Bean.ID] {
			t.Errorf("%s: reason = %q, want %q", r.Bean.ID, r.Reason, want[r.Bean.ID])
		}
	}
	for _, r := range rank(all, front).Rows {
		if r.Bean.ID == "t" && r.Reason != "under epic e (focus)" {
			t.Errorf("t with focus: reason = %q", r.Reason)
		}
	}
}

func TestFocusProblem(t *testing.T) {
	all := []*Bean{
		{ID: "e", Type: "epic", Status: "todo"},
		{ID: "m", Type: "milestone", Status: "completed"},
		{ID: "under", Type: "epic", Status: "todo", Parent: "m"},
		{ID: "t", Type: "task", Status: "todo"},
		{ID: "s", Type: "epic", Status: "scrapped"},
	}
	tests := []struct {
		id   string
		want string
	}{
		{"e", ""},
		{"missing", "not found"},
		{"t", "is a task, not a milestone or epic"},
		{"m", "completed"},
		{"s", "scrapped"},
		{"under", "ancestor m is completed"},
	}
	for _, tt := range tests {
		if got := FocusProblem(tt.id, all); got != tt.want {
			t.Errorf("FocusProblem(%q) = %q, want %q", tt.id, got, tt.want)
		}
	}
}

func TestRankNextSkipsFrontsUnderClosedAncestor(t *testing.T) {
	all := []*Bean{
		{ID: "m", Type: "milestone", Status: "completed"},
		{ID: "dead", Type: "epic", Status: "in-progress", Parent: "m"},
		{ID: "live", Type: "epic", Status: "in-progress"},
	}
	got := rank(all, nil)
	if len(got.Fronts) != 1 || got.Fronts[0].ID != "live" {
		t.Errorf("fronts = %v, want only live", got.Fronts)
	}
}
