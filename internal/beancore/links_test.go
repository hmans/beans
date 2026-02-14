package beancore

import (
	"testing"

	"github.com/hmans/beans/internal/bean"
)

func TestFindIncomingLinks(t *testing.T) {
	core, _ := setupTestCore(t)

	// Create beans with relationships
	// A -> B (blocks)
	// A -> C (parent)
	// D -> B (blocks)
	beanA := &bean.Bean{
		ID:     "aaa1",
		Title:  "Bean A",
		Status: "todo",
		Blocking: []string{"bbb2"},
		Parent: "ccc3",
	}
	beanB := &bean.Bean{ID: "bbb2", Title: "Bean B", Status: "todo"}
	beanC := &bean.Bean{ID: "ccc3", Title: "Bean C", Status: "todo"}
	beanD := &bean.Bean{
		ID:     "ddd4",
		Title:  "Bean D",
		Status: "todo",
		Blocking: []string{"bbb2"},
	}

	for _, b := range []*bean.Bean{beanA, beanB, beanC, beanD} {
		if err := core.Create(b); err != nil {
			t.Fatalf("Create error: %v", err)
		}
	}

	t.Run("multiple incoming blocks", func(t *testing.T) {
		links := core.FindIncomingLinks("bbb2")
		if len(links) != 2 {
			t.Errorf("FindIncomingLinks(bbb2) = %d links, want 2", len(links))
		}

		// Check both A and D block B
		fromIDs := make(map[string]string)
		for _, link := range links {
			fromIDs[link.FromBean.ID] = link.LinkType
		}
		if fromIDs["aaa1"] != "blocking" {
			t.Error("expected aaa1 -> bbb2 via blocks")
		}
		if fromIDs["ddd4"] != "blocking" {
			t.Error("expected ddd4 -> bbb2 via blocks")
		}
	})

	t.Run("single incoming parent link", func(t *testing.T) {
		links := core.FindIncomingLinks("ccc3")
		if len(links) != 1 {
			t.Errorf("FindIncomingLinks(ccc3) = %d links, want 1", len(links))
		}
		if links[0].FromBean.ID != "aaa1" || links[0].LinkType != "parent" {
			t.Errorf("expected aaa1 -> ccc3 via parent, got %s -> ccc3 via %s", links[0].FromBean.ID, links[0].LinkType)
		}
	})

	t.Run("no incoming links", func(t *testing.T) {
		links := core.FindIncomingLinks("aaa1")
		if len(links) != 0 {
			t.Errorf("FindIncomingLinks(aaa1) = %d links, want 0", len(links))
		}
	})

	t.Run("nonexistent bean", func(t *testing.T) {
		links := core.FindIncomingLinks("nonexistent")
		if len(links) != 0 {
			t.Errorf("FindIncomingLinks(nonexistent) = %d links, want 0", len(links))
		}
	})
}

func TestDetectCycle(t *testing.T) {
	core, _ := setupTestCore(t)

	// Create a chain: A blocks B, B blocks C
	beanA := &bean.Bean{
		ID:     "aaa1",
		Title:  "Bean A",
		Status: "todo",
		Blocking: []string{"bbb2"},
	}
	beanB := &bean.Bean{
		ID:     "bbb2",
		Title:  "Bean B",
		Status: "todo",
		Blocking: []string{"ccc3"},
	}
	beanC := &bean.Bean{
		ID:     "ccc3",
		Title:  "Bean C",
		Status: "todo",
	}

	for _, b := range []*bean.Bean{beanA, beanB, beanC} {
		if err := core.Create(b); err != nil {
			t.Fatalf("Create error: %v", err)
		}
	}

	t.Run("would create cycle", func(t *testing.T) {
		// Adding C blocks A would create: A -> B -> C -> A
		cycle := core.DetectCycle("ccc3", "blocking", "aaa1")
		if cycle == nil {
			t.Error("DetectCycle should return cycle path when cycle would be created")
		}
		if len(cycle) < 3 {
			t.Errorf("cycle path too short: %v", cycle)
		}
	})

	t.Run("no cycle", func(t *testing.T) {
		// Adding D blocks A would not create a cycle (D doesn't exist in chain)
		beanD := &bean.Bean{ID: "ddd4", Title: "Bean D", Status: "todo"}
		if err := core.Create(beanD); err != nil {
			t.Fatalf("Create error: %v", err)
		}

		cycle := core.DetectCycle("ddd4", "blocking", "aaa1")
		if cycle != nil {
			t.Errorf("DetectCycle should return nil when no cycle, got: %v", cycle)
		}
	})

	t.Run("parent cycle detection", func(t *testing.T) {
		// Create parent chain: X -> Y -> Z (X has parent Y, Y has parent Z)
		beanX := &bean.Bean{
			ID:     "xxx1",
			Title:  "Bean X",
			Status: "todo",
			Parent: "yyy2",
		}
		beanY := &bean.Bean{
			ID:     "yyy2",
			Title:  "Bean Y",
			Status: "todo",
			Parent: "zzz3",
		}
		beanZ := &bean.Bean{
			ID:     "zzz3",
			Title:  "Bean Z",
			Status: "todo",
		}

		for _, b := range []*bean.Bean{beanX, beanY, beanZ} {
			if err := core.Create(b); err != nil {
				t.Fatalf("Create error: %v", err)
			}
		}

		// Adding Z parent of X would create: X -> Y -> Z -> X
		cycle := core.DetectCycle("zzz3", "parent", "xxx1")
		if cycle == nil {
			t.Error("DetectCycle should detect parent cycles")
		}
	})
}

func TestCheckAllLinks(t *testing.T) {
	core, _ := setupTestCore(t)

	// Create beans with various link issues:
	// - Broken parent link to nonexistent bean
	// - Self-reference in blocks
	// - Cycle (A -> B -> A via blocks)
	beanA := &bean.Bean{
		ID:     "aaa1",
		Title:  "Bean A",
		Status: "todo",
		Blocking: []string{"bbb2", "aaa1"}, // aaa1 is self-reference
		Parent: "nonexistent",
	}
	beanB := &bean.Bean{
		ID:     "bbb2",
		Title:  "Bean B",
		Status: "todo",
		Blocking: []string{"aaa1"}, // creates cycle
	}

	for _, b := range []*bean.Bean{beanA, beanB} {
		if err := core.Create(b); err != nil {
			t.Fatalf("Create error: %v", err)
		}
	}

	result := core.CheckAllLinks()

	t.Run("detects broken links", func(t *testing.T) {
		if len(result.BrokenLinks) != 1 {
			t.Errorf("BrokenLinks = %d, want 1", len(result.BrokenLinks))
		}
		if len(result.BrokenLinks) > 0 {
			bl := result.BrokenLinks[0]
			if bl.BeanID != "aaa1" || bl.LinkType != "parent" || bl.Target != "nonexistent" {
				t.Errorf("unexpected broken link: %+v", bl)
			}
		}
	})

	t.Run("detects self-references", func(t *testing.T) {
		if len(result.SelfLinks) != 1 {
			t.Errorf("SelfLinks = %d, want 1", len(result.SelfLinks))
		}
		if len(result.SelfLinks) > 0 {
			sl := result.SelfLinks[0]
			if sl.BeanID != "aaa1" || sl.LinkType != "blocking" {
				t.Errorf("unexpected self-link: %+v", sl)
			}
		}
	})

	t.Run("detects cycles", func(t *testing.T) {
		if len(result.Cycles) != 1 {
			t.Errorf("Cycles = %d, want 1", len(result.Cycles))
		}
		if len(result.Cycles) > 0 {
			c := result.Cycles[0]
			if c.LinkType != "blocking" {
				t.Errorf("cycle link type = %q, want 'blocks'", c.LinkType)
			}
			if len(c.Path) < 3 {
				t.Errorf("cycle path too short: %v", c.Path)
			}
		}
	})

	t.Run("HasIssues returns true", func(t *testing.T) {
		if !result.HasIssues() {
			t.Error("HasIssues() should return true")
		}
	})

	t.Run("TotalIssues counts all", func(t *testing.T) {
		if result.TotalIssues() != 3 {
			t.Errorf("TotalIssues() = %d, want 3", result.TotalIssues())
		}
	})
}

func TestCheckAllLinksClean(t *testing.T) {
	core, _ := setupTestCore(t)

	// Create clean beans with no issues
	beanA := &bean.Bean{
		ID:     "aaa1",
		Title:  "Bean A",
		Status: "todo",
		Blocking: []string{"bbb2"},
	}
	beanB := &bean.Bean{
		ID:     "bbb2",
		Title:  "Bean B",
		Status: "todo",
	}

	for _, b := range []*bean.Bean{beanA, beanB} {
		if err := core.Create(b); err != nil {
			t.Fatalf("Create error: %v", err)
		}
	}

	result := core.CheckAllLinks()

	if result.HasIssues() {
		t.Errorf("HasIssues() should return false for clean beans, got broken=%d self=%d cycles=%d",
			len(result.BrokenLinks), len(result.SelfLinks), len(result.Cycles))
	}
}

func TestRemoveLinksTo(t *testing.T) {
	core, _ := setupTestCore(t)

	// Create beans where multiple beans link to one target
	beanA := &bean.Bean{
		ID:     "aaa1",
		Title:  "Bean A",
		Status: "todo",
		Blocking: []string{"target"},
		Parent: "target",
	}
	beanB := &bean.Bean{
		ID:     "bbb2",
		Title:  "Bean B",
		Status: "todo",
		Blocking: []string{"target"},
	}
	target := &bean.Bean{
		ID:     "target",
		Title:  "Target Bean",
		Status: "todo",
	}

	for _, b := range []*bean.Bean{beanA, beanB, target} {
		if err := core.Create(b); err != nil {
			t.Fatalf("Create error: %v", err)
		}
	}

	// Remove all links to target
	removed, err := core.RemoveLinksTo("target")
	if err != nil {
		t.Fatalf("RemoveLinksTo error: %v", err)
	}

	if removed != 3 {
		t.Errorf("removed = %d, want 3", removed)
	}

	// Verify links are gone
	loadedA, _ := core.Get("aaa1")
	if loadedA.Parent != "" || len(loadedA.Blocking) != 0 {
		t.Errorf("Bean A still has relationships: parent=%q blocks=%v", loadedA.Parent, loadedA.Blocking)
	}

	loadedB, _ := core.Get("bbb2")
	if len(loadedB.Blocking) != 0 {
		t.Errorf("Bean B still has %d blocks, want 0", len(loadedB.Blocking))
	}
}

func TestFixBrokenLinks(t *testing.T) {
	core, _ := setupTestCore(t)

	// Create bean with broken link and self-reference
	beanA := &bean.Bean{
		ID:     "aaa1",
		Title:  "Bean A",
		Status: "todo",
		Blocking: []string{"bbb2", "aaa1"}, // bbb2 is valid, aaa1 is self-reference
		Parent: "nonexistent",             // broken
	}
	beanB := &bean.Bean{
		ID:     "bbb2",
		Title:  "Bean B",
		Status: "todo",
	}

	for _, b := range []*bean.Bean{beanA, beanB} {
		if err := core.Create(b); err != nil {
			t.Fatalf("Create error: %v", err)
		}
	}

	// Fix broken links
	fixed, err := core.FixBrokenLinks()
	if err != nil {
		t.Fatalf("FixBrokenLinks error: %v", err)
	}

	if fixed != 2 {
		t.Errorf("fixed = %d, want 2", fixed)
	}

	// Verify only valid link remains
	loadedA, _ := core.Get("aaa1")
	if len(loadedA.Blocking) != 1 {
		t.Errorf("Bean A has %d blocks, want 1", len(loadedA.Blocking))
	}
	if !loadedA.IsBlocking("bbb2") {
		t.Error("valid 'blocks' link should be preserved")
	}
	if loadedA.Parent != "" {
		t.Errorf("broken parent should be removed, got %q", loadedA.Parent)
	}
}

func TestLinkCheckResultMethods(t *testing.T) {
	t.Run("empty result", func(t *testing.T) {
		r := &LinkCheckResult{
			BrokenLinks: []BrokenLink{},
			SelfLinks:   []SelfLink{},
			Cycles:      []Cycle{},
		}
		if r.HasIssues() {
			t.Error("empty result should not have issues")
		}
		if r.TotalIssues() != 0 {
			t.Errorf("TotalIssues() = %d, want 0", r.TotalIssues())
		}
	})

	t.Run("with issues", func(t *testing.T) {
		r := &LinkCheckResult{
			BrokenLinks: []BrokenLink{{BeanID: "a", LinkType: "blocking", Target: "x"}},
			SelfLinks:   []SelfLink{{BeanID: "b", LinkType: "parent"}},
			Cycles:      []Cycle{{LinkType: "blocking", Path: []string{"a", "b", "a"}}},
		}
		if !r.HasIssues() {
			t.Error("result with issues should have issues")
		}
		if r.TotalIssues() != 3 {
			t.Errorf("TotalIssues() = %d, want 3", r.TotalIssues())
		}
	})
}

func TestCanonicalCycleKey(t *testing.T) {
	tests := []struct {
		path []string
		want string
	}{
		{[]string{"a", "b", "c", "a"}, "a->b->c"},
		{[]string{"c", "a", "b", "c"}, "a->b->c"}, // same cycle, different start
		{[]string{"b", "c", "a", "b"}, "a->b->c"}, // same cycle, different start
		{[]string{"x", "y", "x"}, "x->y"},
		{[]string{"a"}, ""},
		{[]string{}, ""},
	}

	for _, tt := range tests {
		got := canonicalCycleKey(tt.path)
		if got != tt.want {
			t.Errorf("canonicalCycleKey(%v) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestIsBlocked(t *testing.T) {
	core, _ := setupTestCore(t)

	// Create test beans with various blocking scenarios
	activeBlocker := &bean.Bean{
		ID:       "active-blocker",
		Title:    "Active Blocker",
		Status:   "todo",
		Blocking: []string{"blocked-by-active"},
	}
	completedBlocker := &bean.Bean{
		ID:       "completed-blocker",
		Title:    "Completed Blocker",
		Status:   "completed",
		Blocking: []string{"blocked-by-completed"},
	}
	scrappedBlocker := &bean.Bean{
		ID:       "scrapped-blocker",
		Title:    "Scrapped Blocker",
		Status:   "scrapped",
		Blocking: []string{"blocked-by-scrapped"},
	}
	blockedByActive := &bean.Bean{
		ID:     "blocked-by-active",
		Title:  "Blocked by Active",
		Status: "todo",
	}
	blockedByCompleted := &bean.Bean{
		ID:     "blocked-by-completed",
		Title:  "Blocked by Completed",
		Status: "todo",
	}
	blockedByScrapped := &bean.Bean{
		ID:     "blocked-by-scrapped",
		Title:  "Blocked by Scrapped",
		Status: "todo",
	}
	notBlocked := &bean.Bean{
		ID:     "not-blocked",
		Title:  "Not Blocked",
		Status: "todo",
	}
	// Bean with direct blocked_by field
	blockedByFieldActive := &bean.Bean{
		ID:        "blocked-by-field-active",
		Title:     "Blocked by Field (Active)",
		Status:    "todo",
		BlockedBy: []string{"active-blocker"},
	}
	blockedByFieldCompleted := &bean.Bean{
		ID:        "blocked-by-field-completed",
		Title:     "Blocked by Field (Completed)",
		Status:    "todo",
		BlockedBy: []string{"completed-blocker"},
	}
	// Bean with broken blocker link
	blockedByBroken := &bean.Bean{
		ID:        "blocked-by-broken",
		Title:     "Blocked by Broken Link",
		Status:    "todo",
		BlockedBy: []string{"nonexistent"},
	}
	// Bean with multiple blockers (one active, one completed)
	mixedBlockers := &bean.Bean{
		ID:        "mixed-blockers",
		Title:     "Mixed Blockers",
		Status:    "todo",
		BlockedBy: []string{"active-blocker", "completed-blocker"},
	}
	// Bean with multiple blockers (all completed)
	allResolvedBlockers := &bean.Bean{
		ID:        "all-resolved-blockers",
		Title:     "All Resolved Blockers",
		Status:    "todo",
		BlockedBy: []string{"completed-blocker", "scrapped-blocker"},
	}

	beans := []*bean.Bean{
		activeBlocker, completedBlocker, scrappedBlocker,
		blockedByActive, blockedByCompleted, blockedByScrapped,
		notBlocked, blockedByFieldActive, blockedByFieldCompleted,
		blockedByBroken, mixedBlockers, allResolvedBlockers,
	}
	for _, b := range beans {
		if err := core.Create(b); err != nil {
			t.Fatalf("Create error: %v", err)
		}
	}

	tests := []struct {
		name   string
		beanID string
		want   bool
	}{
		{"blocked by active via Blocking field", "blocked-by-active", true},
		{"blocked by completed via Blocking field", "blocked-by-completed", false},
		{"blocked by scrapped via Blocking field", "blocked-by-scrapped", false},
		{"not blocked", "not-blocked", false},
		{"blocked by active via BlockedBy field", "blocked-by-field-active", true},
		{"blocked by completed via BlockedBy field", "blocked-by-field-completed", false},
		{"broken blocker link", "blocked-by-broken", false},
		{"mixed blockers (one active)", "mixed-blockers", true},
		{"all resolved blockers", "all-resolved-blockers", false},
		{"nonexistent bean", "nonexistent", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := core.IsBlocked(tt.beanID)
			if got != tt.want {
				t.Errorf("IsBlocked(%q) = %v, want %v", tt.beanID, got, tt.want)
			}
		})
	}
}

func TestFindActiveBlockers(t *testing.T) {
	core, _ := setupTestCore(t)

	// Create test beans
	activeBlocker1 := &bean.Bean{
		ID:       "active-blocker-1",
		Title:    "Active Blocker 1",
		Status:   "in-progress",
		Blocking: []string{"target"},
	}
	activeBlocker2 := &bean.Bean{
		ID:     "active-blocker-2",
		Title:  "Active Blocker 2",
		Status: "todo",
	}
	completedBlocker := &bean.Bean{
		ID:       "completed-blocker",
		Title:    "Completed Blocker",
		Status:   "completed",
		Blocking: []string{"target"},
	}
	target := &bean.Bean{
		ID:        "target",
		Title:     "Target Bean",
		Status:    "todo",
		BlockedBy: []string{"active-blocker-2", "completed-blocker"},
	}
	noBlockers := &bean.Bean{
		ID:     "no-blockers",
		Title:  "No Blockers",
		Status: "todo",
	}

	beans := []*bean.Bean{activeBlocker1, activeBlocker2, completedBlocker, target, noBlockers}
	for _, b := range beans {
		if err := core.Create(b); err != nil {
			t.Fatalf("Create error: %v", err)
		}
	}

	t.Run("returns active blockers from both sources", func(t *testing.T) {
		blockers := core.FindActiveBlockers("target")
		if len(blockers) != 2 {
			t.Errorf("FindActiveBlockers() returned %d blockers, want 2", len(blockers))
		}
		// Check that both active blockers are present
		ids := make(map[string]bool)
		for _, b := range blockers {
			ids[b.ID] = true
		}
		if !ids["active-blocker-1"] {
			t.Error("expected active-blocker-1 in result")
		}
		if !ids["active-blocker-2"] {
			t.Error("expected active-blocker-2 in result")
		}
		if ids["completed-blocker"] {
			t.Error("completed-blocker should not be in result")
		}
	})

	t.Run("returns nil for bean with no blockers", func(t *testing.T) {
		blockers := core.FindActiveBlockers("no-blockers")
		if len(blockers) != 0 {
			t.Errorf("FindActiveBlockers() returned %d blockers, want 0", len(blockers))
		}
	})

	t.Run("returns nil for nonexistent bean", func(t *testing.T) {
		blockers := core.FindActiveBlockers("nonexistent")
		if blockers != nil {
			t.Errorf("FindActiveBlockers() returned %v, want nil", blockers)
		}
	})
}

func TestIsTransitivelyBlocked(t *testing.T) {
	t.Run("directly blocked bean", func(t *testing.T) {
		core, _ := setupTestCore(t)

		blocker := &bean.Bean{
			ID:       "blocker",
			Title:    "Blocker",
			Status:   "todo",
			Type:     "task",
			Blocking: []string{"target"},
		}
		target := &bean.Bean{
			ID:     "target",
			Title:  "Target",
			Status: "todo",
			Type:   "task",
		}

		for _, b := range []*bean.Bean{blocker, target} {
			if err := core.Create(b); err != nil {
				t.Fatalf("Create error: %v", err)
			}
		}

		if !core.IsTransitivelyBlocked("target") {
			t.Error("target should be transitively blocked (directly blocked)")
		}
	})

	t.Run("child of blocked parent", func(t *testing.T) {
		core, _ := setupTestCore(t)

		epicA := &bean.Bean{
			ID:       "epic-a",
			Title:    "Epic A",
			Status:   "todo",
			Type:     "epic",
			Blocking: []string{"epic-b"},
		}
		epicB := &bean.Bean{
			ID:     "epic-b",
			Title:  "Epic B",
			Status: "todo",
			Type:   "epic",
		}
		featureX := &bean.Bean{
			ID:     "feature-x",
			Title:  "Feature X",
			Status: "todo",
			Type:   "feature",
			Parent: "epic-b",
		}

		for _, b := range []*bean.Bean{epicA, epicB, featureX} {
			if err := core.Create(b); err != nil {
				t.Fatalf("Create error: %v", err)
			}
		}

		if !core.IsTransitivelyBlocked("feature-x") {
			t.Error("feature-x should be transitively blocked (parent epic-b is blocked by epic-a)")
		}
	})

	t.Run("grandchild of blocked grandparent", func(t *testing.T) {
		core, _ := setupTestCore(t)

		blocker := &bean.Bean{
			ID:       "blocker",
			Title:    "Blocker",
			Status:   "todo",
			Type:     "milestone",
			Blocking: []string{"milestone-1"},
		}
		milestone := &bean.Bean{
			ID:     "milestone-1",
			Title:  "Milestone",
			Status: "todo",
			Type:   "milestone",
		}
		epic := &bean.Bean{
			ID:     "epic-1",
			Title:  "Epic",
			Status: "todo",
			Type:   "epic",
			Parent: "milestone-1",
		}
		task := &bean.Bean{
			ID:     "task-1",
			Title:  "Task",
			Status: "todo",
			Type:   "task",
			Parent: "epic-1",
		}

		for _, b := range []*bean.Bean{blocker, milestone, epic, task} {
			if err := core.Create(b); err != nil {
				t.Fatalf("Create error: %v", err)
			}
		}

		if !core.IsTransitivelyBlocked("task-1") {
			t.Error("task-1 should be transitively blocked (grandparent milestone-1 is blocked)")
		}
	})

	t.Run("unblocked child of unblocked parent", func(t *testing.T) {
		core, _ := setupTestCore(t)

		epicC := &bean.Bean{
			ID:     "epic-c",
			Title:  "Epic C",
			Status: "todo",
			Type:   "epic",
		}
		featureY := &bean.Bean{
			ID:     "feature-y",
			Title:  "Feature Y",
			Status: "todo",
			Type:   "feature",
			Parent: "epic-c",
		}

		for _, b := range []*bean.Bean{epicC, featureY} {
			if err := core.Create(b); err != nil {
				t.Fatalf("Create error: %v", err)
			}
		}

		if core.IsTransitivelyBlocked("feature-y") {
			t.Error("feature-y should not be transitively blocked (parent epic-c has no blockers)")
		}
	})

	t.Run("parent blocker resolved", func(t *testing.T) {
		core, _ := setupTestCore(t)

		epicA := &bean.Bean{
			ID:       "epic-a",
			Title:    "Epic A",
			Status:   "completed",
			Type:     "epic",
			Blocking: []string{"epic-b"},
		}
		epicB := &bean.Bean{
			ID:     "epic-b",
			Title:  "Epic B",
			Status: "todo",
			Type:   "epic",
		}
		featureX := &bean.Bean{
			ID:     "feature-x",
			Title:  "Feature X",
			Status: "todo",
			Type:   "feature",
			Parent: "epic-b",
		}

		for _, b := range []*bean.Bean{epicA, epicB, featureX} {
			if err := core.Create(b); err != nil {
				t.Fatalf("Create error: %v", err)
			}
		}

		if core.IsTransitivelyBlocked("feature-x") {
			t.Error("feature-x should not be transitively blocked (epic-a is completed)")
		}
	})

	t.Run("direct IsBlocked unchanged for child of blocked parent", func(t *testing.T) {
		core, _ := setupTestCore(t)

		epicA := &bean.Bean{
			ID:       "epic-a",
			Title:    "Epic A",
			Status:   "todo",
			Type:     "epic",
			Blocking: []string{"epic-b"},
		}
		epicB := &bean.Bean{
			ID:     "epic-b",
			Title:  "Epic B",
			Status: "todo",
			Type:   "epic",
		}
		featureX := &bean.Bean{
			ID:     "feature-x",
			Title:  "Feature X",
			Status: "todo",
			Type:   "feature",
			Parent: "epic-b",
		}

		for _, b := range []*bean.Bean{epicA, epicB, featureX} {
			if err := core.Create(b); err != nil {
				t.Fatalf("Create error: %v", err)
			}
		}

		// IsBlocked should be false for the child (direct-only)
		if core.IsBlocked("feature-x") {
			t.Error("IsBlocked should return false for child of blocked parent (direct-only check)")
		}
		// But IsTransitivelyBlocked should be true
		if !core.IsTransitivelyBlocked("feature-x") {
			t.Error("IsTransitivelyBlocked should return true for child of blocked parent")
		}
	})
}

func TestFindTransitiveBlockers(t *testing.T) {
	t.Run("returns both direct and ancestor blockers", func(t *testing.T) {
		core, _ := setupTestCore(t)

		epicA := &bean.Bean{
			ID:       "epic-a",
			Title:    "Epic A",
			Status:   "todo",
			Type:     "epic",
			Blocking: []string{"epic-b"},
		}
		epicB := &bean.Bean{
			ID:     "epic-b",
			Title:  "Epic B",
			Status: "todo",
			Type:   "epic",
		}
		directBlocker := &bean.Bean{
			ID:       "direct-blocker",
			Title:    "Direct Blocker",
			Status:   "todo",
			Type:     "task",
			Blocking: []string{"feature-x"},
		}
		featureX := &bean.Bean{
			ID:     "feature-x",
			Title:  "Feature X",
			Status: "todo",
			Type:   "feature",
			Parent: "epic-b",
		}

		for _, b := range []*bean.Bean{epicA, epicB, directBlocker, featureX} {
			if err := core.Create(b); err != nil {
				t.Fatalf("Create error: %v", err)
			}
		}

		blockers := core.FindTransitiveBlockers("feature-x")
		if len(blockers) != 2 {
			t.Fatalf("expected 2 blockers, got %d", len(blockers))
		}
		ids := make(map[string]bool)
		for _, b := range blockers {
			ids[b.ID] = true
		}
		if !ids["epic-a"] {
			t.Error("expected epic-a in blockers (inherited from parent)")
		}
		if !ids["direct-blocker"] {
			t.Error("expected direct-blocker in blockers (direct)")
		}
	})

	t.Run("FindActiveBlockers only returns direct blockers", func(t *testing.T) {
		core, _ := setupTestCore(t)

		epicA := &bean.Bean{
			ID:       "epic-a",
			Title:    "Epic A",
			Status:   "todo",
			Type:     "epic",
			Blocking: []string{"epic-b"},
		}
		epicB := &bean.Bean{
			ID:     "epic-b",
			Title:  "Epic B",
			Status: "todo",
			Type:   "epic",
		}
		directBlocker := &bean.Bean{
			ID:       "direct-blocker",
			Title:    "Direct Blocker",
			Status:   "todo",
			Type:     "task",
			Blocking: []string{"feature-x"},
		}
		featureX := &bean.Bean{
			ID:     "feature-x",
			Title:  "Feature X",
			Status: "todo",
			Type:   "feature",
			Parent: "epic-b",
		}

		for _, b := range []*bean.Bean{epicA, epicB, directBlocker, featureX} {
			if err := core.Create(b); err != nil {
				t.Fatalf("Create error: %v", err)
			}
		}

		// FindActiveBlockers should only return the direct blocker
		blockers := core.FindActiveBlockers("feature-x")
		if len(blockers) != 1 {
			t.Fatalf("expected 1 direct blocker, got %d", len(blockers))
		}
		if blockers[0].ID != "direct-blocker" {
			t.Errorf("expected direct-blocker, got %s", blockers[0].ID)
		}
	})

	t.Run("returns nil for nonexistent bean", func(t *testing.T) {
		core, _ := setupTestCore(t)

		blockers := core.FindTransitiveBlockers("nonexistent")
		if blockers != nil {
			t.Errorf("FindTransitiveBlockers() returned %v, want nil", blockers)
		}
	})
}

func TestIsResolvedStatus(t *testing.T) {
	tests := []struct {
		status string
		want   bool
	}{
		{"completed", true},
		{"scrapped", true},
		{"todo", false},
		{"in-progress", false},
		{"draft", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			got := isResolvedStatus(tt.status)
			if got != tt.want {
				t.Errorf("isResolvedStatus(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}
