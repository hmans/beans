package tui

import (
	"testing"

	"github.com/hmans/beans/internal/bean"
)

func TestSortBeans(t *testing.T) {
	// Define the expected order from DefaultStatuses, DefaultPriorities, and DefaultTypes
	statusNames := []string{"draft", "todo", "in-progress", "completed", "scrapped"}
	priorityNames := []string{"critical", "high", "normal", "low", "deferred"}
	typeNames := []string{"milestone", "epic", "bug", "feature", "task"}

	t.Run("sorts by status order first", func(t *testing.T) {
		beans := []*bean.Bean{
			{ID: "1", Status: "completed", Type: "task", Title: "A"},
			{ID: "2", Status: "draft", Type: "task", Title: "B"},
			{ID: "3", Status: "in-progress", Type: "task", Title: "C"},
			{ID: "4", Status: "todo", Type: "task", Title: "D"},
			{ID: "5", Status: "scrapped", Type: "task", Title: "E"},
		}

		bean.SortByStatusPriorityAndType(beans, statusNames, priorityNames, typeNames)

		expected := []string{"draft", "todo", "in-progress", "completed", "scrapped"}
		for i, want := range expected {
			if beans[i].Status != want {
				t.Errorf("index %d: got status %q, want %q", i, beans[i].Status, want)
			}
		}
	})

	t.Run("sorts by priority within same status", func(t *testing.T) {
		beans := []*bean.Bean{
			{ID: "1", Status: "todo", Type: "task", Priority: "low", Title: "A"},
			{ID: "2", Status: "todo", Type: "task", Priority: "critical", Title: "B"},
			{ID: "3", Status: "todo", Type: "task", Priority: "high", Title: "C"},
			{ID: "4", Status: "todo", Type: "task", Priority: "", Title: "D"},       // empty = normal
			{ID: "5", Status: "todo", Type: "task", Priority: "deferred", Title: "E"},
		}

		bean.SortByStatusPriorityAndType(beans, statusNames, priorityNames, typeNames)

		// Order: critical, high, normal (empty), low, deferred
		expectedPriorities := []string{"critical", "high", "", "low", "deferred"}
		for i, want := range expectedPriorities {
			if beans[i].Priority != want {
				t.Errorf("index %d: got priority %q, want %q", i, beans[i].Priority, want)
			}
		}
	})

	t.Run("sorts by type order within same status and priority", func(t *testing.T) {
		beans := []*bean.Bean{
			{ID: "1", Status: "todo", Type: "task", Title: "A"},
			{ID: "2", Status: "todo", Type: "milestone", Title: "B"},
			{ID: "3", Status: "todo", Type: "bug", Title: "C"},
			{ID: "4", Status: "todo", Type: "epic", Title: "D"},
			{ID: "5", Status: "todo", Type: "feature", Title: "E"},
		}

		bean.SortByStatusPriorityAndType(beans, statusNames, priorityNames, typeNames)

		expected := []string{"milestone", "epic", "bug", "feature", "task"}
		for i, want := range expected {
			if beans[i].Type != want {
				t.Errorf("index %d: got type %q, want %q", i, beans[i].Type, want)
			}
		}
	})

	t.Run("sorts by title within same status, priority, and type", func(t *testing.T) {
		beans := []*bean.Bean{
			{ID: "1", Status: "todo", Type: "task", Title: "Zebra"},
			{ID: "2", Status: "todo", Type: "task", Title: "Apple"},
			{ID: "3", Status: "todo", Type: "task", Title: "Mango"},
		}

		bean.SortByStatusPriorityAndType(beans, statusNames, priorityNames, typeNames)

		expected := []string{"Apple", "Mango", "Zebra"}
		for i, want := range expected {
			if beans[i].Title != want {
				t.Errorf("index %d: got title %q, want %q", i, beans[i].Title, want)
			}
		}
	})

	t.Run("title sort is case-insensitive", func(t *testing.T) {
		beans := []*bean.Bean{
			{ID: "1", Status: "todo", Type: "task", Title: "zebra"},
			{ID: "2", Status: "todo", Type: "task", Title: "Apple"},
			{ID: "3", Status: "todo", Type: "task", Title: "MANGO"},
		}

		bean.SortByStatusPriorityAndType(beans, statusNames, priorityNames, typeNames)

		expected := []string{"Apple", "MANGO", "zebra"}
		for i, want := range expected {
			if beans[i].Title != want {
				t.Errorf("index %d: got title %q, want %q", i, beans[i].Title, want)
			}
		}
	})

	t.Run("combined sort order: status > priority > type > title", func(t *testing.T) {
		beans := []*bean.Bean{
			{ID: "1", Status: "completed", Type: "bug", Title: "Z"},
			{ID: "2", Status: "todo", Type: "task", Priority: "low", Title: "A"},
			{ID: "3", Status: "todo", Type: "bug", Priority: "high", Title: "B"},
			{ID: "4", Status: "todo", Type: "bug", Priority: "high", Title: "A"},
			{ID: "5", Status: "draft", Type: "epic", Title: "X"},
		}

		bean.SortByStatusPriorityAndType(beans, statusNames, priorityNames, typeNames)

		// Expected order:
		// 1. draft/epic/X (ID: 5)
		// 2. todo/high/bug/A (ID: 4)
		// 3. todo/high/bug/B (ID: 3)
		// 4. todo/low/task/A (ID: 2)
		// 5. completed/bug/Z (ID: 1)
		expectedIDs := []string{"5", "4", "3", "2", "1"}
		for i, want := range expectedIDs {
			if beans[i].ID != want {
				t.Errorf("index %d: got ID %q, want %q (status=%s, priority=%s, type=%s, title=%s)",
					i, beans[i].ID, want, beans[i].Status, beans[i].Priority, beans[i].Type, beans[i].Title)
			}
		}
	})

	t.Run("unrecognized status sorts last", func(t *testing.T) {
		beans := []*bean.Bean{
			{ID: "1", Status: "unknown", Type: "task", Title: "A"},
			{ID: "2", Status: "todo", Type: "task", Title: "B"},
			{ID: "3", Status: "draft", Type: "task", Title: "C"},
		}

		bean.SortByStatusPriorityAndType(beans, statusNames, priorityNames, typeNames)

		// unknown status should be last
		if beans[2].Status != "unknown" {
			t.Errorf("unrecognized status should be last, got %q at position 2", beans[2].Status)
		}
	})

	t.Run("unrecognized type sorts last within status", func(t *testing.T) {
		beans := []*bean.Bean{
			{ID: "1", Status: "todo", Type: "unknown", Title: "A"},
			{ID: "2", Status: "todo", Type: "task", Title: "B"},
			{ID: "3", Status: "todo", Type: "bug", Title: "C"},
		}

		bean.SortByStatusPriorityAndType(beans, statusNames, priorityNames, typeNames)

		// unknown type should be last within todo status
		if beans[2].Type != "unknown" {
			t.Errorf("unrecognized type should be last, got %q at position 2", beans[2].Type)
		}
	})

	t.Run("empty slice does not panic", func(t *testing.T) {
		beans := []*bean.Bean{}
		bean.SortByStatusPriorityAndType(beans, statusNames, priorityNames, typeNames)
		// No assertion needed, just checking it doesn't panic
	})

	t.Run("single bean does not panic", func(t *testing.T) {
		beans := []*bean.Bean{
			{ID: "1", Status: "todo", Type: "task", Title: "A"},
		}
		bean.SortByStatusPriorityAndType(beans, statusNames, priorityNames, typeNames)
		if beans[0].ID != "1" {
			t.Error("single bean should remain unchanged")
		}
	})
}

func TestCompareBeansByStatusPriorityAndType(t *testing.T) {
	statusNames := []string{"draft", "todo", "in-progress", "completed", "scrapped"}
	priorityNames := []string{"critical", "high", "normal", "low", "deferred"}
	typeNames := []string{"milestone", "epic", "bug", "feature", "task"}

	t.Run("compares by status first", func(t *testing.T) {
		a := &bean.Bean{ID: "1", Status: "todo", Type: "task", Title: "A"}
		b := &bean.Bean{ID: "2", Status: "draft", Type: "task", Title: "B"}

		// draft < todo, so b should come before a
		if compareBeansByStatusPriorityAndType(a, b, statusNames, priorityNames, typeNames) {
			t.Error("draft bean should come before todo bean")
		}
		if !compareBeansByStatusPriorityAndType(b, a, statusNames, priorityNames, typeNames) {
			t.Error("draft bean should come before todo bean")
		}
	})

	t.Run("compares by priority within same status", func(t *testing.T) {
		a := &bean.Bean{ID: "1", Status: "todo", Type: "task", Priority: "low", Title: "A"}
		b := &bean.Bean{ID: "2", Status: "todo", Type: "task", Priority: "high", Title: "B"}

		// high < low, so b should come before a
		if compareBeansByStatusPriorityAndType(a, b, statusNames, priorityNames, typeNames) {
			t.Error("high priority bean should come before low priority bean")
		}
		if !compareBeansByStatusPriorityAndType(b, a, statusNames, priorityNames, typeNames) {
			t.Error("high priority bean should come before low priority bean")
		}
	})

	t.Run("compares by type within same status and priority", func(t *testing.T) {
		a := &bean.Bean{ID: "1", Status: "todo", Type: "task", Title: "A"}
		b := &bean.Bean{ID: "2", Status: "todo", Type: "bug", Title: "B"}

		// bug < task, so b should come before a
		if compareBeansByStatusPriorityAndType(a, b, statusNames, priorityNames, typeNames) {
			t.Error("bug bean should come before task bean")
		}
		if !compareBeansByStatusPriorityAndType(b, a, statusNames, priorityNames, typeNames) {
			t.Error("bug bean should come before task bean")
		}
	})

	t.Run("compares by title within same status, priority, and type", func(t *testing.T) {
		a := &bean.Bean{ID: "1", Status: "todo", Type: "task", Title: "Zebra"}
		b := &bean.Bean{ID: "2", Status: "todo", Type: "task", Title: "Apple"}

		// Apple < Zebra, so b should come before a
		if compareBeansByStatusPriorityAndType(a, b, statusNames, priorityNames, typeNames) {
			t.Error("Apple bean should come before Zebra bean")
		}
		if !compareBeansByStatusPriorityAndType(b, a, statusNames, priorityNames, typeNames) {
			t.Error("Apple bean should come before Zebra bean")
		}
	})

	t.Run("title comparison is case-insensitive", func(t *testing.T) {
		a := &bean.Bean{ID: "1", Status: "todo", Type: "task", Title: "zebra"}
		b := &bean.Bean{ID: "2", Status: "todo", Type: "task", Title: "APPLE"}

		// apple < zebra (case-insensitive), so b should come before a
		if compareBeansByStatusPriorityAndType(a, b, statusNames, priorityNames, typeNames) {
			t.Error("APPLE bean should come before zebra bean (case-insensitive)")
		}
	})

	t.Run("empty priority treated as normal", func(t *testing.T) {
		a := &bean.Bean{ID: "1", Status: "todo", Type: "task", Priority: "", Title: "A"}
		b := &bean.Bean{ID: "2", Status: "todo", Type: "task", Priority: "normal", Title: "B"}

		// Both should be equivalent in priority ordering
		// Since titles differ, A < B, so a should come before b
		if !compareBeansByStatusPriorityAndType(a, b, statusNames, priorityNames, typeNames) {
			t.Error("empty priority should be treated as normal")
		}
	})

	t.Run("unrecognized status sorts last", func(t *testing.T) {
		a := &bean.Bean{ID: "1", Status: "unknown", Type: "task", Title: "A"}
		b := &bean.Bean{ID: "2", Status: "scrapped", Type: "task", Title: "B"}

		// scrapped is last known status, unknown should be after it
		if compareBeansByStatusPriorityAndType(a, b, statusNames, priorityNames, typeNames) {
			t.Error("unknown status should sort after scrapped")
		}
		if !compareBeansByStatusPriorityAndType(b, a, statusNames, priorityNames, typeNames) {
			t.Error("scrapped should sort before unknown")
		}
	})

	t.Run("unrecognized type sorts last within status", func(t *testing.T) {
		a := &bean.Bean{ID: "1", Status: "todo", Type: "unknown", Title: "A"}
		b := &bean.Bean{ID: "2", Status: "todo", Type: "task", Title: "B"}

		// task is last known type, unknown should be after it
		if compareBeansByStatusPriorityAndType(a, b, statusNames, priorityNames, typeNames) {
			t.Error("unknown type should sort after task")
		}
		if !compareBeansByStatusPriorityAndType(b, a, statusNames, priorityNames, typeNames) {
			t.Error("task should sort before unknown")
		}
	})
}
