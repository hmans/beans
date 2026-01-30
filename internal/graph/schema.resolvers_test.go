package graph

import (
	"context"
	"errors"
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
		ID:       "blocker-1",
		Title:    "Blocker",
		Status:   "todo",
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
		ID:       "has-blocks",
		Title:    "Has Blocks",
		Status:   "todo",
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

func TestIsBlockedFilterWithResolvedBlockers(t *testing.T) {
	resolver, core := setupTestResolver(t)
	ctx := context.Background()

	// Create beans to test blocking with various blocker statuses
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
	// Bean with mixed blockers (one active, one completed)
	mixedBlocker := &bean.Bean{
		ID:       "mixed-blocker",
		Title:    "Mixed Blocker (active)",
		Status:   "in-progress",
		Blocking: []string{"mixed-blocked"},
	}
	mixedBlockerCompleted := &bean.Bean{
		ID:       "mixed-blocker-completed",
		Title:    "Mixed Blocker (completed)",
		Status:   "completed",
		Blocking: []string{"mixed-blocked"},
	}
	mixedBlocked := &bean.Bean{
		ID:     "mixed-blocked",
		Title:  "Mixed Blocked",
		Status: "todo",
	}

	beans := []*bean.Bean{
		activeBlocker, completedBlocker, scrappedBlocker,
		blockedByActive, blockedByCompleted, blockedByScrapped,
		notBlocked, mixedBlocker, mixedBlockerCompleted, mixedBlocked,
	}
	for _, b := range beans {
		if err := core.Create(b); err != nil {
			t.Fatalf("Create error: %v", err)
		}
	}

	t.Run("isBlocked true returns only beans with active blockers", func(t *testing.T) {
		qr := resolver.Query()
		isBlocked := true
		filter := &model.BeanFilter{
			IsBlocked: &isBlocked,
		}
		got, err := qr.Beans(ctx, filter)
		if err != nil {
			t.Fatalf("Beans() error = %v", err)
		}

		// Should only return beans blocked by active blockers
		ids := make(map[string]bool)
		for _, b := range got {
			ids[b.ID] = true
		}

		if !ids["blocked-by-active"] {
			t.Error("expected blocked-by-active in results (has active blocker)")
		}
		if !ids["mixed-blocked"] {
			t.Error("expected mixed-blocked in results (has one active blocker)")
		}
		if ids["blocked-by-completed"] {
			t.Error("blocked-by-completed should NOT be in results (blocker is completed)")
		}
		if ids["blocked-by-scrapped"] {
			t.Error("blocked-by-scrapped should NOT be in results (blocker is scrapped)")
		}
		if ids["not-blocked"] {
			t.Error("not-blocked should NOT be in results (no blockers)")
		}
	})

	t.Run("isBlocked false excludes beans with active blockers", func(t *testing.T) {
		qr := resolver.Query()
		isBlocked := false
		filter := &model.BeanFilter{
			IsBlocked: &isBlocked,
		}
		got, err := qr.Beans(ctx, filter)
		if err != nil {
			t.Fatalf("Beans() error = %v", err)
		}

		ids := make(map[string]bool)
		for _, b := range got {
			ids[b.ID] = true
		}

		// Should include beans with no active blockers
		if !ids["blocked-by-completed"] {
			t.Error("expected blocked-by-completed in results (blocker is completed)")
		}
		if !ids["blocked-by-scrapped"] {
			t.Error("expected blocked-by-scrapped in results (blocker is scrapped)")
		}
		if !ids["not-blocked"] {
			t.Error("expected not-blocked in results (no blockers)")
		}
		// Should exclude beans with active blockers
		if ids["blocked-by-active"] {
			t.Error("blocked-by-active should NOT be in results (has active blocker)")
		}
		if ids["mixed-blocked"] {
			t.Error("mixed-blocked should NOT be in results (has active blocker)")
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
			Blocking: []string{"some-target"},
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
		got, err := mr.SetParent(ctx, "child-1", &parentID, nil)
		if err != nil {
			t.Fatalf("SetParent() error = %v", err)
		}
		if got.Parent != "parent-1" {
			t.Errorf("SetParent().Parent = %q, want %q", got.Parent, "parent-1")
		}
	})

	t.Run("clear parent", func(t *testing.T) {
		mr := resolver.Mutation()
		got, err := mr.SetParent(ctx, "child-1", nil, nil)
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
		_, err := mr.SetParent(ctx, "nonexistent", &parentID, nil)
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
		got, err := mr.AddBlocking(ctx, "blocker-1", "target-1", nil)
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
		got, err := mr.RemoveBlocking(ctx, "blocker-1", "target-1", nil)
		if err != nil {
			t.Fatalf("RemoveBlocking() error = %v", err)
		}
		if len(got.Blocking) != 0 {
			t.Errorf("RemoveBlocking().Blocking count = %d, want 0", len(got.Blocking))
		}
	})

	t.Run("add block to nonexistent bean", func(t *testing.T) {
		mr := resolver.Mutation()
		_, err := mr.AddBlocking(ctx, "nonexistent", "target-1", nil)
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
			ID:       "linker-bean",
			Title:    "Linker",
			Status:   "todo",
			Type:     "task",
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

// setupTestResolverWithRequireIfMatch creates a test resolver with require_if_match enabled.
func setupTestResolverWithRequireIfMatch(t *testing.T) (*Resolver, *beancore.Core) {
	t.Helper()
	tmpDir := t.TempDir()
	beansDir := filepath.Join(tmpDir, ".beans")
	if err := os.MkdirAll(beansDir, 0755); err != nil {
		t.Fatalf("failed to create test .beans dir: %v", err)
	}

	cfg := config.Default()
	cfg.Beans.RequireIfMatch = true
	core := beancore.New(beansDir, cfg)
	if err := core.Load(); err != nil {
		t.Fatalf("failed to load core: %v", err)
	}

	return &Resolver{Core: core}, core
}

func TestETagValidation(t *testing.T) {
	t.Run("update with correct etag succeeds", func(t *testing.T) {
		resolver, core := setupTestResolver(t)
		ctx := context.Background()

		b := &bean.Bean{ID: "etag-test-1", Title: "Test", Status: "todo"}
		core.Create(b)

		// Get current etag
		currentETag := b.ETag()

		mr := resolver.Mutation()
		newTitle := "Updated"
		input := model.UpdateBeanInput{
			Title:   &newTitle,
			IfMatch: &currentETag,
		}
		got, err := mr.UpdateBean(ctx, "etag-test-1", input)
		if err != nil {
			t.Fatalf("UpdateBean() with correct etag error = %v", err)
		}
		if got.Title != "Updated" {
			t.Errorf("UpdateBean().Title = %q, want %q", got.Title, "Updated")
		}
	})

	t.Run("update with incorrect etag fails", func(t *testing.T) {
		resolver, core := setupTestResolver(t)
		ctx := context.Background()

		b := &bean.Bean{ID: "etag-test-2", Title: "Test", Status: "todo"}
		core.Create(b)

		mr := resolver.Mutation()
		newTitle := "Updated"
		wrongETag := "wrongetagvalue1"
		input := model.UpdateBeanInput{
			Title:   &newTitle,
			IfMatch: &wrongETag,
		}
		_, err := mr.UpdateBean(ctx, "etag-test-2", input)
		if err == nil {
			t.Error("UpdateBean() with wrong etag should fail")
		}
		if !strings.Contains(err.Error(), "etag mismatch") {
			t.Errorf("Error should mention etag mismatch, got: %v", err)
		}
	})

	t.Run("update without etag succeeds when not required", func(t *testing.T) {
		resolver, core := setupTestResolver(t)
		ctx := context.Background()

		b := &bean.Bean{ID: "etag-test-3", Title: "Test", Status: "todo"}
		core.Create(b)

		mr := resolver.Mutation()
		newTitle := "Updated"
		input := model.UpdateBeanInput{
			Title: &newTitle,
		}
		got, err := mr.UpdateBean(ctx, "etag-test-3", input)
		if err != nil {
			t.Fatalf("UpdateBean() without etag error = %v", err)
		}
		if got.Title != "Updated" {
			t.Errorf("UpdateBean().Title = %q, want %q", got.Title, "Updated")
		}
	})
}

func TestRequireIfMatchConfig(t *testing.T) {
	t.Run("update without etag fails when require_if_match is true", func(t *testing.T) {
		resolver, core := setupTestResolverWithRequireIfMatch(t)
		ctx := context.Background()

		b := &bean.Bean{ID: "require-etag-1", Title: "Test", Status: "todo"}
		core.Create(b)

		mr := resolver.Mutation()
		newTitle := "Updated"
		input := model.UpdateBeanInput{
			Title: &newTitle,
		}
		_, err := mr.UpdateBean(ctx, "require-etag-1", input)
		if err == nil {
			t.Error("UpdateBean() without etag should fail when require_if_match is true")
		}
		if !strings.Contains(err.Error(), "if-match etag is required") {
			t.Errorf("Error should mention etag is required, got: %v", err)
		}
	})

	t.Run("update with correct etag succeeds when require_if_match is true", func(t *testing.T) {
		resolver, core := setupTestResolverWithRequireIfMatch(t)
		ctx := context.Background()

		b := &bean.Bean{ID: "require-etag-2", Title: "Test", Status: "todo"}
		core.Create(b)

		currentETag := b.ETag()

		mr := resolver.Mutation()
		newTitle := "Updated"
		input := model.UpdateBeanInput{
			Title:   &newTitle,
			IfMatch: &currentETag,
		}
		got, err := mr.UpdateBean(ctx, "require-etag-2", input)
		if err != nil {
			t.Fatalf("UpdateBean() with correct etag error = %v", err)
		}
		if got.Title != "Updated" {
			t.Errorf("UpdateBean().Title = %q, want %q", got.Title, "Updated")
		}
	})

	t.Run("setParent without etag fails when require_if_match is true", func(t *testing.T) {
		resolver, core := setupTestResolverWithRequireIfMatch(t)
		ctx := context.Background()

		parent := &bean.Bean{ID: "req-parent", Title: "Parent", Status: "todo", Type: "epic"}
		child := &bean.Bean{ID: "req-child", Title: "Child", Status: "todo", Type: "task"}
		core.Create(parent)
		core.Create(child)

		mr := resolver.Mutation()
		parentID := "req-parent"
		_, err := mr.SetParent(ctx, "req-child", &parentID, nil)
		if err == nil {
			t.Error("SetParent() without etag should fail when require_if_match is true")
		}
	})

	t.Run("addBlocking without etag fails when require_if_match is true", func(t *testing.T) {
		resolver, core := setupTestResolverWithRequireIfMatch(t)
		ctx := context.Background()

		b1 := &bean.Bean{ID: "req-blocker", Title: "Blocker", Status: "todo"}
		b2 := &bean.Bean{ID: "req-target", Title: "Target", Status: "todo"}
		core.Create(b1)
		core.Create(b2)

		mr := resolver.Mutation()
		_, err := mr.AddBlocking(ctx, "req-blocker", "req-target", nil)
		if err == nil {
			t.Error("AddBlocking() without etag should fail when require_if_match is true")
		}
	})

	t.Run("removeBlocking without etag fails when require_if_match is true", func(t *testing.T) {
		resolver, core := setupTestResolverWithRequireIfMatch(t)
		ctx := context.Background()

		b1 := &bean.Bean{ID: "req-blocker2", Title: "Blocker", Status: "todo", Blocking: []string{"req-target2"}}
		b2 := &bean.Bean{ID: "req-target2", Title: "Target", Status: "todo"}
		core.Create(b1)
		core.Create(b2)

		mr := resolver.Mutation()
		_, err := mr.RemoveBlocking(ctx, "req-blocker2", "req-target2", nil)
		if err == nil {
			t.Error("RemoveBlocking() without etag should fail when require_if_match is true")
		}
	})
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
		got, err := mr.SetParent(ctx, "beans-child1", &shortParentID, nil)
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
		got, err := mr.AddBlocking(ctx, "beans-child1", "target1", nil)
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
		got, err := mr.RemoveBlocking(ctx, "beans-child1", "target1", nil)
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

func TestUpdateBeanWithBodyMod(t *testing.T) {
	resolver, core := setupTestResolver(t)
	ctx := context.Background()

	t.Run("bodyMod with single replacement only", func(t *testing.T) {
		b := &bean.Bean{
			ID:     "bodymod-test-1",
			Title:  "Test",
			Status: "todo",
			Body:   "## Tasks\n- [ ] Task 1\n- [ ] Task 2",
		}
		core.Create(b)

		input := model.UpdateBeanInput{
			BodyMod: &model.BodyModification{
				Replace: []*model.ReplaceOperation{
					{Old: "- [ ] Task 1", New: "- [x] Task 1"},
				},
			},
		}

		got, err := resolver.Mutation().UpdateBean(ctx, "bodymod-test-1", input)
		if err != nil {
			t.Fatalf("UpdateBean() error = %v", err)
		}
		want := "## Tasks\n- [x] Task 1\n- [ ] Task 2"
		if got.Body != want {
			t.Errorf("UpdateBean().Body = %q, want %q", got.Body, want)
		}
	})

	t.Run("bodyMod with append only", func(t *testing.T) {
		b := &bean.Bean{
			ID:     "bodymod-test-2",
			Title:  "Test",
			Status: "todo",
			Body:   "Existing content",
		}
		core.Create(b)

		appendText := "## Notes\n\nNew section"
		input := model.UpdateBeanInput{
			BodyMod: &model.BodyModification{
				Append: &appendText,
			},
		}

		got, err := resolver.Mutation().UpdateBean(ctx, "bodymod-test-2", input)
		if err != nil {
			t.Fatalf("UpdateBean() error = %v", err)
		}
		want := "Existing content\n\n## Notes\n\nNew section"
		if got.Body != want {
			t.Errorf("UpdateBean().Body = %q, want %q", got.Body, want)
		}
	})

	t.Run("bodyMod with replacement and append combined", func(t *testing.T) {
		b := &bean.Bean{
			ID:     "bodymod-test-3",
			Title:  "Test",
			Status: "todo",
			Body:   "## Tasks\n- [ ] Deploy",
		}
		core.Create(b)

		appendText := "## Summary\n\nCompleted"
		input := model.UpdateBeanInput{
			BodyMod: &model.BodyModification{
				Replace: []*model.ReplaceOperation{
					{Old: "- [ ] Deploy", New: "- [x] Deploy"},
				},
				Append: &appendText,
			},
		}

		got, err := resolver.Mutation().UpdateBean(ctx, "bodymod-test-3", input)
		if err != nil {
			t.Fatalf("UpdateBean() error = %v", err)
		}
		want := "## Tasks\n- [x] Deploy\n\n## Summary\n\nCompleted"
		if got.Body != want {
			t.Errorf("UpdateBean().Body = %q, want %q", got.Body, want)
		}
	})

	t.Run("bodyMod with multiple replacements sequential", func(t *testing.T) {
		b := &bean.Bean{
			ID:     "bodymod-test-4",
			Title:  "Test",
			Status: "todo",
			Body:   "- [ ] Task 1\n- [ ] Task 2\n- [ ] Task 3",
		}
		core.Create(b)

		input := model.UpdateBeanInput{
			BodyMod: &model.BodyModification{
				Replace: []*model.ReplaceOperation{
					{Old: "- [ ] Task 1", New: "- [x] Task 1"},
					{Old: "- [ ] Task 2", New: "- [x] Task 2"},
					{Old: "- [ ] Task 3", New: "- [x] Task 3"},
				},
			},
		}

		got, err := resolver.Mutation().UpdateBean(ctx, "bodymod-test-4", input)
		if err != nil {
			t.Fatalf("UpdateBean() error = %v", err)
		}
		want := "- [x] Task 1\n- [x] Task 2\n- [x] Task 3"
		if got.Body != want {
			t.Errorf("UpdateBean().Body = %q, want %q", got.Body, want)
		}
	})

	t.Run("bodyMod with metadata update", func(t *testing.T) {
		b := &bean.Bean{
			ID:     "bodymod-test-5",
			Title:  "Test",
			Status: "todo",
			Body:   "- [ ] Task",
		}
		core.Create(b)

		status := "completed"
		appendText := "## Done"
		input := model.UpdateBeanInput{
			Status: &status,
			BodyMod: &model.BodyModification{
				Replace: []*model.ReplaceOperation{
					{Old: "- [ ] Task", New: "- [x] Task"},
				},
				Append: &appendText,
			},
		}

		got, err := resolver.Mutation().UpdateBean(ctx, "bodymod-test-5", input)
		if err != nil {
			t.Fatalf("UpdateBean() error = %v", err)
		}
		if got.Status != "completed" {
			t.Errorf("UpdateBean().Status = %q, want %q", got.Status, "completed")
		}
		want := "- [x] Task\n\n## Done"
		if got.Body != want {
			t.Errorf("UpdateBean().Body = %q, want %q", got.Body, want)
		}
	})

	t.Run("error when both body and bodyMod provided", func(t *testing.T) {
		b := &bean.Bean{
			ID:     "bodymod-test-6",
			Title:  "Test",
			Status: "todo",
			Body:   "Original",
		}
		core.Create(b)

		bodyText := "New body"
		appendText := "Append"
		input := model.UpdateBeanInput{
			Body: &bodyText,
			BodyMod: &model.BodyModification{
				Append: &appendText,
			},
		}

		_, err := resolver.Mutation().UpdateBean(ctx, "bodymod-test-6", input)
		if err == nil {
			t.Error("UpdateBean() expected error when both body and bodyMod provided")
		}
		if !strings.Contains(err.Error(), "cannot specify both body and bodyMod") {
			t.Errorf("Error should mention mutual exclusivity, got: %v", err)
		}
	})

	t.Run("error when replacement text not found", func(t *testing.T) {
		b := &bean.Bean{
			ID:     "bodymod-test-7",
			Title:  "Test",
			Status: "todo",
			Body:   "Hello world",
		}
		core.Create(b)

		input := model.UpdateBeanInput{
			BodyMod: &model.BodyModification{
				Replace: []*model.ReplaceOperation{
					{Old: "nonexistent", New: "fail"},
				},
			},
		}

		_, err := resolver.Mutation().UpdateBean(ctx, "bodymod-test-7", input)
		if err == nil {
			t.Error("UpdateBean() expected error when replacement text not found")
		}
		if !strings.Contains(err.Error(), "text not found") {
			t.Errorf("Error should mention text not found, got: %v", err)
		}
	})

	t.Run("error when replacement text found multiple times", func(t *testing.T) {
		b := &bean.Bean{
			ID:     "bodymod-test-8",
			Title:  "Test",
			Status: "todo",
			Body:   "foo foo foo",
		}
		core.Create(b)

		input := model.UpdateBeanInput{
			BodyMod: &model.BodyModification{
				Replace: []*model.ReplaceOperation{
					{Old: "foo", New: "bar"},
				},
			},
		}

		_, err := resolver.Mutation().UpdateBean(ctx, "bodymod-test-8", input)
		if err == nil {
			t.Error("UpdateBean() expected error when replacement text found multiple times")
		}
		if !strings.Contains(err.Error(), "found 3 times") {
			t.Errorf("Error should mention multiple matches, got: %v", err)
		}
	})

	t.Run("transactional: later replacement fails, nothing saved", func(t *testing.T) {
		b := &bean.Bean{
			ID:     "bodymod-test-9",
			Title:  "Test",
			Status: "todo",
			Body:   "Task 1\nTask 2",
		}
		core.Create(b)
		originalBody := b.Body

		input := model.UpdateBeanInput{
			BodyMod: &model.BodyModification{
				Replace: []*model.ReplaceOperation{
					{Old: "Task 1", New: "Done 1"},    // This should succeed
					{Old: "nonexistent", New: "fail"}, // This should fail
				},
			},
		}

		_, err := resolver.Mutation().UpdateBean(ctx, "bodymod-test-9", input)
		if err == nil {
			t.Error("UpdateBean() expected error")
		}

		// Verify bean wasn't modified
		updated, _ := core.Get("bodymod-test-9")
		if updated.Body != originalBody {
			t.Errorf("Bean body was modified despite error. Got %q, want %q", updated.Body, originalBody)
		}
	})

	t.Run("empty append is no-op", func(t *testing.T) {
		b := &bean.Bean{
			ID:     "bodymod-test-10",
			Title:  "Test",
			Status: "todo",
			Body:   "Original content",
		}
		core.Create(b)

		emptyAppend := ""
		input := model.UpdateBeanInput{
			BodyMod: &model.BodyModification{
				Append: &emptyAppend,
			},
		}

		got, err := resolver.Mutation().UpdateBean(ctx, "bodymod-test-10", input)
		if err != nil {
			t.Fatalf("UpdateBean() error = %v", err)
		}
		if got.Body != "Original content" {
			t.Errorf("UpdateBean().Body = %q, want %q (no-op for empty append)", got.Body, "Original content")
		}
	})

	t.Run("transactional: later replacement fails, nothing saved", func(t *testing.T) {
		b := &bean.Bean{
			ID:     "bodymod-test-9",
			Title:  "Test",
			Status: "todo",
			Body:   "Task 1\nTask 2",
		}
		core.Create(b)
		originalBody := b.Body

		input := model.UpdateBeanInput{
			BodyMod: &model.BodyModification{
				Replace: []*model.ReplaceOperation{
					{Old: "Task 1", New: "Done 1"},    // This should succeed
					{Old: "nonexistent", New: "fail"}, // This should fail
				},
			},
		}

		_, err := resolver.Mutation().UpdateBean(ctx, "bodymod-test-9", input)
		if err == nil {
			t.Error("UpdateBean() expected error")
		}

		// Verify bean wasn't modified
		updated, _ := core.Get("bodymod-test-9")
		if updated.Body != originalBody {
			t.Errorf("Bean body was modified despite error. Got %q, want %q", updated.Body, originalBody)
		}
	})
}

func TestUpdateBeanWithRelationships(t *testing.T) {
	resolver, core := setupTestResolver(t)
	ctx := context.Background()

	t.Run("atomic update with parent and blocking", func(t *testing.T) {
		epic := &bean.Bean{ID: "epic-1", Title: "Epic", Type: "epic", Status: "todo"}
		task := &bean.Bean{ID: "task-1", Title: "Task", Type: "task", Status: "todo"}
		blocker := &bean.Bean{ID: "blocker-1", Title: "Blocker", Type: "task", Status: "todo"}
		core.Create(epic)
		core.Create(task)
		core.Create(blocker)

		input := model.UpdateBeanInput{
			Status:      stringPtr("in-progress"),
			Parent:      stringPtr("epic-1"),
			AddBlocking: []string{"blocker-1"},
		}

		got, err := resolver.Mutation().UpdateBean(ctx, "task-1", input)
		if err != nil {
			t.Fatalf("UpdateBean() error = %v", err)
		}

		if got.Status != "in-progress" {
			t.Errorf("UpdateBean().Status = %q, want %q", got.Status, "in-progress")
		}
		if got.Parent != "epic-1" {
			t.Errorf("UpdateBean().Parent = %q, want %q", got.Parent, "epic-1")
		}
		if len(got.Blocking) != 1 || got.Blocking[0] != "blocker-1" {
			t.Errorf("UpdateBean().Blocking = %v, want [blocker-1]", got.Blocking)
		}
	})

	t.Run("atomic update with bodyMod and relationships", func(t *testing.T) {
		epic := &bean.Bean{ID: "epic-2", Title: "Epic", Type: "epic", Status: "todo"}
		task := &bean.Bean{ID: "task-2", Title: "Task", Type: "task", Status: "todo", Body: "- [ ] Step 1"}
		blocker := &bean.Bean{ID: "blocker-2", Title: "Blocker", Type: "task", Status: "todo"}
		core.Create(epic)
		core.Create(task)
		core.Create(blocker)

		input := model.UpdateBeanInput{
			Status: stringPtr("completed"),
			Parent: stringPtr("epic-2"),
			BodyMod: &model.BodyModification{
				Replace: []*model.ReplaceOperation{
					{Old: "- [ ] Step 1", New: "- [x] Step 1"},
				},
				Append: stringPtr("## Done"),
			},
			AddBlocking: []string{"blocker-2"},
		}

		got, err := resolver.Mutation().UpdateBean(ctx, "task-2", input)
		if err != nil {
			t.Fatalf("UpdateBean() error = %v", err)
		}

		if got.Status != "completed" {
			t.Errorf("Status = %q, want completed", got.Status)
		}
		if got.Parent != "epic-2" {
			t.Errorf("Parent = %q, want epic-2", got.Parent)
		}
		if !strings.Contains(got.Body, "- [x] Step 1") {
			t.Errorf("Body missing completed task")
		}
		if !strings.Contains(got.Body, "## Done") {
			t.Errorf("Body missing appended content")
		}
		if len(got.Blocking) != 1 {
			t.Errorf("Blocking count = %d, want 1", len(got.Blocking))
		}
	})

	t.Run("parent validation fails for invalid type hierarchy", func(t *testing.T) {
		task1 := &bean.Bean{ID: "task-invalid-1", Title: "Task 1", Type: "task", Status: "todo"}
		task2 := &bean.Bean{ID: "task-invalid-2", Title: "Task 2", Type: "task", Status: "todo"}
		core.Create(task1)
		core.Create(task2)

		input := model.UpdateBeanInput{
			Parent: stringPtr("task-invalid-2"),
		}

		_, err := resolver.Mutation().UpdateBean(ctx, "task-invalid-1", input)
		if err == nil {
			t.Error("UpdateBean() should fail for invalid parent type")
		}
	})

	t.Run("blocking self-reference validation", func(t *testing.T) {
		task := &bean.Bean{ID: "task-self", Title: "Task", Type: "task", Status: "todo"}
		core.Create(task)

		input := model.UpdateBeanInput{
			AddBlocking: []string{"task-self"},
		}

		_, err := resolver.Mutation().UpdateBean(ctx, "task-self", input)
		if err == nil {
			t.Error("UpdateBean() should fail when bean blocks itself")
		}
		if !strings.Contains(err.Error(), "block itself") {
			t.Errorf("Error should mention self-blocking, got: %v", err)
		}
	})

	t.Run("blocking cycle detection", func(t *testing.T) {
		task1 := &bean.Bean{ID: "task-block-1", Title: "Task 1", Type: "task", Status: "todo"}
		task2 := &bean.Bean{ID: "task-block-2", Title: "Task 2", Type: "task", Status: "todo", Blocking: []string{"task-block-1"}}
		core.Create(task1)
		core.Create(task2)

		// Try to make task-1 block task-2 (would create cycle)
		input := model.UpdateBeanInput{
			AddBlocking: []string{"task-block-2"},
		}

		_, err := resolver.Mutation().UpdateBean(ctx, "task-block-1", input)
		if err == nil {
			t.Error("UpdateBean() should fail when creating blocking cycle")
		}
		if !strings.Contains(err.Error(), "cycle") {
			t.Errorf("Error should mention cycle, got: %v", err)
		}
	})

	t.Run("blocking target not found", func(t *testing.T) {
		task := &bean.Bean{ID: "task-notfound", Title: "Task", Type: "task", Status: "todo"}
		core.Create(task)

		input := model.UpdateBeanInput{
			AddBlocking: []string{"nonexistent"},
		}

		_, err := resolver.Mutation().UpdateBean(ctx, "task-notfound", input)
		if err == nil {
			t.Error("UpdateBean() should fail when blocking target doesn't exist")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("Error should mention not found, got: %v", err)
		}
	})

	t.Run("remove blocking relationships", func(t *testing.T) {
		task := &bean.Bean{ID: "task-remove-1", Title: "Task", Type: "task", Status: "todo", Blocking: []string{"other-1", "other-2"}}
		other1 := &bean.Bean{ID: "other-1", Title: "Other 1", Type: "task", Status: "todo"}
		other2 := &bean.Bean{ID: "other-2", Title: "Other 2", Type: "task", Status: "todo"}
		core.Create(task)
		core.Create(other1)
		core.Create(other2)

		input := model.UpdateBeanInput{
			RemoveBlocking: []string{"other-1"},
		}

		got, err := resolver.Mutation().UpdateBean(ctx, "task-remove-1", input)
		if err != nil {
			t.Fatalf("UpdateBean() error = %v", err)
		}

		if len(got.Blocking) != 1 || got.Blocking[0] != "other-2" {
			t.Errorf("Blocking = %v, want [other-2]", got.Blocking)
		}
	})

	t.Run("blockedBy self-reference validation", func(t *testing.T) {
		task := &bean.Bean{ID: "task-blockedby-self", Title: "Task", Type: "task", Status: "todo"}
		core.Create(task)

		input := model.UpdateBeanInput{
			AddBlockedBy: []string{"task-blockedby-self"},
		}

		_, err := resolver.Mutation().UpdateBean(ctx, "task-blockedby-self", input)
		if err == nil {
			t.Error("UpdateBean() should fail when bean is blocked by itself")
		}
		if !strings.Contains(err.Error(), "blocked by itself") {
			t.Errorf("Error should mention self-blocking, got: %v", err)
		}
	})

	t.Run("blockedBy target not found", func(t *testing.T) {
		task := &bean.Bean{ID: "task-blockedby-notfound", Title: "Task", Type: "task", Status: "todo"}
		core.Create(task)

		input := model.UpdateBeanInput{
			AddBlockedBy: []string{"nonexistent"},
		}

		_, err := resolver.Mutation().UpdateBean(ctx, "task-blockedby-notfound", input)
		if err == nil {
			t.Error("UpdateBean() should fail when blocker doesn't exist")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("Error should mention not found, got: %v", err)
		}
	})

	t.Run("combined add and remove operations", func(t *testing.T) {
		task := &bean.Bean{ID: "task-combined", Title: "Task", Type: "task", Status: "todo", Blocking: []string{"old-1"}}
		old1 := &bean.Bean{ID: "old-1", Title: "Old", Type: "task", Status: "todo"}
		new1 := &bean.Bean{ID: "new-1", Title: "New", Type: "task", Status: "todo"}
		core.Create(task)
		core.Create(old1)
		core.Create(new1)

		input := model.UpdateBeanInput{
			RemoveBlocking: []string{"old-1"},
			AddBlocking:    []string{"new-1"},
		}

		got, err := resolver.Mutation().UpdateBean(ctx, "task-combined", input)
		if err != nil {
			t.Fatalf("UpdateBean() error = %v", err)
		}

		if len(got.Blocking) != 1 || got.Blocking[0] != "new-1" {
			t.Errorf("Blocking = %v, want [new-1]", got.Blocking)
		}
	})

	t.Run("blockedBy cycle detection", func(t *testing.T) {
		task1 := &bean.Bean{ID: "task-blockedby-cycle-1", Title: "Task 1", Type: "task", Status: "todo"}
		task2 := &bean.Bean{ID: "task-blockedby-cycle-2", Title: "Task 2", Type: "task", Status: "todo", BlockedBy: []string{"task-blockedby-cycle-1"}}
		core.Create(task1)
		core.Create(task2)

		// Try to make task-1 blocked by task-2 (would create cycle)
		input := model.UpdateBeanInput{
			AddBlockedBy: []string{"task-blockedby-cycle-2"},
		}

		_, err := resolver.Mutation().UpdateBean(ctx, "task-blockedby-cycle-1", input)
		if err == nil {
			t.Error("UpdateBean() should fail when creating blockedBy cycle")
		}
		if !strings.Contains(err.Error(), "cycle") {
			t.Errorf("Error should mention cycle, got: %v", err)
		}
	})

	t.Run("remove parent", func(t *testing.T) {
		epic := &bean.Bean{ID: "epic-parent-remove", Title: "Epic", Type: "epic", Status: "todo"}
		task := &bean.Bean{ID: "task-parent-remove", Title: "Task", Type: "task", Status: "todo", Parent: "epic-parent-remove"}
		core.Create(epic)
		core.Create(task)

		// Remove parent by setting to empty string
		emptyParent := ""
		input := model.UpdateBeanInput{
			Parent: &emptyParent,
		}

		got, err := resolver.Mutation().UpdateBean(ctx, "task-parent-remove", input)
		if err != nil {
			t.Fatalf("UpdateBean() error = %v", err)
		}

		if got.Parent != "" {
			t.Errorf("Parent = %q, want empty string", got.Parent)
		}
	})

	t.Run("remove blockedBy relationships", func(t *testing.T) {
		task := &bean.Bean{ID: "task-remove-blockedby", Title: "Task", Type: "task", Status: "todo", BlockedBy: []string{"blocker-1", "blocker-2"}}
		blocker1 := &bean.Bean{ID: "blocker-1", Title: "Blocker 1", Type: "task", Status: "todo"}
		blocker2 := &bean.Bean{ID: "blocker-2", Title: "Blocker 2", Type: "task", Status: "todo"}
		core.Create(task)
		core.Create(blocker1)
		core.Create(blocker2)

		input := model.UpdateBeanInput{
			RemoveBlockedBy: []string{"blocker-1"},
		}

		got, err := resolver.Mutation().UpdateBean(ctx, "task-remove-blockedby", input)
		if err != nil {
			t.Fatalf("UpdateBean() error = %v", err)
		}

		if len(got.BlockedBy) != 1 || got.BlockedBy[0] != "blocker-2" {
			t.Errorf("BlockedBy = %v, want [blocker-2]", got.BlockedBy)
		}
	})

	t.Run("multiple blocking additions", func(t *testing.T) {
		task := &bean.Bean{ID: "task-multi-blocking", Title: "Task", Type: "task", Status: "todo"}
		target1 := &bean.Bean{ID: "target-1", Title: "Target 1", Type: "task", Status: "todo"}
		target2 := &bean.Bean{ID: "target-2", Title: "Target 2", Type: "task", Status: "todo"}
		core.Create(task)
		core.Create(target1)
		core.Create(target2)

		input := model.UpdateBeanInput{
			AddBlocking: []string{"target-1", "target-2"},
		}

		got, err := resolver.Mutation().UpdateBean(ctx, "task-multi-blocking", input)
		if err != nil {
			t.Fatalf("UpdateBean() error = %v", err)
		}

		if len(got.Blocking) != 2 {
			t.Errorf("Blocking count = %d, want 2", len(got.Blocking))
		}
	})

	t.Run("all relationship types combined", func(t *testing.T) {
		epic := &bean.Bean{ID: "epic-all", Title: "Epic", Type: "epic", Status: "todo"}
		task := &bean.Bean{ID: "task-all", Title: "Task", Type: "task", Status: "todo", Blocking: []string{"old-blocking"}}
		blocker := &bean.Bean{ID: "new-blocker", Title: "Blocker", Type: "task", Status: "todo"}
		blocked := &bean.Bean{ID: "new-blocked", Title: "Blocked", Type: "task", Status: "todo"}
		oldBlocking := &bean.Bean{ID: "old-blocking", Title: "Old Blocking", Type: "task", Status: "todo"}
		core.Create(epic)
		core.Create(task)
		core.Create(blocker)
		core.Create(blocked)
		core.Create(oldBlocking)

		input := model.UpdateBeanInput{
			Status:         stringPtr("in-progress"),
			Parent:         stringPtr("epic-all"),
			AddBlocking:    []string{"new-blocked"},
			RemoveBlocking: []string{"old-blocking"},
			AddBlockedBy:   []string{"new-blocker"},
		}

		got, err := resolver.Mutation().UpdateBean(ctx, "task-all", input)
		if err != nil {
			t.Fatalf("UpdateBean() error = %v", err)
		}

		if got.Status != "in-progress" {
			t.Errorf("Status = %q, want in-progress", got.Status)
		}
		if got.Parent != "epic-all" {
			t.Errorf("Parent = %q, want epic-all", got.Parent)
		}
		if len(got.Blocking) != 1 || got.Blocking[0] != "new-blocked" {
			t.Errorf("Blocking = %v, want [new-blocked]", got.Blocking)
		}
		if len(got.BlockedBy) != 1 || got.BlockedBy[0] != "new-blocker" {
			t.Errorf("BlockedBy = %v, want [new-blocker]", got.BlockedBy)
		}
	})
}

// Helper function for tests
func stringPtr(s string) *string {
	return &s
}

func TestBlockedByCycleDetection(t *testing.T) {
	resolver, core := setupTestResolver(t)
	ctx := context.Background()

	t.Run("blocked_by self-reference fails", func(t *testing.T) {
		b := &bean.Bean{ID: "self-ref", Title: "Self Reference", Status: "todo"}
		core.Create(b)

		mr := resolver.Mutation()
		_, err := mr.AddBlockedBy(ctx, "self-ref", "self-ref", nil)
		if err == nil {
			t.Error("AddBlockedBy() should fail for self-reference")
		}
		if !strings.Contains(err.Error(), "blocked by itself") {
			t.Errorf("Error should mention self-reference, got: %v", err)
		}
	})

	t.Run("blocked_by cycle via blocked_by only is detected", func(t *testing.T) {
		// This tests the scenario where cycles are created using only blocked_by
		a := &bean.Bean{ID: "cycle-a", Title: "Bean A", Status: "todo"}
		b := &bean.Bean{ID: "cycle-b", Title: "Bean B", Status: "todo"}
		core.Create(a)
		core.Create(b)

		mr := resolver.Mutation()

		// A is blocked by B (B → A)
		_, err := mr.AddBlockedBy(ctx, "cycle-a", "cycle-b", nil)
		if err != nil {
			t.Fatalf("AddBlockedBy(A, B) error = %v", err)
		}

		// B is blocked by A (A → B) - should create cycle A → B → A
		_, err = mr.AddBlockedBy(ctx, "cycle-b", "cycle-a", nil)
		if err == nil {
			t.Error("AddBlockedBy(B, A) should fail - would create cycle")
		}
		if !strings.Contains(err.Error(), "cycle") {
			t.Errorf("Error should mention cycle, got: %v", err)
		}
	})

	t.Run("blocked_by cycle via blocking is detected", func(t *testing.T) {
		// A blocks B, then B is blocked_by A creates a conflict
		a := &bean.Bean{ID: "cross-a", Title: "Bean A", Status: "todo"}
		b := &bean.Bean{ID: "cross-b", Title: "Bean B", Status: "todo"}
		core.Create(a)
		core.Create(b)

		mr := resolver.Mutation()

		// A blocks B (A → B)
		_, err := mr.AddBlocking(ctx, "cross-a", "cross-b", nil)
		if err != nil {
			t.Fatalf("AddBlocking(A, B) error = %v", err)
		}

		// A is blocked by B (B → A) - should create cycle
		_, err = mr.AddBlockedBy(ctx, "cross-a", "cross-b", nil)
		if err == nil {
			t.Error("AddBlockedBy(A, B) should fail - would create cycle")
		}
		if !strings.Contains(err.Error(), "cycle") {
			t.Errorf("Error should mention cycle, got: %v", err)
		}
	})

	t.Run("blocking cycle via blocked_by is detected", func(t *testing.T) {
		// A is blocked_by B, then A blocking B creates a conflict
		a := &bean.Bean{ID: "cross2-a", Title: "Bean A", Status: "todo"}
		b := &bean.Bean{ID: "cross2-b", Title: "Bean B", Status: "todo"}
		core.Create(a)
		core.Create(b)

		mr := resolver.Mutation()

		// A is blocked by B (B → A)
		_, err := mr.AddBlockedBy(ctx, "cross2-a", "cross2-b", nil)
		if err != nil {
			t.Fatalf("AddBlockedBy(A, B) error = %v", err)
		}

		// A blocks B (A → B) - should create cycle
		_, err = mr.AddBlocking(ctx, "cross2-a", "cross2-b", nil)
		if err == nil {
			t.Error("AddBlocking(A, B) should fail - would create cycle")
		}
		if !strings.Contains(err.Error(), "cycle") {
			t.Errorf("Error should mention cycle, got: %v", err)
		}
	})

	t.Run("blocker bean not found fails", func(t *testing.T) {
		a := &bean.Bean{ID: "exists-a", Title: "Bean A", Status: "todo"}
		core.Create(a)

		mr := resolver.Mutation()
		_, err := mr.AddBlockedBy(ctx, "exists-a", "nonexistent", nil)
		if err == nil {
			t.Error("AddBlockedBy() should fail when blocker doesn't exist")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("Error should mention not found, got: %v", err)
		}
	})
}

func TestCreateBeanBlockedByValidation(t *testing.T) {
	resolver, core := setupTestResolver(t)
	ctx := context.Background()

	t.Run("create with blocked_by referencing nonexistent bean fails", func(t *testing.T) {
		mr := resolver.Mutation()
		input := model.CreateBeanInput{
			Title:     "New Bean",
			BlockedBy: []string{"nonexistent"},
		}
		_, err := mr.CreateBean(ctx, input)
		if err == nil {
			t.Error("CreateBean() should fail when blocked_by references nonexistent bean")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("Error should mention not found, got: %v", err)
		}
	})

	t.Run("create with blocking referencing nonexistent bean fails", func(t *testing.T) {
		mr := resolver.Mutation()
		input := model.CreateBeanInput{
			Title:    "New Bean",
			Blocking: []string{"nonexistent"},
		}
		_, err := mr.CreateBean(ctx, input)
		if err == nil {
			t.Error("CreateBean() should fail when blocking references nonexistent bean")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("Error should mention not found, got: %v", err)
		}
	})

	t.Run("create with same bean in both blocking and blocked_by fails", func(t *testing.T) {
		target := &bean.Bean{ID: "target-bean", Title: "Target", Status: "todo"}
		core.Create(target)

		mr := resolver.Mutation()
		input := model.CreateBeanInput{
			Title:     "Cyclic Bean",
			Blocking:  []string{"target-bean"},
			BlockedBy: []string{"target-bean"},
		}
		_, err := mr.CreateBean(ctx, input)
		if err == nil {
			t.Error("CreateBean() should fail when same bean is in both blocking and blocked_by")
		}
		if !strings.Contains(err.Error(), "cycle") {
			t.Errorf("Error should mention cycle, got: %v", err)
		}
	})

	t.Run("create with valid blocked_by succeeds", func(t *testing.T) {
		blocker := &bean.Bean{ID: "valid-blocker", Title: "Blocker", Status: "todo"}
		core.Create(blocker)

		mr := resolver.Mutation()
		input := model.CreateBeanInput{
			Title:     "Blocked Bean",
			BlockedBy: []string{"valid-blocker"},
		}
		got, err := mr.CreateBean(ctx, input)
		if err != nil {
			t.Fatalf("CreateBean() error = %v", err)
		}
		if len(got.BlockedBy) != 1 {
			t.Errorf("CreateBean().BlockedBy count = %d, want 1", len(got.BlockedBy))
		}
		if got.BlockedBy[0] != "valid-blocker" {
			t.Errorf("CreateBean().BlockedBy[0] = %q, want %q", got.BlockedBy[0], "valid-blocker")
		}
	})
}

func TestMutationAddRemoveBlockedBy(t *testing.T) {
	resolver, core := setupTestResolver(t)
	ctx := context.Background()

	// Create test beans
	blocked := &bean.Bean{ID: "blocked-1", Title: "Blocked", Status: "todo"}
	blocker := &bean.Bean{ID: "blocker-1", Title: "Blocker", Status: "todo"}
	core.Create(blocked)
	core.Create(blocker)

	t.Run("add blocked_by", func(t *testing.T) {
		mr := resolver.Mutation()
		got, err := mr.AddBlockedBy(ctx, "blocked-1", "blocker-1", nil)
		if err != nil {
			t.Fatalf("AddBlockedBy() error = %v", err)
		}
		if len(got.BlockedBy) != 1 {
			t.Errorf("AddBlockedBy().BlockedBy count = %d, want 1", len(got.BlockedBy))
		}
		if got.BlockedBy[0] != "blocker-1" {
			t.Errorf("AddBlockedBy().BlockedBy[0] = %q, want %q", got.BlockedBy[0], "blocker-1")
		}
	})

	t.Run("remove blocked_by", func(t *testing.T) {
		mr := resolver.Mutation()
		got, err := mr.RemoveBlockedBy(ctx, "blocked-1", "blocker-1", nil)
		if err != nil {
			t.Fatalf("RemoveBlockedBy() error = %v", err)
		}
		if len(got.BlockedBy) != 0 {
			t.Errorf("RemoveBlockedBy().BlockedBy count = %d, want 0", len(got.BlockedBy))
		}
	})

	t.Run("add blocked_by to nonexistent bean fails", func(t *testing.T) {
		mr := resolver.Mutation()
		_, err := mr.AddBlockedBy(ctx, "nonexistent", "blocker-1", nil)
		if err == nil {
			t.Error("AddBlockedBy() expected error for nonexistent bean")
		}
	})
}

func TestUpdateBeanWithETag(t *testing.T) {
	resolver, core := setupTestResolver(t)
	ctx := context.Background()

	t.Run("update with correct etag succeeds", func(t *testing.T) {
		b := &bean.Bean{
			ID:     "etag-update-1",
			Title:  "Test",
			Status: "todo",
		}
		core.Create(b)

		currentETag := b.ETag()
		newTitle := "Updated"
		input := model.UpdateBeanInput{
			Title:   &newTitle,
			IfMatch: &currentETag,
		}

		got, err := resolver.Mutation().UpdateBean(ctx, "etag-update-1", input)
		if err != nil {
			t.Fatalf("UpdateBean() with correct etag failed: %v", err)
		}
		if got.Title != "Updated" {
			t.Errorf("UpdateBean().Title = %q, want %q", got.Title, "Updated")
		}
	})

	t.Run("update with wrong etag fails", func(t *testing.T) {
		b := &bean.Bean{
			ID:     "etag-update-2",
			Title:  "Test",
			Status: "todo",
		}
		core.Create(b)

		wrongETag := "wrongetag123"
		newTitle := "Should Fail"
		input := model.UpdateBeanInput{
			Title:   &newTitle,
			IfMatch: &wrongETag,
		}

		_, err := resolver.Mutation().UpdateBean(ctx, "etag-update-2", input)
		if err == nil {
			t.Error("UpdateBean() with wrong etag should fail")
		}

		var mismatchErr *beancore.ETagMismatchError
		if !errors.As(err, &mismatchErr) {
			t.Errorf("Expected ETagMismatchError, got %T: %v", err, err)
		}
	})
}

func TestSetParentWithETag(t *testing.T) {
	resolver, core := setupTestResolver(t)
	ctx := context.Background()

	// Create parent
	parent := &bean.Bean{
		ID:     "parent-etag",
		Title:  "Parent",
		Status: "todo",
		Type:   "epic",
	}
	core.Create(parent)

	t.Run("setParent with correct etag succeeds", func(t *testing.T) {
		child := &bean.Bean{
			ID:     "child-etag-1",
			Title:  "Child",
			Status: "todo",
			Type:   "task",
		}
		core.Create(child)

		currentETag := child.ETag()
		parentID := "parent-etag"

		got, err := resolver.Mutation().SetParent(ctx, "child-etag-1", &parentID, &currentETag)
		if err != nil {
			t.Fatalf("SetParent() with correct etag failed: %v", err)
		}
		if got.Parent != "parent-etag" {
			t.Errorf("SetParent().Parent = %q, want %q", got.Parent, "parent-etag")
		}
	})

	t.Run("setParent with wrong etag fails", func(t *testing.T) {
		child := &bean.Bean{
			ID:     "child-etag-2",
			Title:  "Child",
			Status: "todo",
			Type:   "task",
		}
		core.Create(child)

		wrongETag := "wrongetag123"
		parentID := "parent-etag"

		_, err := resolver.Mutation().SetParent(ctx, "child-etag-2", &parentID, &wrongETag)
		if err == nil {
			t.Error("SetParent() with wrong etag should fail")
		}

		var mismatchErr *beancore.ETagMismatchError
		if !errors.As(err, &mismatchErr) {
			t.Errorf("Expected ETagMismatchError, got %T: %v", err, err)
		}
	})
}

func TestAddBlockingWithETag(t *testing.T) {
	resolver, core := setupTestResolver(t)
	ctx := context.Background()

	// Create target bean
	target := &bean.Bean{
		ID:     "target-etag",
		Title:  "Target",
		Status: "todo",
		Type:   "task",
	}
	core.Create(target)

	t.Run("addBlocking with correct etag succeeds", func(t *testing.T) {
		blocker := &bean.Bean{
			ID:     "blocker-etag-1",
			Title:  "Blocker",
			Status: "todo",
			Type:   "task",
		}
		core.Create(blocker)

		currentETag := blocker.ETag()

		got, err := resolver.Mutation().AddBlocking(ctx, "blocker-etag-1", "target-etag", &currentETag)
		if err != nil {
			t.Fatalf("AddBlocking() with correct etag failed: %v", err)
		}
		if len(got.Blocking) != 1 || got.Blocking[0] != "target-etag" {
			t.Errorf("AddBlocking().Blocking = %v, want [target-etag]", got.Blocking)
		}
	})

	t.Run("addBlocking with wrong etag fails", func(t *testing.T) {
		blocker := &bean.Bean{
			ID:     "blocker-etag-2",
			Title:  "Blocker",
			Status: "todo",
			Type:   "task",
		}
		core.Create(blocker)

		wrongETag := "wrongetag123"

		_, err := resolver.Mutation().AddBlocking(ctx, "blocker-etag-2", "target-etag", &wrongETag)
		if err == nil {
			t.Error("AddBlocking() with wrong etag should fail")
		}

		var mismatchErr *beancore.ETagMismatchError
		if !errors.As(err, &mismatchErr) {
			t.Errorf("Expected ETagMismatchError, got %T: %v", err, err)
		}
	})
}

func TestRemoveBlockingWithETag(t *testing.T) {
	resolver, core := setupTestResolver(t)
	ctx := context.Background()

	// Create target bean
	target := &bean.Bean{
		ID:     "target-rm-etag",
		Title:  "Target",
		Status: "todo",
		Type:   "task",
	}
	core.Create(target)

	t.Run("removeBlocking with correct etag succeeds", func(t *testing.T) {
		blocker := &bean.Bean{
			ID:       "blocker-rm-etag-1",
			Title:    "Blocker",
			Status:   "todo",
			Type:     "task",
			Blocking: []string{"target-rm-etag"},
		}
		core.Create(blocker)

		currentETag := blocker.ETag()

		got, err := resolver.Mutation().RemoveBlocking(ctx, "blocker-rm-etag-1", "target-rm-etag", &currentETag)
		if err != nil {
			t.Fatalf("RemoveBlocking() with correct etag failed: %v", err)
		}
		if len(got.Blocking) != 0 {
			t.Errorf("RemoveBlocking().Blocking = %v, want []", got.Blocking)
		}
	})

	t.Run("removeBlocking with wrong etag fails", func(t *testing.T) {
		blocker := &bean.Bean{
			ID:       "blocker-rm-etag-2",
			Title:    "Blocker",
			Status:   "todo",
			Type:     "task",
			Blocking: []string{"target-rm-etag"},
		}
		core.Create(blocker)

		wrongETag := "wrongetag123"

		_, err := resolver.Mutation().RemoveBlocking(ctx, "blocker-rm-etag-2", "target-rm-etag", &wrongETag)
		if err == nil {
			t.Error("RemoveBlocking() with wrong etag should fail")
		}

		var mismatchErr *beancore.ETagMismatchError
		if !errors.As(err, &mismatchErr) {
			t.Errorf("Expected ETagMismatchError, got %T: %v", err, err)
		}
	})
}

