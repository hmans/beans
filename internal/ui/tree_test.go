package ui

import (
	"testing"

	"github.com/hmans/beans/internal/bean"
)

func TestBuildTree(t *testing.T) {
	// Create test beans with parent relationships:
	// milestone1
	//   └── epic1
	//       └── task1
	// task2 (orphan)

	milestone1 := &bean.Bean{ID: "m1", Title: "Milestone 1", Type: "milestone"}
	epic1 := &bean.Bean{ID: "e1", Title: "Epic 1", Type: "epic", Milestone: "m1"}
	task1 := &bean.Bean{ID: "t1", Title: "Task 1", Type: "task", Epic: "e1"}
	task2 := &bean.Bean{ID: "t2", Title: "Task 2", Type: "task"} // orphan

	allBeans := []*bean.Bean{milestone1, epic1, task1, task2}

	// Identity sort function (no sorting)
	noSort := func(b []*bean.Bean) {}

	t.Run("all beans matched", func(t *testing.T) {
		tree := BuildTree(allBeans, allBeans, noSort)

		// Should have 2 root nodes: milestone1 and task2
		if len(tree) != 2 {
			t.Errorf("expected 2 root nodes, got %d", len(tree))
		}

		// Find milestone node
		var milestoneNode *TreeNode
		for _, n := range tree {
			if n.Bean.ID == "m1" {
				milestoneNode = n
				break
			}
		}
		if milestoneNode == nil {
			t.Fatal("milestone node not found")
		}
		if !milestoneNode.Matched {
			t.Error("milestone should be marked as matched")
		}

		// Milestone should have epic as child
		if len(milestoneNode.Children) != 1 {
			t.Errorf("milestone should have 1 child, got %d", len(milestoneNode.Children))
		}
		epicNode := milestoneNode.Children[0]
		if epicNode.Bean.ID != "e1" {
			t.Errorf("expected epic child, got %s", epicNode.Bean.ID)
		}

		// Epic should have task as child
		if len(epicNode.Children) != 1 {
			t.Errorf("epic should have 1 child, got %d", len(epicNode.Children))
		}
		taskNode := epicNode.Children[0]
		if taskNode.Bean.ID != "t1" {
			t.Errorf("expected task child, got %s", taskNode.Bean.ID)
		}
	})

	t.Run("filter leaf only - ancestors included", func(t *testing.T) {
		// Only task1 matched, but ancestors should be included
		matchedBeans := []*bean.Bean{task1}
		tree := BuildTree(matchedBeans, allBeans, noSort)

		// Should have 1 root: milestone (as ancestor)
		if len(tree) != 1 {
			t.Errorf("expected 1 root node, got %d", len(tree))
		}

		milestoneNode := tree[0]
		if milestoneNode.Bean.ID != "m1" {
			t.Errorf("expected milestone as root, got %s", milestoneNode.Bean.ID)
		}
		if milestoneNode.Matched {
			t.Error("milestone should NOT be marked as matched (it's an ancestor)")
		}

		// Should have epic as child (also ancestor)
		if len(milestoneNode.Children) != 1 {
			t.Fatalf("milestone should have 1 child, got %d", len(milestoneNode.Children))
		}
		epicNode := milestoneNode.Children[0]
		if epicNode.Matched {
			t.Error("epic should NOT be marked as matched (it's an ancestor)")
		}

		// Task should be matched
		if len(epicNode.Children) != 1 {
			t.Fatalf("epic should have 1 child, got %d", len(epicNode.Children))
		}
		taskNode := epicNode.Children[0]
		if !taskNode.Matched {
			t.Error("task should be marked as matched")
		}
	})

	t.Run("filter middle - ancestors included", func(t *testing.T) {
		// Only epic1 matched
		matchedBeans := []*bean.Bean{epic1}
		tree := BuildTree(matchedBeans, allBeans, noSort)

		// Should have 1 root: milestone (ancestor)
		if len(tree) != 1 {
			t.Errorf("expected 1 root node, got %d", len(tree))
		}

		milestoneNode := tree[0]
		if milestoneNode.Matched {
			t.Error("milestone should NOT be marked as matched")
		}

		epicNode := milestoneNode.Children[0]
		if !epicNode.Matched {
			t.Error("epic should be marked as matched")
		}

		// Epic should have no children (task1 was not matched)
		if len(epicNode.Children) != 0 {
			t.Errorf("epic should have 0 children (task not matched), got %d", len(epicNode.Children))
		}
	})

	t.Run("orphan bean", func(t *testing.T) {
		matchedBeans := []*bean.Bean{task2}
		tree := BuildTree(matchedBeans, allBeans, noSort)

		if len(tree) != 1 {
			t.Errorf("expected 1 root node, got %d", len(tree))
		}
		if tree[0].Bean.ID != "t2" {
			t.Errorf("expected task2 as root, got %s", tree[0].Bean.ID)
		}
		if !tree[0].Matched {
			t.Error("task2 should be marked as matched")
		}
	})

	t.Run("broken parent link", func(t *testing.T) {
		// Bean with epic that doesn't exist
		brokenBean := &bean.Bean{ID: "broken", Title: "Broken", Epic: "nonexistent"}
		matchedBeans := []*bean.Bean{brokenBean}
		allBeansWithBroken := append(allBeans, brokenBean)

		tree := BuildTree(matchedBeans, allBeansWithBroken, noSort)

		// Should be treated as root (epic not found)
		if len(tree) != 1 {
			t.Errorf("expected 1 root node, got %d", len(tree))
		}
		if tree[0].Bean.ID != "broken" {
			t.Errorf("expected broken bean as root, got %s", tree[0].Bean.ID)
		}
	})
}

func TestTreeNodeToJSON(t *testing.T) {
	b := &bean.Bean{
		ID:       "test-id",
		Slug:     "test-slug",
		Path:     "test.md",
		Title:    "Test Title",
		Status:   "todo",
		Type:     "task",
		Priority: "high",
		Tags:     []string{"tag1", "tag2"},
		Body:     "Test body content",
	}

	node := &TreeNode{
		Bean:    b,
		Matched: true,
		Children: []*TreeNode{
			{
				Bean:    &bean.Bean{ID: "child-id", Title: "Child"},
				Matched: false,
			},
		},
	}

	t.Run("without full body", func(t *testing.T) {
		json := node.ToJSON(false)
		if json.ID != "test-id" {
			t.Errorf("expected id 'test-id', got %s", json.ID)
		}
		if json.Body != "" {
			t.Error("body should be empty when includeFull is false")
		}
		if !json.Matched {
			t.Error("matched should be true")
		}
		if len(json.Children) != 1 {
			t.Errorf("expected 1 child, got %d", len(json.Children))
		}
	})

	t.Run("with full body", func(t *testing.T) {
		json := node.ToJSON(true)
		if json.Body != "Test body content" {
			t.Errorf("expected body content, got %s", json.Body)
		}
	})
}
