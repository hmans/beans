package graph

import (
	"context"
	"os"
	"path/filepath"
	"strings"
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

	// Test partial ID not found (no prefix matching)
	t.Run("partial ID not found", func(t *testing.T) {
		qr := resolver.Query()
		got, err := qr.Bean(ctx, "test")
		if err != nil {
			t.Fatalf("Bean() error = %v", err)
		}
		if got != nil {
			t.Errorf("Bean() = %v, want nil (partial IDs should not match)", got)
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
		Parent: "parent-1",
	}
	child2 := &bean.Bean{
		ID:     "child-2",
		Title:  "Child 2",
		Status: "todo",
		Parent: "parent-1",
	}
	blocker := &bean.Bean{
		ID:     "blocker-1",
		Title:  "Blocker",
		Status: "todo",
		Blocking: []string{"child-1"},
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
		got, err := br.Children(ctx, parent, nil)
		if err != nil {
			t.Fatalf("Children() error = %v", err)
		}
		if len(got) != 2 {
			t.Errorf("Children() count = %d, want 2", len(got))
		}
	})

	t.Run("blockedBy resolver", func(t *testing.T) {
		br := resolver.Bean()
		got, err := br.BlockedBy(ctx, child1, nil)
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
		got, err := br.Blocking(ctx, blocker, nil)
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
		Parent: "nonexistent",
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
}

func TestQueryBeansWithParentAndBlocks(t *testing.T) {
	resolver, core := setupTestResolver(t)
	ctx := context.Background()

	// Create beans with various relationship configurations
	noRels := &bean.Bean{ID: "no-rels", Title: "No Relationships", Status: "todo"}
	hasParent := &bean.Bean{
		ID:     "has-parent",
		Title:  "Has Parent",
		Status: "todo",
		Parent: "no-rels",
	}
	hasBlocks := &bean.Bean{
		ID:     "has-blocks",
		Title:  "Has Blocks",
		Status: "todo",
		Blocking: []string{"has-parent"},
	}

	core.Create(noRels)
	core.Create(hasParent)
	core.Create(hasBlocks)

	t.Run("filter hasParent", func(t *testing.T) {
		qr := resolver.Query()
		hasParentBool := true
		filter := &model.BeanFilter{
			HasParent: &hasParentBool,
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

	t.Run("filter noParent", func(t *testing.T) {
		qr := resolver.Query()
		noParentBool := true
		filter := &model.BeanFilter{
			NoParent: &noParentBool,
		}
		got, err := qr.Beans(ctx, filter)
		if err != nil {
			t.Fatalf("Beans() error = %v", err)
		}
		if len(got) != 2 {
			t.Errorf("Beans() count = %d, want 2", len(got))
		}
	})

	t.Run("filter hasBlocks", func(t *testing.T) {
		qr := resolver.Query()
		hasBlocksBool := true
		filter := &model.BeanFilter{
			HasBlocking: &hasBlocksBool,
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

	t.Run("filter isBlocked true", func(t *testing.T) {
		qr := resolver.Query()
		isBlockedBool := true
		filter := &model.BeanFilter{
			IsBlocked: &isBlockedBool,
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

	t.Run("filter isBlocked false", func(t *testing.T) {
		qr := resolver.Query()
		isBlockedBool := false
		filter := &model.BeanFilter{
			IsBlocked: &isBlockedBool,
		}
		got, err := qr.Beans(ctx, filter)
		if err != nil {
			t.Fatalf("Beans() error = %v", err)
		}
		// Should return all beans except "has-parent" (which is blocked by "has-blocks")
		if len(got) != 2 {
			t.Errorf("Beans() count = %d, want 2", len(got))
		}
		// Verify "has-parent" is not in results
		for _, b := range got {
			if b.ID == "has-parent" {
				t.Errorf("Beans() should not contain blocked bean 'has-parent'")
			}
		}
	})

	t.Run("filter by parentId", func(t *testing.T) {
		qr := resolver.Query()
		parentID := "no-rels"
		filter := &model.BeanFilter{
			ParentID: &parentID,
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
}

func TestMutationCreateBean(t *testing.T) {
	resolver, core := setupTestResolver(t)
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
		// Create parent and target beans first
		parentBean := &bean.Bean{
			ID:     "some-parent",
			Title:  "Parent Bean",
			Status: "todo",
			Type:   "epic",
		}
		targetBean := &bean.Bean{
			ID:     "some-target",
			Title:  "Target Bean",
			Status: "todo",
			Type:   "task",
		}
		core.Create(parentBean)
		core.Create(targetBean)

		mr := resolver.Mutation()
		beanType := "feature"
		status := "in-progress"
		priority := "high"
		body := "Test body content"
		parent := "some-parent"
		input := model.CreateBeanInput{
			Title:    "Full Bean",
			Type:     &beanType,
			Status:   &status,
			Priority: &priority,
			Body:     &body,
			Tags:     []string{"tag1", "tag2"},
			Parent:   &parent,
			Blocking:   []string{"some-target"},
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
		if got.Parent != "some-parent" {
			t.Errorf("CreateBean().Parent = %q, want %q", got.Parent, "some-parent")
		}
		if len(got.Blocking) != 1 {
			t.Errorf("CreateBean().Blocking count = %d, want 1", len(got.Blocking))
		}
	})
}

func TestMutationCreateBeanWithCustomPrefix(t *testing.T) {
	resolver, _ := setupTestResolver(t)
	ctx := context.Background()

	t.Run("create with custom prefix", func(t *testing.T) {
		mr := resolver.Mutation()
		customPrefix := "SYNC-TASK-"
		input := model.CreateBeanInput{
			Title:  "Custom Prefix Bean",
			Prefix: &customPrefix,
		}
		got, err := mr.CreateBean(ctx, input)
		if err != nil {
			t.Fatalf("CreateBean() error = %v", err)
		}
		if got == nil {
			t.Fatal("CreateBean() returned nil")
		}
		// ID should start with the custom prefix
		if !strings.HasPrefix(got.ID, "SYNC-TASK-") {
			t.Errorf("CreateBean().ID = %q, want prefix %q", got.ID, "SYNC-TASK-")
		}
		// ID should be prefix + 4 chars (default length)
		if len(got.ID) != len("SYNC-TASK-")+4 {
			t.Errorf("CreateBean().ID length = %d, want %d", len(got.ID), len("SYNC-TASK-")+4)
		}
	})

	t.Run("create without prefix uses config default", func(t *testing.T) {
		mr := resolver.Mutation()
		input := model.CreateBeanInput{
			Title: "No Custom Prefix Bean",
		}
		got, err := mr.CreateBean(ctx, input)
		if err != nil {
			t.Fatalf("CreateBean() error = %v", err)
		}
		// Without custom prefix, should use config default (empty in test setup)
		// ID should just be 4 chars
		if len(got.ID) != 4 {
			t.Errorf("CreateBean().ID length = %d, want 4", len(got.ID))
		}
	})

	t.Run("create with empty prefix string uses config default", func(t *testing.T) {
		mr := resolver.Mutation()
		emptyPrefix := ""
		input := model.CreateBeanInput{
			Title:  "Empty Prefix Bean",
			Prefix: &emptyPrefix,
		}
		got, err := mr.CreateBean(ctx, input)
		if err != nil {
			t.Fatalf("CreateBean() error = %v", err)
		}
		// Empty string prefix should fall back to config default
		if len(got.ID) != 4 {
			t.Errorf("CreateBean().ID length = %d, want 4", len(got.ID))
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

func TestMutationSetParent(t *testing.T) {
	resolver, core := setupTestResolver(t)
	ctx := context.Background()

	// Create test beans
	parent := &bean.Bean{ID: "parent-1", Title: "Parent", Status: "todo", Type: "epic"}
	child := &bean.Bean{ID: "child-1", Title: "Child", Status: "todo", Type: "task"}
	core.Create(parent)
	core.Create(child)

	t.Run("set parent", func(t *testing.T) {
		mr := resolver.Mutation()
		parentID := "parent-1"
		got, err := mr.SetParent(ctx, "child-1", &parentID)
		if err != nil {
			t.Fatalf("SetParent() error = %v", err)
		}
		if got.Parent != "parent-1" {
			t.Errorf("SetParent().Parent = %q, want %q", got.Parent, "parent-1")
		}
	})

	t.Run("clear parent", func(t *testing.T) {
		mr := resolver.Mutation()
		got, err := mr.SetParent(ctx, "child-1", nil)
		if err != nil {
			t.Fatalf("SetParent() error = %v", err)
		}
		if got.Parent != "" {
			t.Errorf("SetParent().Parent = %q, want empty", got.Parent)
		}
	})

	t.Run("set parent on nonexistent bean", func(t *testing.T) {
		mr := resolver.Mutation()
		parentID := "parent-1"
		_, err := mr.SetParent(ctx, "nonexistent", &parentID)
		if err == nil {
			t.Error("SetParent() expected error for nonexistent bean")
		}
	})
}

func TestMutationAddRemoveBlocking(t *testing.T) {
	resolver, core := setupTestResolver(t)
	ctx := context.Background()

	// Create test beans
	blocker := &bean.Bean{ID: "blocker-1", Title: "Blocker", Status: "todo", Type: "task"}
	target := &bean.Bean{ID: "target-1", Title: "Target", Status: "todo", Type: "task"}
	core.Create(blocker)
	core.Create(target)

	t.Run("add block", func(t *testing.T) {
		mr := resolver.Mutation()
		got, err := mr.AddBlocking(ctx, "blocker-1", "target-1")
		if err != nil {
			t.Fatalf("AddBlocking() error = %v", err)
		}
		if len(got.Blocking) != 1 {
			t.Errorf("AddBlocking().Blocking count = %d, want 1", len(got.Blocking))
		}
		if got.Blocking[0] != "target-1" {
			t.Errorf("AddBlocking().Blocking[0] = %q, want %q", got.Blocking[0], "target-1")
		}
	})

	t.Run("remove block", func(t *testing.T) {
		mr := resolver.Mutation()
		got, err := mr.RemoveBlocking(ctx, "blocker-1", "target-1")
		if err != nil {
			t.Fatalf("RemoveBlocking() error = %v", err)
		}
		if len(got.Blocking) != 0 {
			t.Errorf("RemoveBlocking().Blocking count = %d, want 0", len(got.Blocking))
		}
	})

	t.Run("add block to nonexistent bean", func(t *testing.T) {
		mr := resolver.Mutation()
		_, err := mr.AddBlocking(ctx, "nonexistent", "target-1")
		if err == nil {
			t.Error("AddBlocking() expected error for nonexistent bean")
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
			Blocking: []string{"target-bean"},
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
		if len(updated.Blocking) != 0 {
			t.Errorf("Linker still has %d blocks, want 0", len(updated.Blocking))
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

func TestRelationshipFieldsWithFilter(t *testing.T) {
	resolver, core := setupTestResolver(t)
	ctx := context.Background()

	// Create a parent (milestone) with multiple children (tasks) of different statuses
	parent := &bean.Bean{
		ID:     "parent-filter-test",
		Title:  "Parent Milestone",
		Type:   "milestone",
		Status: "in-progress",
	}
	child1 := &bean.Bean{
		ID:     "child-todo",
		Title:  "Todo Task",
		Type:   "task",
		Status: "todo",
		Parent: "parent-filter-test",
	}
	child2 := &bean.Bean{
		ID:     "child-completed",
		Title:  "Completed Task",
		Type:   "task",
		Status: "completed",
		Parent: "parent-filter-test",
	}
	child3 := &bean.Bean{
		ID:       "child-inprogress",
		Title:    "In Progress Task",
		Type:     "task",
		Status:   "in-progress",
		Parent:   "parent-filter-test",
		Priority: "high",
	}

	// Create blocking relationships with different types
	blocker1 := &bean.Bean{
		ID:       "blocker-bug",
		Title:    "Blocking Bug",
		Type:     "bug",
		Status:   "todo",
		Blocking: []string{"child-todo"},
	}
	blocker2 := &bean.Bean{
		ID:       "blocker-task",
		Title:    "Blocking Task",
		Type:     "task",
		Status:   "completed",
		Blocking: []string{"child-todo"},
	}

	for _, b := range []*bean.Bean{parent, child1, child2, child3, blocker1, blocker2} {
		if err := core.Create(b); err != nil {
			t.Fatalf("Failed to create bean %s: %v", b.ID, err)
		}
	}

	br := resolver.Bean()

	t.Run("children with status filter", func(t *testing.T) {
		filter := &model.BeanFilter{
			Status: []string{"todo"},
		}
		got, err := br.Children(ctx, parent, filter)
		if err != nil {
			t.Fatalf("Children() error = %v", err)
		}
		if len(got) != 1 {
			t.Errorf("Children(filter status=todo) count = %d, want 1", len(got))
		}
		if len(got) > 0 && got[0].ID != "child-todo" {
			t.Errorf("Children(filter status=todo)[0].ID = %q, want %q", got[0].ID, "child-todo")
		}
	})

	t.Run("children with excludeStatus filter", func(t *testing.T) {
		filter := &model.BeanFilter{
			ExcludeStatus: []string{"completed"},
		}
		got, err := br.Children(ctx, parent, filter)
		if err != nil {
			t.Fatalf("Children() error = %v", err)
		}
		if len(got) != 2 {
			t.Errorf("Children(filter excludeStatus=completed) count = %d, want 2", len(got))
		}
	})

	t.Run("children with priority filter", func(t *testing.T) {
		filter := &model.BeanFilter{
			Priority: []string{"high"},
		}
		got, err := br.Children(ctx, parent, filter)
		if err != nil {
			t.Fatalf("Children() error = %v", err)
		}
		if len(got) != 1 {
			t.Errorf("Children(filter priority=high) count = %d, want 1", len(got))
		}
		if len(got) > 0 && got[0].ID != "child-inprogress" {
			t.Errorf("Children(filter priority=high)[0].ID = %q, want %q", got[0].ID, "child-inprogress")
		}
	})

	t.Run("children with nil filter returns all", func(t *testing.T) {
		got, err := br.Children(ctx, parent, nil)
		if err != nil {
			t.Fatalf("Children() error = %v", err)
		}
		if len(got) != 3 {
			t.Errorf("Children(nil filter) count = %d, want 3", len(got))
		}
	})

	t.Run("blockedBy with type filter", func(t *testing.T) {
		filter := &model.BeanFilter{
			Type: []string{"bug"},
		}
		got, err := br.BlockedBy(ctx, child1, filter)
		if err != nil {
			t.Fatalf("BlockedBy() error = %v", err)
		}
		if len(got) != 1 {
			t.Errorf("BlockedBy(filter type=bug) count = %d, want 1", len(got))
		}
		if len(got) > 0 && got[0].ID != "blocker-bug" {
			t.Errorf("BlockedBy(filter type=bug)[0].ID = %q, want %q", got[0].ID, "blocker-bug")
		}
	})

	t.Run("blockedBy with excludeStatus filter", func(t *testing.T) {
		filter := &model.BeanFilter{
			ExcludeStatus: []string{"completed"},
		}
		got, err := br.BlockedBy(ctx, child1, filter)
		if err != nil {
			t.Fatalf("BlockedBy() error = %v", err)
		}
		if len(got) != 1 {
			t.Errorf("BlockedBy(filter excludeStatus=completed) count = %d, want 1", len(got))
		}
		if len(got) > 0 && got[0].ID != "blocker-bug" {
			t.Errorf("BlockedBy(filter excludeStatus=completed)[0].ID = %q, want %q", got[0].ID, "blocker-bug")
		}
	})

	t.Run("blocking with status filter", func(t *testing.T) {
		filter := &model.BeanFilter{
			Status: []string{"todo"},
		}
		got, err := br.Blocking(ctx, blocker1, filter)
		if err != nil {
			t.Fatalf("Blocking() error = %v", err)
		}
		if len(got) != 1 {
			t.Errorf("Blocking(filter status=todo) count = %d, want 1", len(got))
		}
	})

	t.Run("blocking filter excludes all", func(t *testing.T) {
		filter := &model.BeanFilter{
			Status: []string{"completed"},
		}
		got, err := br.Blocking(ctx, blocker1, filter)
		if err != nil {
			t.Fatalf("Blocking() error = %v", err)
		}
		if len(got) != 0 {
			t.Errorf("Blocking(filter status=completed) count = %d, want 0", len(got))
		}
	})
}

// setupTestResolverWithPrefix creates a test resolver with a configured prefix.
func setupTestResolverWithPrefix(t *testing.T, prefix string) (*Resolver, *beancore.Core) {
	t.Helper()
	tmpDir := t.TempDir()
	beansDir := filepath.Join(tmpDir, ".beans")
	if err := os.MkdirAll(beansDir, 0755); err != nil {
		t.Fatalf("failed to create test .beans dir: %v", err)
	}

	cfg := config.DefaultWithPrefix(prefix)
	core := beancore.New(beansDir, cfg)
	if err := core.Load(); err != nil {
		t.Fatalf("failed to load core: %v", err)
	}

	return &Resolver{Core: core}, core
}

func TestShortIDNormalization(t *testing.T) {
	// Use a prefix so we can test short ID resolution
	resolver, core := setupTestResolverWithPrefix(t, "beans-")
	ctx := context.Background()

	// Create test beans with full IDs (prefix + short ID)
	parent := &bean.Bean{ID: "beans-parent1", Title: "Parent", Status: "todo", Type: "epic"}
	child := &bean.Bean{ID: "beans-child1", Title: "Child", Status: "todo", Type: "task"}
	target := &bean.Bean{ID: "beans-target1", Title: "Target", Status: "todo", Type: "task"}
	core.Create(parent)
	core.Create(child)
	core.Create(target)

	t.Run("SetParent normalizes short ID", func(t *testing.T) {
		mr := resolver.Mutation()
		// Use short ID (without prefix)
		shortParentID := "parent1"
		got, err := mr.SetParent(ctx, "beans-child1", &shortParentID)
		if err != nil {
			t.Fatalf("SetParent() error = %v", err)
		}
		// Should store the full ID, not the short one
		if got.Parent != "beans-parent1" {
			t.Errorf("SetParent().Parent = %q, want %q", got.Parent, "beans-parent1")
		}
	})

	t.Run("AddBlocking normalizes short ID", func(t *testing.T) {
		mr := resolver.Mutation()
		// Use short ID (without prefix)
		got, err := mr.AddBlocking(ctx, "beans-child1", "target1")
		if err != nil {
			t.Fatalf("AddBlocking() error = %v", err)
		}
		// Should store the full ID, not the short one
		if len(got.Blocking) != 1 {
			t.Fatalf("AddBlocking().Blocking count = %d, want 1", len(got.Blocking))
		}
		if got.Blocking[0] != "beans-target1" {
			t.Errorf("AddBlocking().Blocking[0] = %q, want %q", got.Blocking[0], "beans-target1")
		}
	})

	t.Run("RemoveBlocking normalizes short ID", func(t *testing.T) {
		mr := resolver.Mutation()
		// Remove using short ID
		got, err := mr.RemoveBlocking(ctx, "beans-child1", "target1")
		if err != nil {
			t.Fatalf("RemoveBlocking() error = %v", err)
		}
		if len(got.Blocking) != 0 {
			t.Errorf("RemoveBlocking().Blocking count = %d, want 0", len(got.Blocking))
		}
	})

	t.Run("CreateBean normalizes parent short ID", func(t *testing.T) {
		mr := resolver.Mutation()
		beanType := "task"
		shortParentID := "parent1"
		input := model.CreateBeanInput{
			Title:  "New Child",
			Type:   &beanType,
			Parent: &shortParentID,
		}
		got, err := mr.CreateBean(ctx, input)
		if err != nil {
			t.Fatalf("CreateBean() error = %v", err)
		}
		if got.Parent != "beans-parent1" {
			t.Errorf("CreateBean().Parent = %q, want %q", got.Parent, "beans-parent1")
		}
	})

	t.Run("CreateBean normalizes blocking short IDs", func(t *testing.T) {
		mr := resolver.Mutation()
		beanType := "task"
		input := model.CreateBeanInput{
			Title:    "Blocker Bean",
			Type:     &beanType,
			Blocking: []string{"target1"},
		}
		got, err := mr.CreateBean(ctx, input)
		if err != nil {
			t.Fatalf("CreateBean() error = %v", err)
		}
		if len(got.Blocking) != 1 {
			t.Fatalf("CreateBean().Blocking count = %d, want 1", len(got.Blocking))
		}
		if got.Blocking[0] != "beans-target1" {
			t.Errorf("CreateBean().Blocking[0] = %q, want %q", got.Blocking[0], "beans-target1")
		}
	})
}
