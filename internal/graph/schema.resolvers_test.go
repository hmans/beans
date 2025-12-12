package graph

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/beancore"
	"github.com/hmans/beans/internal/config"
	"github.com/hmans/beans/internal/graph/model"
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
	parent := &bean.Bean{ID: "parent-1", Title: "Parent", Status: "todo", Type: "epic"}
	child1 := &bean.Bean{
		ID:     "child-1",
		Title:  "Child 1",
		Status: "todo",
		Epic:   "parent-1",
	}
	child2 := &bean.Bean{
		ID:     "child-2",
		Title:  "Child 2",
		Status: "todo",
		Epic:   "parent-1",
	}
	blocker := &bean.Bean{
		ID:     "blocker-1",
		Title:  "Blocker",
		Status: "todo",
		Blocks: []string{"child-1"},
	}

	core.Create(parent)
	core.Create(child1)
	core.Create(child2)
	core.Create(blocker)

	t.Run("epic resolver (parent)", func(t *testing.T) {
		br := resolver.Bean()
		got, err := br.Epic(ctx, child1)
		if err != nil {
			t.Fatalf("Epic() error = %v", err)
		}
		if got == nil {
			t.Fatal("Epic() returned nil")
		}
		if got.ID != "parent-1" {
			t.Errorf("Epic().ID = %q, want %q", got.ID, "parent-1")
		}
	})

	t.Run("epicItems resolver (children)", func(t *testing.T) {
		br := resolver.Bean()
		got, err := br.EpicItems(ctx, parent)
		if err != nil {
			t.Fatalf("EpicItems() error = %v", err)
		}
		if len(got) != 2 {
			t.Errorf("EpicItems() count = %d, want 2", len(got))
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
		Epic:   "nonexistent",
	}
	core.Create(b)

	t.Run("broken epic link returns nil", func(t *testing.T) {
		br := resolver.Bean()
		got, err := br.Epic(ctx, b)
		if err != nil {
			t.Fatalf("Epic() error = %v", err)
		}
		if got != nil {
			t.Errorf("Epic() = %v, want nil for broken link", got)
		}
	})
}

func TestQueryBeansWithLinks(t *testing.T) {
	resolver, core := setupTestResolver(t)
	ctx := context.Background()

	// Create beans with various link configurations
	noLinks := &bean.Bean{ID: "no-links", Title: "No Links", Status: "todo", Type: "epic"}
	hasParent := &bean.Bean{
		ID:     "has-parent",
		Title:  "Has Parent",
		Status: "todo",
		Epic:   "no-links",
	}
	hasBlocks := &bean.Bean{
		ID:     "has-blocks",
		Title:  "Has Blocks",
		Status: "todo",
		Blocks: []string{"has-parent"},
	}

	core.Create(noLinks)
	core.Create(hasParent)
	core.Create(hasBlocks)

	t.Run("filter hasLinks epic", func(t *testing.T) {
		qr := resolver.Query()
		filter := &model.BeanFilter{
			HasLinks: []*model.LinkFilter{{Type: "epic"}},
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

	t.Run("filter noLinks epic", func(t *testing.T) {
		qr := resolver.Query()
		filter := &model.BeanFilter{
			NoLinks: []*model.LinkFilter{{Type: "epic"}},
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

func TestMutationCreateBean(t *testing.T) {
	resolver, _ := setupTestResolver(t)
	ctx := context.Background()

	t.Run("create with required fields only", func(t *testing.T) {
		mr := resolver.Mutation()
		input := model.CreateBeanInput{
			Title: "New Bean",
		}
		got, err := mr.CreateBean(ctx, input)
		if err != nil {
			t.Fatalf("CreateBean() error = %v", err)
		}
		if got == nil {
			t.Fatal("CreateBean() returned nil")
		}
		if got.Title != "New Bean" {
			t.Errorf("CreateBean().Title = %q, want %q", got.Title, "New Bean")
		}
		// Type defaults to "task"
		if got.Type != "task" {
			t.Errorf("CreateBean().Type = %q, want %q (default)", got.Type, "task")
		}
		if got.ID == "" {
			t.Error("CreateBean().ID is empty")
		}
	})

	t.Run("create with all fields", func(t *testing.T) {
		mr := resolver.Mutation()
		beanType := "feature"
		status := "in-progress"
		priority := "high"
		body := "Test body content"
		input := model.CreateBeanInput{
			Title:    "Full Bean",
			Type:     &beanType,
			Status:   &status,
			Priority: &priority,
			Body:     &body,
			Tags:     []string{"tag1", "tag2"},
			Related:  []string{"some-id"},
		}
		got, err := mr.CreateBean(ctx, input)
		if err != nil {
			t.Fatalf("CreateBean() error = %v", err)
		}
		if got.Type != "feature" {
			t.Errorf("CreateBean().Type = %q, want %q", got.Type, "feature")
		}
		if got.Status != "in-progress" {
			t.Errorf("CreateBean().Status = %q, want %q", got.Status, "in-progress")
		}
		if got.Priority != "high" {
			t.Errorf("CreateBean().Priority = %q, want %q", got.Priority, "high")
		}
		if got.Body != "Test body content" {
			t.Errorf("CreateBean().Body = %q, want %q", got.Body, "Test body content")
		}
		if len(got.Tags) != 2 {
			t.Errorf("CreateBean().Tags count = %d, want 2", len(got.Tags))
		}
		if len(got.Related) != 1 {
			t.Errorf("CreateBean().Related count = %d, want 1", len(got.Related))
		}
	})
}

func TestMutationUpdateBean(t *testing.T) {
	resolver, core := setupTestResolver(t)
	ctx := context.Background()

	// Create a test bean
	b := &bean.Bean{
		ID:       "update-test",
		Title:    "Original Title",
		Status:   "todo",
		Type:     "task",
		Priority: "normal",
		Body:     "Original body",
		Tags:     []string{"original"},
	}
	core.Create(b)

	t.Run("update single field", func(t *testing.T) {
		mr := resolver.Mutation()
		newStatus := "in-progress"
		input := model.UpdateBeanInput{
			Status: &newStatus,
		}
		got, err := mr.UpdateBean(ctx, "update-test", input)
		if err != nil {
			t.Fatalf("UpdateBean() error = %v", err)
		}
		if got.Status != "in-progress" {
			t.Errorf("UpdateBean().Status = %q, want %q", got.Status, "in-progress")
		}
		// Other fields unchanged
		if got.Title != "Original Title" {
			t.Errorf("UpdateBean().Title = %q, want %q", got.Title, "Original Title")
		}
	})

	t.Run("update multiple fields", func(t *testing.T) {
		mr := resolver.Mutation()
		newTitle := "Updated Title"
		newPriority := "high"
		newBody := "Updated body"
		input := model.UpdateBeanInput{
			Title:    &newTitle,
			Priority: &newPriority,
			Body:     &newBody,
		}
		got, err := mr.UpdateBean(ctx, "update-test", input)
		if err != nil {
			t.Fatalf("UpdateBean() error = %v", err)
		}
		if got.Title != "Updated Title" {
			t.Errorf("UpdateBean().Title = %q, want %q", got.Title, "Updated Title")
		}
		if got.Priority != "high" {
			t.Errorf("UpdateBean().Priority = %q, want %q", got.Priority, "high")
		}
		if got.Body != "Updated body" {
			t.Errorf("UpdateBean().Body = %q, want %q", got.Body, "Updated body")
		}
	})

	t.Run("replace tags", func(t *testing.T) {
		mr := resolver.Mutation()
		input := model.UpdateBeanInput{
			Tags: []string{"new-tag-1", "new-tag-2"},
		}
		got, err := mr.UpdateBean(ctx, "update-test", input)
		if err != nil {
			t.Fatalf("UpdateBean() error = %v", err)
		}
		if len(got.Tags) != 2 {
			t.Errorf("UpdateBean().Tags count = %d, want 2", len(got.Tags))
		}
	})

	t.Run("update nonexistent bean", func(t *testing.T) {
		mr := resolver.Mutation()
		newTitle := "Whatever"
		input := model.UpdateBeanInput{
			Title: &newTitle,
		}
		_, err := mr.UpdateBean(ctx, "nonexistent", input)
		if err == nil {
			t.Error("UpdateBean() expected error for nonexistent bean")
		}
	})
}

func TestMutationAddRemoveBlock(t *testing.T) {
	resolver, core := setupTestResolver(t)
	ctx := context.Background()

	// Create test beans
	b := &bean.Bean{
		ID:     "link-test",
		Title:  "Link Test",
		Status: "todo",
		Type:   "task",
	}
	core.Create(b)

	target := &bean.Bean{
		ID:     "target-1",
		Title:  "Target",
		Status: "todo",
	}
	core.Create(target)

	t.Run("add block", func(t *testing.T) {
		mr := resolver.Mutation()
		got, err := mr.AddBlock(ctx, "link-test", "target-1")
		if err != nil {
			t.Fatalf("AddBlock() error = %v", err)
		}
		if len(got.Blocks) != 1 {
			t.Errorf("AddBlock().Blocks count = %d, want 1", len(got.Blocks))
		}
		if got.Blocks[0] != "target-1" {
			t.Errorf("AddBlock().Blocks[0] = %q, want %q", got.Blocks[0], "target-1")
		}
	})

	t.Run("add another block", func(t *testing.T) {
		// Create another target
		target2 := &bean.Bean{ID: "target-2", Title: "Target 2", Status: "todo"}
		core.Create(target2)

		mr := resolver.Mutation()
		got, err := mr.AddBlock(ctx, "link-test", "target-2")
		if err != nil {
			t.Fatalf("AddBlock() error = %v", err)
		}
		if len(got.Blocks) != 2 {
			t.Errorf("AddBlock().Blocks count = %d, want 2", len(got.Blocks))
		}
	})

	t.Run("remove block", func(t *testing.T) {
		mr := resolver.Mutation()
		got, err := mr.RemoveBlock(ctx, "link-test", "target-1")
		if err != nil {
			t.Fatalf("RemoveBlock() error = %v", err)
		}
		if len(got.Blocks) != 1 {
			t.Errorf("RemoveBlock().Blocks count = %d, want 1", len(got.Blocks))
		}
		if got.Blocks[0] != "target-2" {
			t.Errorf("RemoveBlock().Blocks[0] = %q, want %q", got.Blocks[0], "target-2")
		}
	})

	t.Run("add block to nonexistent bean", func(t *testing.T) {
		mr := resolver.Mutation()
		_, err := mr.AddBlock(ctx, "nonexistent", "whatever")
		if err == nil {
			t.Error("AddBlock() expected error for nonexistent bean")
		}
	})
}

func TestMutationSetMilestone(t *testing.T) {
	resolver, core := setupTestResolver(t)
	ctx := context.Background()

	// Create test beans
	milestone := &bean.Bean{
		ID:     "ms-1",
		Title:  "Milestone 1",
		Status: "todo",
		Type:   "milestone",
	}
	core.Create(milestone)

	task := &bean.Bean{
		ID:     "task-1",
		Title:  "Task 1",
		Status: "todo",
		Type:   "task",
	}
	core.Create(task)

	t.Run("set milestone", func(t *testing.T) {
		mr := resolver.Mutation()
		target := "ms-1"
		got, err := mr.SetMilestone(ctx, "task-1", &target)
		if err != nil {
			t.Fatalf("SetMilestone() error = %v", err)
		}
		if got.Milestone != "ms-1" {
			t.Errorf("SetMilestone().Milestone = %q, want %q", got.Milestone, "ms-1")
		}
	})

	t.Run("clear milestone", func(t *testing.T) {
		mr := resolver.Mutation()
		got, err := mr.SetMilestone(ctx, "task-1", nil)
		if err != nil {
			t.Fatalf("SetMilestone() error = %v", err)
		}
		if got.Milestone != "" {
			t.Errorf("SetMilestone().Milestone = %q, want empty", got.Milestone)
		}
	})
}

func TestMutationDeleteBean(t *testing.T) {
	resolver, core := setupTestResolver(t)
	ctx := context.Background()

	t.Run("delete existing bean", func(t *testing.T) {
		// Create a bean to delete
		b := &bean.Bean{ID: "delete-me", Title: "Delete Me", Status: "todo", Type: "task"}
		core.Create(b)

		mr := resolver.Mutation()
		got, err := mr.DeleteBean(ctx, "delete-me")
		if err != nil {
			t.Fatalf("DeleteBean() error = %v", err)
		}
		if !got {
			t.Error("DeleteBean() = false, want true")
		}

		// Verify it's gone
		qr := resolver.Query()
		bean, _ := qr.Bean(ctx, "delete-me")
		if bean != nil {
			t.Error("Bean still exists after delete")
		}
	})

	t.Run("delete removes incoming links", func(t *testing.T) {
		// Create target bean
		target := &bean.Bean{ID: "target-bean", Title: "Target", Status: "todo", Type: "task"}
		core.Create(target)

		// Create bean that links to target
		linker := &bean.Bean{
			ID:     "linker-bean",
			Title:  "Linker",
			Status: "todo",
			Type:   "task",
			Blocks: []string{"target-bean"},
		}
		core.Create(linker)

		// Delete target - should remove the link from linker
		mr := resolver.Mutation()
		_, err := mr.DeleteBean(ctx, "target-bean")
		if err != nil {
			t.Fatalf("DeleteBean() error = %v", err)
		}

		// Verify linker no longer has the link
		qr := resolver.Query()
		updated, _ := qr.Bean(ctx, "linker-bean")
		if updated == nil {
			t.Fatal("Linker bean was deleted unexpectedly")
		}
		if len(updated.Blocks) != 0 {
			t.Errorf("Linker still has %d blocks, want 0", len(updated.Blocks))
		}
	})

	t.Run("delete nonexistent bean", func(t *testing.T) {
		mr := resolver.Mutation()
		_, err := mr.DeleteBean(ctx, "nonexistent")
		if err == nil {
			t.Error("DeleteBean() expected error for nonexistent bean")
		}
	})
}
