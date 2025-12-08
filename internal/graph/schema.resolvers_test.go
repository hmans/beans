package graph

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"hmans.dev/beans/internal/bean"
	"hmans.dev/beans/internal/beancore"
	"hmans.dev/beans/internal/config"
	"hmans.dev/beans/internal/graph/model"
)

func setupTestResolver(t *testing.T) (*Resolver, *beancore.Core) {
	t.Helper()
	tmpDir := t.TempDir()
	beansDir := filepath.Join(tmpDir, ".beans")
	if err := os.MkdirAll(beansDir, 0755); err != nil {
		t.Fatalf("failed to create test .beans dir: %v", err)
	}

	cfg := config.Default()
	core := beancore.New(beansDir, cfg)
	if err := core.Load(); err != nil {
		t.Fatalf("failed to load core: %v", err)
	}

	return &Resolver{Core: core}, core
}

func createTestBean(t *testing.T, core *beancore.Core, id, title, status string) *bean.Bean {
	t.Helper()
	b := &bean.Bean{
		ID:     id,
		Slug:   bean.Slugify(title),
		Title:  title,
		Status: status,
	}
	if err := core.Create(b); err != nil {
		t.Fatalf("failed to create test bean: %v", err)
	}
	return b
}

func TestQueryBean(t *testing.T) {
	resolver, core := setupTestResolver(t)
	ctx := context.Background()

	// Create test bean
	createTestBean(t, core, "test-1", "Test Bean", "todo")

	// Test exact match
	t.Run("exact match", func(t *testing.T) {
		qr := resolver.Query()
		got, err := qr.Bean(ctx, "test-1")
		if err != nil {
			t.Fatalf("Bean() error = %v", err)
		}
		if got == nil {
			t.Fatal("Bean() returned nil")
		}
		if got.ID != "test-1" {
			t.Errorf("Bean().ID = %q, want %q", got.ID, "test-1")
		}
	})

	// Test prefix match
	t.Run("prefix match", func(t *testing.T) {
		qr := resolver.Query()
		got, err := qr.Bean(ctx, "test")
		if err != nil {
			t.Fatalf("Bean() error = %v", err)
		}
		if got == nil {
			t.Fatal("Bean() returned nil")
		}
		if got.ID != "test-1" {
			t.Errorf("Bean().ID = %q, want %q", got.ID, "test-1")
		}
	})

	// Test not found
	t.Run("not found", func(t *testing.T) {
		qr := resolver.Query()
		got, err := qr.Bean(ctx, "nonexistent")
		if err != nil {
			t.Fatalf("Bean() error = %v", err)
		}
		if got != nil {
			t.Errorf("Bean() = %v, want nil", got)
		}
	})
}

func TestQueryBeans(t *testing.T) {
	resolver, core := setupTestResolver(t)
	ctx := context.Background()

	// Create test beans
	createTestBean(t, core, "bean-1", "First Bean", "todo")
	createTestBean(t, core, "bean-2", "Second Bean", "in-progress")
	createTestBean(t, core, "bean-3", "Third Bean", "completed")

	t.Run("no filter", func(t *testing.T) {
		qr := resolver.Query()
		got, err := qr.Beans(ctx, nil)
		if err != nil {
			t.Fatalf("Beans() error = %v", err)
		}
		if len(got) != 3 {
			t.Errorf("Beans() count = %d, want 3", len(got))
		}
	})

	t.Run("filter by status", func(t *testing.T) {
		qr := resolver.Query()
		filter := &model.BeanFilter{
			Status: []string{"todo"},
		}
		got, err := qr.Beans(ctx, filter)
		if err != nil {
			t.Fatalf("Beans() error = %v", err)
		}
		if len(got) != 1 {
			t.Errorf("Beans() count = %d, want 1", len(got))
		}
		if got[0].ID != "bean-1" {
			t.Errorf("Beans()[0].ID = %q, want %q", got[0].ID, "bean-1")
		}
	})

	t.Run("filter by multiple statuses", func(t *testing.T) {
		qr := resolver.Query()
		filter := &model.BeanFilter{
			Status: []string{"todo", "in-progress"},
		}
		got, err := qr.Beans(ctx, filter)
		if err != nil {
			t.Fatalf("Beans() error = %v", err)
		}
		if len(got) != 2 {
			t.Errorf("Beans() count = %d, want 2", len(got))
		}
	})

	t.Run("exclude status", func(t *testing.T) {
		qr := resolver.Query()
		filter := &model.BeanFilter{
			ExcludeStatus: []string{"completed"},
		}
		got, err := qr.Beans(ctx, filter)
		if err != nil {
			t.Fatalf("Beans() error = %v", err)
		}
		if len(got) != 2 {
			t.Errorf("Beans() count = %d, want 2", len(got))
		}
	})
}

func TestQueryBeansWithTags(t *testing.T) {
	resolver, core := setupTestResolver(t)
	ctx := context.Background()

	// Create test beans with tags
	b1 := &bean.Bean{ID: "tag-1", Title: "Tagged 1", Status: "todo", Tags: []string{"frontend", "urgent"}}
	b2 := &bean.Bean{ID: "tag-2", Title: "Tagged 2", Status: "todo", Tags: []string{"backend"}}
	b3 := &bean.Bean{ID: "tag-3", Title: "No Tags", Status: "todo"}
	core.Create(b1)
	core.Create(b2)
	core.Create(b3)

	t.Run("filter by tag", func(t *testing.T) {
		qr := resolver.Query()
		filter := &model.BeanFilter{
			Tags: []string{"frontend"},
		}
		got, err := qr.Beans(ctx, filter)
		if err != nil {
			t.Fatalf("Beans() error = %v", err)
		}
		if len(got) != 1 {
			t.Errorf("Beans() count = %d, want 1", len(got))
		}
	})

	t.Run("filter by multiple tags (OR)", func(t *testing.T) {
		qr := resolver.Query()
		filter := &model.BeanFilter{
			Tags: []string{"frontend", "backend"},
		}
		got, err := qr.Beans(ctx, filter)
		if err != nil {
			t.Fatalf("Beans() error = %v", err)
		}
		if len(got) != 2 {
			t.Errorf("Beans() count = %d, want 2", len(got))
		}
	})

	t.Run("exclude by tag", func(t *testing.T) {
		qr := resolver.Query()
		filter := &model.BeanFilter{
			ExcludeTags: []string{"urgent"},
		}
		got, err := qr.Beans(ctx, filter)
		if err != nil {
			t.Fatalf("Beans() error = %v", err)
		}
		if len(got) != 2 {
			t.Errorf("Beans() count = %d, want 2", len(got))
		}
	})
}

func TestQueryBeansWithPriority(t *testing.T) {
	resolver, core := setupTestResolver(t)
	ctx := context.Background()

	// Create test beans with various priorities
	// Empty priority should be treated as "normal"
	b1 := &bean.Bean{ID: "pri-1", Title: "Critical", Status: "todo", Priority: "critical"}
	b2 := &bean.Bean{ID: "pri-2", Title: "High", Status: "todo", Priority: "high"}
	b3 := &bean.Bean{ID: "pri-3", Title: "Normal Explicit", Status: "todo", Priority: "normal"}
	b4 := &bean.Bean{ID: "pri-4", Title: "Normal Implicit", Status: "todo", Priority: ""} // empty = normal
	b5 := &bean.Bean{ID: "pri-5", Title: "Low", Status: "todo", Priority: "low"}
	core.Create(b1)
	core.Create(b2)
	core.Create(b3)
	core.Create(b4)
	core.Create(b5)

	t.Run("filter by normal includes empty priority", func(t *testing.T) {
		qr := resolver.Query()
		filter := &model.BeanFilter{
			Priority: []string{"normal"},
		}
		got, err := qr.Beans(ctx, filter)
		if err != nil {
			t.Fatalf("Beans() error = %v", err)
		}
		// Should include both explicit "normal" and implicit (empty) priority
		if len(got) != 2 {
			t.Errorf("Beans() count = %d, want 2", len(got))
		}
		ids := make(map[string]bool)
		for _, b := range got {
			ids[b.ID] = true
		}
		if !ids["pri-3"] || !ids["pri-4"] {
			t.Errorf("Beans() should include pri-3 and pri-4, got %v", ids)
		}
	})

	t.Run("filter by critical", func(t *testing.T) {
		qr := resolver.Query()
		filter := &model.BeanFilter{
			Priority: []string{"critical"},
		}
		got, err := qr.Beans(ctx, filter)
		if err != nil {
			t.Fatalf("Beans() error = %v", err)
		}
		if len(got) != 1 {
			t.Errorf("Beans() count = %d, want 1", len(got))
		}
		if got[0].ID != "pri-1" {
			t.Errorf("Beans()[0].ID = %q, want %q", got[0].ID, "pri-1")
		}
	})

	t.Run("filter by multiple priorities", func(t *testing.T) {
		qr := resolver.Query()
		filter := &model.BeanFilter{
			Priority: []string{"critical", "high"},
		}
		got, err := qr.Beans(ctx, filter)
		if err != nil {
			t.Fatalf("Beans() error = %v", err)
		}
		if len(got) != 2 {
			t.Errorf("Beans() count = %d, want 2", len(got))
		}
	})

	t.Run("exclude normal excludes empty priority", func(t *testing.T) {
		qr := resolver.Query()
		filter := &model.BeanFilter{
			ExcludePriority: []string{"normal"},
		}
		got, err := qr.Beans(ctx, filter)
		if err != nil {
			t.Fatalf("Beans() error = %v", err)
		}
		// Should exclude both explicit "normal" and implicit (empty) priority
		if len(got) != 3 {
			t.Errorf("Beans() count = %d, want 3", len(got))
		}
		for _, b := range got {
			if b.ID == "pri-3" || b.ID == "pri-4" {
				t.Errorf("Beans() should not include %s", b.ID)
			}
		}
	})
}

func TestBeanRelationships(t *testing.T) {
	resolver, core := setupTestResolver(t)
	ctx := context.Background()

	// Create beans with relationships
	parent := &bean.Bean{ID: "parent-1", Title: "Parent", Status: "todo"}
	child1 := &bean.Bean{
		ID:     "child-1",
		Title:  "Child 1",
		Status: "todo",
		Links:  bean.Links{{Type: "parent", Target: "parent-1"}},
	}
	child2 := &bean.Bean{
		ID:     "child-2",
		Title:  "Child 2",
		Status: "todo",
		Links:  bean.Links{{Type: "parent", Target: "parent-1"}},
	}
	blocker := &bean.Bean{
		ID:     "blocker-1",
		Title:  "Blocker",
		Status: "todo",
		Links:  bean.Links{{Type: "blocks", Target: "child-1"}},
	}

	core.Create(parent)
	core.Create(child1)
	core.Create(child2)
	core.Create(blocker)

	t.Run("parent resolver", func(t *testing.T) {
		br := resolver.Bean()
		got, err := br.Parent(ctx, child1)
		if err != nil {
			t.Fatalf("Parent() error = %v", err)
		}
		if got == nil {
			t.Fatal("Parent() returned nil")
		}
		if got.ID != "parent-1" {
			t.Errorf("Parent().ID = %q, want %q", got.ID, "parent-1")
		}
	})

	t.Run("children resolver", func(t *testing.T) {
		br := resolver.Bean()
		got, err := br.Children(ctx, parent)
		if err != nil {
			t.Fatalf("Children() error = %v", err)
		}
		if len(got) != 2 {
			t.Errorf("Children() count = %d, want 2", len(got))
		}
	})

	t.Run("blockedBy resolver", func(t *testing.T) {
		br := resolver.Bean()
		got, err := br.BlockedBy(ctx, child1)
		if err != nil {
			t.Fatalf("BlockedBy() error = %v", err)
		}
		if len(got) != 1 {
			t.Errorf("BlockedBy() count = %d, want 1", len(got))
		}
		if got[0].ID != "blocker-1" {
			t.Errorf("BlockedBy()[0].ID = %q, want %q", got[0].ID, "blocker-1")
		}
	})

	t.Run("blocks resolver", func(t *testing.T) {
		br := resolver.Bean()
		got, err := br.Blocks(ctx, blocker)
		if err != nil {
			t.Fatalf("Blocks() error = %v", err)
		}
		if len(got) != 1 {
			t.Errorf("Blocks() count = %d, want 1", len(got))
		}
		if got[0].ID != "child-1" {
			t.Errorf("Blocks()[0].ID = %q, want %q", got[0].ID, "child-1")
		}
	})
}

func TestBrokenLinksFiltered(t *testing.T) {
	resolver, core := setupTestResolver(t)
	ctx := context.Background()

	// Create bean with broken link
	b := &bean.Bean{
		ID:     "orphan-1",
		Title:  "Orphan",
		Status: "todo",
		Links:  bean.Links{{Type: "parent", Target: "nonexistent"}},
	}
	core.Create(b)

	t.Run("broken parent link returns nil", func(t *testing.T) {
		br := resolver.Bean()
		got, err := br.Parent(ctx, b)
		if err != nil {
			t.Fatalf("Parent() error = %v", err)
		}
		if got != nil {
			t.Errorf("Parent() = %v, want nil for broken link", got)
		}
	})

	t.Run("link targetBean returns nil for broken", func(t *testing.T) {
		lr := resolver.Link()
		link := &bean.Link{Type: "parent", Target: "nonexistent"}
		got, err := lr.TargetBean(ctx, link)
		if err != nil {
			t.Fatalf("TargetBean() error = %v", err)
		}
		if got != nil {
			t.Errorf("TargetBean() = %v, want nil for broken link", got)
		}
	})
}

func TestQueryBeansWithLinks(t *testing.T) {
	resolver, core := setupTestResolver(t)
	ctx := context.Background()

	// Create beans with various link configurations
	noLinks := &bean.Bean{ID: "no-links", Title: "No Links", Status: "todo"}
	hasParent := &bean.Bean{
		ID:     "has-parent",
		Title:  "Has Parent",
		Status: "todo",
		Links:  bean.Links{{Type: "parent", Target: "no-links"}},
	}
	hasBlocks := &bean.Bean{
		ID:     "has-blocks",
		Title:  "Has Blocks",
		Status: "todo",
		Links:  bean.Links{{Type: "blocks", Target: "has-parent"}},
	}

	core.Create(noLinks)
	core.Create(hasParent)
	core.Create(hasBlocks)

	t.Run("filter hasLinks parent", func(t *testing.T) {
		qr := resolver.Query()
		filter := &model.BeanFilter{
			HasLinks: []*model.LinkFilter{{Type: "parent"}},
		}
		got, err := qr.Beans(ctx, filter)
		if err != nil {
			t.Fatalf("Beans() error = %v", err)
		}
		if len(got) != 1 {
			t.Errorf("Beans() count = %d, want 1", len(got))
		}
		if got[0].ID != "has-parent" {
			t.Errorf("Beans()[0].ID = %q, want %q", got[0].ID, "has-parent")
		}
	})

	t.Run("filter noLinks parent", func(t *testing.T) {
		qr := resolver.Query()
		filter := &model.BeanFilter{
			NoLinks: []*model.LinkFilter{{Type: "parent"}},
		}
		got, err := qr.Beans(ctx, filter)
		if err != nil {
			t.Fatalf("Beans() error = %v", err)
		}
		if len(got) != 2 {
			t.Errorf("Beans() count = %d, want 2", len(got))
		}
	})

	t.Run("filter linkedAs blocks", func(t *testing.T) {
		qr := resolver.Query()
		filter := &model.BeanFilter{
			LinkedAs: []*model.LinkFilter{{Type: "blocks"}},
		}
		got, err := qr.Beans(ctx, filter)
		if err != nil {
			t.Fatalf("Beans() error = %v", err)
		}
		if len(got) != 1 {
			t.Errorf("Beans() count = %d, want 1", len(got))
		}
		if got[0].ID != "has-parent" {
			t.Errorf("Beans()[0].ID = %q, want %q", got[0].ID, "has-parent")
		}
	})

	t.Run("filter noLinkedAs blocks", func(t *testing.T) {
		qr := resolver.Query()
		filter := &model.BeanFilter{
			NoLinkedAs: []*model.LinkFilter{{Type: "blocks"}},
		}
		got, err := qr.Beans(ctx, filter)
		if err != nil {
			t.Fatalf("Beans() error = %v", err)
		}
		if len(got) != 2 {
			t.Errorf("Beans() count = %d, want 2", len(got))
		}
	})

	// Test type:target link filtering
	t.Run("filter hasLinks with target", func(t *testing.T) {
		qr := resolver.Query()
		target := "has-parent"
		filter := &model.BeanFilter{
			HasLinks: []*model.LinkFilter{{Type: "blocks", Target: &target}},
		}
		got, err := qr.Beans(ctx, filter)
		if err != nil {
			t.Fatalf("Beans() error = %v", err)
		}
		if len(got) != 1 {
			t.Errorf("Beans() count = %d, want 1", len(got))
		}
		if got[0].ID != "has-blocks" {
			t.Errorf("Beans()[0].ID = %q, want %q", got[0].ID, "has-blocks")
		}
	})

	t.Run("filter hasLinks with non-matching target", func(t *testing.T) {
		qr := resolver.Query()
		target := "no-links"
		filter := &model.BeanFilter{
			HasLinks: []*model.LinkFilter{{Type: "blocks", Target: &target}},
		}
		got, err := qr.Beans(ctx, filter)
		if err != nil {
			t.Fatalf("Beans() error = %v", err)
		}
		if len(got) != 0 {
			t.Errorf("Beans() count = %d, want 0", len(got))
		}
	})

	t.Run("filter noLinks with target", func(t *testing.T) {
		qr := resolver.Query()
		target := "has-parent"
		filter := &model.BeanFilter{
			NoLinks: []*model.LinkFilter{{Type: "blocks", Target: &target}},
		}
		got, err := qr.Beans(ctx, filter)
		if err != nil {
			t.Fatalf("Beans() error = %v", err)
		}
		// Should exclude has-blocks (which blocks has-parent), leaving no-links and has-parent
		if len(got) != 2 {
			t.Errorf("Beans() count = %d, want 2", len(got))
		}
	})
}
