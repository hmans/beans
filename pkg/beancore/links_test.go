package beancore

import (
	"testing"

	"github.com/hmans/beans/pkg/bean"
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

func TestActiveBlockedByIds(t *testing.T) {
	core, _ := setupTestCore(t)

	activeTodo := &bean.Bean{ID: "blocker-todo", Title: "Active Todo", Status: "todo"}
	activeInProgress := &bean.Bean{ID: "blocker-inprogress", Title: "Active In-Progress", Status: "in-progress"}
	completedBlocker := &bean.Bean{ID: "blocker-completed", Title: "Completed", Status: "completed"}
	scrappedBlocker := &bean.Bean{ID: "blocker-scrapped", Title: "Scrapped", Status: "scrapped"}
	target := &bean.Bean{
		ID:        "target",
		Title:     "Target",
		Status:    "todo",
		BlockedBy: []string{"blocker-todo", "blocker-inprogress", "blocker-completed", "blocker-scrapped"},
	}
	noBlockedBy := &bean.Bean{ID: "no-blocked-by", Title: "No blocked-by", Status: "todo"}

	for _, b := range []*bean.Bean{activeTodo, activeInProgress, completedBlocker, scrappedBlocker, target, noBlockedBy} {
		if err := core.Create(b); err != nil {
			t.Fatalf("Create error: %v", err)
		}
	}

	t.Run("returns only active blocked-by IDs", func(t *testing.T) {
		ids := core.ActiveBlockedByIds("target")
		if len(ids) != 2 {
			t.Errorf("ActiveBlockedByIds() returned %d IDs, want 2: %v", len(ids), ids)
		}
		idSet := make(map[string]bool)
		for _, id := range ids {
			idSet[id] = true
		}
		if !idSet["blocker-todo"] {
			t.Error("expected blocker-todo in result")
		}
		if !idSet["blocker-inprogress"] {
			t.Error("expected blocker-inprogress in result")
		}
		if idSet["blocker-completed"] {
			t.Error("completed blocker should not be in result")
		}
		if idSet["blocker-scrapped"] {
			t.Error("scrapped blocker should not be in result")
		}
	})

	t.Run("returns nil for bean with no blocked-by", func(t *testing.T) {
		ids := core.ActiveBlockedByIds("no-blocked-by")
		if ids != nil {
			t.Errorf("ActiveBlockedByIds() returned %v, want nil", ids)
		}
	})

	t.Run("returns nil for nonexistent bean", func(t *testing.T) {
		ids := core.ActiveBlockedByIds("nonexistent")
		if ids != nil {
			t.Errorf("ActiveBlockedByIds() returned %v, want nil", ids)
		}
	})
}

func TestActiveBlockingIds(t *testing.T) {
	core, _ := setupTestCore(t)

	activeTodo := &bean.Bean{ID: "active-todo", Title: "Active Todo", Status: "todo"}
	activeInProgress := &bean.Bean{ID: "active-inprogress", Title: "Active In-Progress", Status: "in-progress"}
	completed := &bean.Bean{ID: "completed-target", Title: "Completed Target", Status: "completed"}
	scrapped := &bean.Bean{ID: "scrapped-target", Title: "Scrapped Target", Status: "scrapped"}
	blocker := &bean.Bean{
		ID:       "the-blocker",
		Title:    "The Blocker",
		Status:   "todo",
		Blocking: []string{"active-todo", "active-inprogress", "completed-target", "scrapped-target"},
	}
	noBlocking := &bean.Bean{ID: "no-blocking", Title: "No blocking", Status: "todo"}

	for _, b := range []*bean.Bean{activeTodo, activeInProgress, completed, scrapped, blocker, noBlocking} {
		if err := core.Create(b); err != nil {
			t.Fatalf("Create error: %v", err)
		}
	}

	t.Run("returns only active blocking IDs", func(t *testing.T) {
		ids := core.ActiveBlockingIds("the-blocker")
		if len(ids) != 2 {
			t.Errorf("ActiveBlockingIds() returned %d IDs, want 2: %v", len(ids), ids)
		}
		idSet := make(map[string]bool)
		for _, id := range ids {
			idSet[id] = true
		}
		if !idSet["active-todo"] {
			t.Error("expected active-todo in result")
		}
		if !idSet["active-inprogress"] {
			t.Error("expected active-inprogress in result")
		}
		if idSet["completed-target"] {
			t.Error("completed target should not be in result")
		}
		if idSet["scrapped-target"] {
			t.Error("scrapped target should not be in result")
		}
	})

	t.Run("returns nil for bean with no blocking", func(t *testing.T) {
		ids := core.ActiveBlockingIds("no-blocking")
		if ids != nil {
			t.Errorf("ActiveBlockingIds() returned %v, want nil", ids)
		}
	})

	t.Run("returns nil for nonexistent bean", func(t *testing.T) {
		ids := core.ActiveBlockingIds("nonexistent")
		if ids != nil {
			t.Errorf("ActiveBlockingIds() returned %v, want nil", ids)
		}
	})
}

func TestImplicitStatus(t *testing.T) {
	core, _ := setupTestCore(t)

	// Hierarchy: milestone (scrapped) -> epic (todo) -> task (in-progress)
	milestone := &bean.Bean{ID: "m001", Title: "Milestone", Status: "scrapped", Type: "milestone"}
	epic := &bean.Bean{ID: "e001", Title: "Epic", Status: "todo", Type: "epic", Parent: "m001"}
	task := &bean.Bean{ID: "t001", Title: "Task", Status: "in-progress", Type: "task", Parent: "e001"}
	orphan := &bean.Bean{ID: "x001", Title: "Orphan", Status: "todo", Type: "task"}

	// completed parent hierarchy: milestone2 (completed) -> task2
	milestone2 := &bean.Bean{ID: "m002", Title: "Milestone2", Status: "completed", Type: "milestone"}
	task2 := &bean.Bean{ID: "t002", Title: "Task2", Status: "todo", Type: "task", Parent: "m002"}

	for _, b := range []*bean.Bean{milestone, epic, task, orphan, milestone2, task2} {
		if err := core.Create(b); err != nil {
			t.Fatalf("Create error: %v", err)
		}
	}

	t.Run("direct scrapped parent", func(t *testing.T) {
		status, fromID := core.ImplicitStatus("e001")
		if status != "scrapped" {
			t.Errorf("want status=scrapped, got %q", status)
		}
		if fromID != "m001" {
			t.Errorf("want fromID=m001, got %q", fromID)
		}
	})

	t.Run("transitive scrapped grandparent", func(t *testing.T) {
		status, fromID := core.ImplicitStatus("t001")
		if status != "scrapped" {
			t.Errorf("want status=scrapped, got %q", status)
		}
		if fromID != "m001" {
			t.Errorf("want fromID=m001, got %q", fromID)
		}
	})

	t.Run("no terminal ancestor", func(t *testing.T) {
		status, fromID := core.ImplicitStatus("x001")
		if status != "" || fromID != "" {
			t.Errorf("want empty, got status=%q fromID=%q", status, fromID)
		}
	})

	t.Run("no parent at all (milestone itself)", func(t *testing.T) {
		status, fromID := core.ImplicitStatus("m001")
		if status != "" || fromID != "" {
			t.Errorf("want empty, got status=%q fromID=%q", status, fromID)
		}
	})

	t.Run("completed parent", func(t *testing.T) {
		status, fromID := core.ImplicitStatus("t002")
		if status != "completed" {
			t.Errorf("want status=completed, got %q", status)
		}
		if fromID != "m002" {
			t.Errorf("want fromID=m002, got %q", fromID)
		}
	})

	t.Run("nonexistent bean", func(t *testing.T) {
		status, fromID := core.ImplicitStatus("nope")
		if status != "" || fromID != "" {
			t.Errorf("want empty, got status=%q fromID=%q", status, fromID)
		}
	})
}

func TestImplicitBlocking(t *testing.T) {
	t.Run("IsBlocked includes implicit blocking", func(t *testing.T) {
		core, _ := setupTestCore(t)

		blocker := &bean.Bean{ID: "blocker", Title: "Blocker", Status: "todo", Type: "epic", Blocking: []string{"epic-b"}}
		epicB := &bean.Bean{ID: "epic-b", Title: "Epic B", Status: "todo", Type: "epic"}
		child := &bean.Bean{ID: "child", Title: "Child", Status: "todo", Type: "feature", Parent: "epic-b"}

		for _, b := range []*bean.Bean{blocker, epicB, child} {
			if err := core.Create(b); err != nil {
				t.Fatalf("Create error: %v", err)
			}
		}

		// child is not explicitly blocked, but parent epic-b is
		if !core.IsBlocked("child") {
			t.Error("child should be blocked (parent epic-b is blocked by blocker)")
		}
		if !core.IsBlocked("epic-b") {
			t.Error("epic-b should be blocked (directly by blocker)")
		}
	})

	t.Run("IsExplicitlyBlocked only checks direct blockers", func(t *testing.T) {
		core, _ := setupTestCore(t)

		blocker := &bean.Bean{ID: "blocker", Title: "Blocker", Status: "todo", Type: "epic", Blocking: []string{"epic-b"}}
		epicB := &bean.Bean{ID: "epic-b", Title: "Epic B", Status: "todo", Type: "epic"}
		child := &bean.Bean{ID: "child", Title: "Child", Status: "todo", Type: "feature", Parent: "epic-b"}

		for _, b := range []*bean.Bean{blocker, epicB, child} {
			if err := core.Create(b); err != nil {
				t.Fatalf("Create error: %v", err)
			}
		}

		if core.IsExplicitlyBlocked("child") {
			t.Error("child should NOT be explicitly blocked (no direct blockers)")
		}
		if !core.IsExplicitlyBlocked("epic-b") {
			t.Error("epic-b should be explicitly blocked")
		}
	})

	t.Run("IsImplicitlyBlocked only checks ancestor blockers", func(t *testing.T) {
		core, _ := setupTestCore(t)

		blocker := &bean.Bean{ID: "blocker", Title: "Blocker", Status: "todo", Type: "epic", Blocking: []string{"epic-b"}}
		epicB := &bean.Bean{ID: "epic-b", Title: "Epic B", Status: "todo", Type: "epic"}
		child := &bean.Bean{ID: "child", Title: "Child", Status: "todo", Type: "feature", Parent: "epic-b"}

		for _, b := range []*bean.Bean{blocker, epicB, child} {
			if err := core.Create(b); err != nil {
				t.Fatalf("Create error: %v", err)
			}
		}

		if !core.IsImplicitlyBlocked("child") {
			t.Error("child should be implicitly blocked (parent epic-b is blocked)")
		}
		if core.IsImplicitlyBlocked("epic-b") {
			t.Error("epic-b should NOT be implicitly blocked (it's directly blocked, no blocked ancestor)")
		}
	})

	t.Run("grandchild is implicitly blocked", func(t *testing.T) {
		core, _ := setupTestCore(t)

		blocker := &bean.Bean{ID: "blocker", Title: "Blocker", Status: "todo", Type: "milestone", Blocking: []string{"ms"}}
		ms := &bean.Bean{ID: "ms", Title: "Milestone", Status: "todo", Type: "milestone"}
		epic := &bean.Bean{ID: "epic", Title: "Epic", Status: "todo", Type: "epic", Parent: "ms"}
		task := &bean.Bean{ID: "task", Title: "Task", Status: "todo", Type: "task", Parent: "epic"}

		for _, b := range []*bean.Bean{blocker, ms, epic, task} {
			if err := core.Create(b); err != nil {
				t.Fatalf("Create error: %v", err)
			}
		}

		if !core.IsBlocked("task") {
			t.Error("task should be blocked (grandparent ms is blocked)")
		}
		if !core.IsImplicitlyBlocked("task") {
			t.Error("task should be implicitly blocked")
		}
	})

	t.Run("resolved blocker means not blocked", func(t *testing.T) {
		core, _ := setupTestCore(t)

		blocker := &bean.Bean{ID: "blocker", Title: "Blocker", Status: "completed", Type: "epic", Blocking: []string{"epic-b"}}
		epicB := &bean.Bean{ID: "epic-b", Title: "Epic B", Status: "todo", Type: "epic"}
		child := &bean.Bean{ID: "child", Title: "Child", Status: "todo", Type: "feature", Parent: "epic-b"}

		for _, b := range []*bean.Bean{blocker, epicB, child} {
			if err := core.Create(b); err != nil {
				t.Fatalf("Create error: %v", err)
			}
		}

		if core.IsBlocked("child") {
			t.Error("child should NOT be blocked (blocker is completed)")
		}
	})

	t.Run("bean with no parent is not implicitly blocked", func(t *testing.T) {
		core, _ := setupTestCore(t)

		task := &bean.Bean{ID: "task", Title: "Task", Status: "todo", Type: "task"}
		if err := core.Create(task); err != nil {
			t.Fatalf("Create error: %v", err)
		}

		if core.IsImplicitlyBlocked("task") {
			t.Error("task with no parent should not be implicitly blocked")
		}
	})

	t.Run("FindAllBlockers returns both direct and ancestor blockers", func(t *testing.T) {
		core, _ := setupTestCore(t)

		ancestorBlocker := &bean.Bean{ID: "anc-blocker", Title: "Ancestor Blocker", Status: "todo", Type: "milestone", Blocking: []string{"ms"}}
		ms := &bean.Bean{ID: "ms", Title: "Milestone", Status: "todo", Type: "milestone"}
		directBlocker := &bean.Bean{ID: "dir-blocker", Title: "Direct Blocker", Status: "todo", Type: "task", Blocking: []string{"task"}}
		task := &bean.Bean{ID: "task", Title: "Task", Status: "todo", Type: "task", Parent: "ms"}

		for _, b := range []*bean.Bean{ancestorBlocker, ms, directBlocker, task} {
			if err := core.Create(b); err != nil {
				t.Fatalf("Create error: %v", err)
			}
		}

		allBlockers := core.FindAllBlockers("task")
		if len(allBlockers) != 2 {
			t.Errorf("want 2 blockers (direct + ancestor), got %d", len(allBlockers))
		}

		directBlockers := core.FindActiveBlockers("task")
		if len(directBlockers) != 1 {
			t.Errorf("want 1 direct blocker, got %d", len(directBlockers))
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
