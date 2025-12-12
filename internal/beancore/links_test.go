package beancore

import (
	"testing"

	"github.com/hmans/beans/internal/bean"
)

func TestFindIncomingLinks(t *testing.T) {
	core, _ := setupTestCore(t)

	// Create beans with links
	// A -> B (blocks)
	// A -> C (epic - hierarchy)
	beanA := &bean.Bean{
		ID:     "aaa1",
		Title:  "Bean A",
		Status: "todo",
		Blocks: []string{"bbb2"},
		Epic:   "ccc3",
	}
	beanB := &bean.Bean{ID: "bbb2", Title: "Bean B", Status: "todo"}
	beanC := &bean.Bean{ID: "ccc3", Title: "Bean C", Status: "todo", Type: "epic"}

	for _, b := range []*bean.Bean{beanA, beanB, beanC} {
		if err := core.Create(b); err != nil {
			t.Fatalf("Create error: %v", err)
		}
	}

	t.Run("incoming blocks link", func(t *testing.T) {
		links := core.FindIncomingLinks("bbb2")
		if len(links) != 1 {
			t.Errorf("FindIncomingLinks(bbb2) = %d links, want 1", len(links))
		}

		if links[0].FromBean.ID != "aaa1" || links[0].LinkType != "blocks" {
			t.Error("expected aaa1 -> bbb2 via blocks")
		}
	})

	t.Run("single incoming link (epic hierarchy)", func(t *testing.T) {
		links := core.FindIncomingLinks("ccc3")
		if len(links) != 1 {
			t.Errorf("FindIncomingLinks(ccc3) = %d links, want 1", len(links))
		}
		if links[0].FromBean.ID != "aaa1" || links[0].LinkType != "epic" {
			t.Errorf("expected aaa1 -> ccc3 via epic, got %s -> ccc3 via %s", links[0].FromBean.ID, links[0].LinkType)
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
		Blocks: []string{"bbb2"},
	}
	beanB := &bean.Bean{
		ID:     "bbb2",
		Title:  "Bean B",
		Status: "todo",
		Blocks: []string{"ccc3"},
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
		cycle := core.DetectCycle("ccc3", "blocks", "aaa1")
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

		cycle := core.DetectCycle("ddd4", "blocks", "aaa1")
		if cycle != nil {
			t.Errorf("DetectCycle should return nil when no cycle, got: %v", cycle)
		}
	})

	t.Run("milestone hierarchy cycle detection", func(t *testing.T) {
		// Create milestone chain: X -> Y (milestone), Y -> Z (milestone)
		beanX := &bean.Bean{
			ID:        "xxx1",
			Title:     "Bean X",
			Status:    "todo",
			Milestone: "yyy2",
		}
		beanY := &bean.Bean{
			ID:        "yyy2",
			Title:     "Bean Y",
			Status:    "todo",
			Type:      "milestone",
			Milestone: "zzz3",
		}
		beanZ := &bean.Bean{
			ID:     "zzz3",
			Title:  "Bean Z",
			Status: "todo",
			Type:   "milestone",
		}

		for _, b := range []*bean.Bean{beanX, beanY, beanZ} {
			if err := core.Create(b); err != nil {
				t.Fatalf("Create error: %v", err)
			}
		}

		// Adding Z -> X (milestone) would create: X -> Y -> Z -> X
		cycle := core.DetectCycle("zzz3", "milestone", "xxx1")
		if cycle == nil {
			t.Error("DetectCycle should detect milestone cycles")
		}
	})
}

func TestCheckAllLinks(t *testing.T) {
	core, _ := setupTestCore(t)

	// Create a bean with various link issues:
	// - Broken link to nonexistent bean
	// - Self-reference (via epic)
	// - Cycle (A -> B -> A via blocks)
	beanA := &bean.Bean{
		ID:        "aaa1",
		Title:     "Bean A",
		Status:    "todo",
		Blocks:    []string{"bbb2"},
		Milestone: "nonexistent", // broken hierarchy link
		Epic:      "aaa1",        // self-reference
	}
	beanB := &bean.Bean{
		ID:     "bbb2",
		Title:  "Bean B",
		Status: "todo",
		Blocks: []string{"aaa1"}, // creates cycle
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
			if bl.BeanID != "aaa1" || bl.LinkType != "milestone" || bl.Target != "nonexistent" {
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
			if sl.BeanID != "aaa1" || sl.LinkType != "epic" {
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
			if c.LinkType != "blocks" {
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
		Blocks: []string{"bbb2"},
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
		Blocks: []string{"target"},
		Epic:   "target",
	}
	target := &bean.Bean{
		ID:     "target",
		Title:  "Target Bean",
		Status: "todo",
		Type:   "epic",
	}

	for _, b := range []*bean.Bean{beanA, target} {
		if err := core.Create(b); err != nil {
			t.Fatalf("Create error: %v", err)
		}
	}

	// Remove all links to target
	removed, err := core.RemoveLinksTo("target")
	if err != nil {
		t.Fatalf("RemoveLinksTo error: %v", err)
	}

	if removed != 2 {
		t.Errorf("removed = %d, want 2", removed)
	}

	// Verify links are gone
	loadedA, _ := core.Get("aaa1")
	if len(loadedA.Blocks) != 0 || loadedA.Epic != "" {
		t.Errorf("Bean A still has links: Blocks=%v Epic=%q", loadedA.Blocks, loadedA.Epic)
	}
}

func TestFixBrokenLinks(t *testing.T) {
	core, _ := setupTestCore(t)

	// Create bean with broken link and self-reference
	beanA := &bean.Bean{
		ID:        "aaa1",
		Title:     "Bean A",
		Status:    "todo",
		Blocks:    []string{"bbb2"},
		Milestone: "nonexistent", // broken
		Epic:      "aaa1",        // self-reference
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
	if !loadedA.HasBlock("bbb2") {
		t.Error("valid 'blocks' link should be preserved")
	}
	if loadedA.Milestone != "" {
		t.Error("broken milestone link should be removed")
	}
	if loadedA.Epic != "" {
		t.Error("self-reference epic link should be removed")
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
			BrokenLinks: []BrokenLink{{BeanID: "a", LinkType: "blocks", Target: "x"}},
			SelfLinks:   []SelfLink{{BeanID: "b", LinkType: "milestone"}},
			Cycles:      []Cycle{{LinkType: "blocks", Path: []string{"a", "b", "a"}}},
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
