package bean

import (
	"testing"
)

func TestSortByStatusPriorityAndType(t *testing.T) {
	statusNames := []string{"draft", "todo", "in-progress", "completed"}
	priorityNames := []string{"critical", "high", "normal", "low", "deferred"}
	typeNames := []string{"bug", "feature", "task"}

	t.Run("sorts by status first", func(t *testing.T) {
		beans := []*Bean{
			{ID: "1", Title: "A", Status: "completed", Priority: "critical"},
			{ID: "2", Title: "B", Status: "todo", Priority: "low"},
			{ID: "3", Title: "C", Status: "draft", Priority: "high"},
		}

		SortByStatusPriorityAndType(beans, statusNames, priorityNames, typeNames)

		if beans[0].Status != "draft" {
			t.Errorf("First bean status = %q, want \"draft\"", beans[0].Status)
		}
		if beans[1].Status != "todo" {
			t.Errorf("Second bean status = %q, want \"todo\"", beans[1].Status)
		}
		if beans[2].Status != "completed" {
			t.Errorf("Third bean status = %q, want \"completed\"", beans[2].Status)
		}
	})

	t.Run("sorts by priority within same status", func(t *testing.T) {
		beans := []*Bean{
			{ID: "1", Title: "E Low", Status: "todo", Priority: "low"},
			{ID: "2", Title: "A Critical", Status: "todo", Priority: "critical"},
			{ID: "3", Title: "B High", Status: "todo", Priority: "high"},
			{ID: "4", Title: "C Normal", Status: "todo", Priority: "normal"},
			{ID: "5", Title: "D No Priority", Status: "todo", Priority: ""},
		}

		SortByStatusPriorityAndType(beans, statusNames, priorityNames, typeNames)

		// Order by priority: critical, high, normal (and empty), low, deferred
		// Within same priority, order by title alphabetically
		expectedOrder := []string{"A Critical", "B High", "C Normal", "D No Priority", "E Low"}
		for i, expected := range expectedOrder {
			if beans[i].Title != expected {
				t.Errorf("beans[%d].Title = %q, want %q", i, beans[i].Title, expected)
			}
		}
	})

	t.Run("empty priority treated as normal", func(t *testing.T) {
		beans := []*Bean{
			{ID: "1", Title: "Low", Status: "todo", Priority: "low"},
			{ID: "2", Title: "Empty", Status: "todo", Priority: ""},
			{ID: "3", Title: "Normal", Status: "todo", Priority: "normal"},
			{ID: "4", Title: "High", Status: "todo", Priority: "high"},
		}

		SortByStatusPriorityAndType(beans, statusNames, priorityNames, typeNames)

		// High should come first, then Normal and Empty (same priority level), then Low
		if beans[0].Title != "High" {
			t.Errorf("First bean = %q, want \"High\"", beans[0].Title)
		}
		if beans[3].Title != "Low" {
			t.Errorf("Last bean = %q, want \"Low\"", beans[3].Title)
		}
		// Empty and Normal should be adjacent (both at normal priority level)
		normalIdx, emptyIdx := -1, -1
		for i, b := range beans {
			if b.Title == "Normal" {
				normalIdx = i
			}
			if b.Title == "Empty" {
				emptyIdx = i
			}
		}
		if normalIdx != 1 && normalIdx != 2 {
			t.Errorf("Normal should be at index 1 or 2, got %d", normalIdx)
		}
		if emptyIdx != 1 && emptyIdx != 2 {
			t.Errorf("Empty should be at index 1 or 2, got %d", emptyIdx)
		}
	})

	t.Run("sorts by type after priority", func(t *testing.T) {
		beans := []*Bean{
			{ID: "1", Title: "Task", Status: "todo", Priority: "high", Type: "task"},
			{ID: "2", Title: "Bug", Status: "todo", Priority: "high", Type: "bug"},
			{ID: "3", Title: "Feature", Status: "todo", Priority: "high", Type: "feature"},
		}

		SortByStatusPriorityAndType(beans, statusNames, priorityNames, typeNames)

		if beans[0].Type != "bug" {
			t.Errorf("First bean type = %q, want \"bug\"", beans[0].Type)
		}
		if beans[1].Type != "feature" {
			t.Errorf("Second bean type = %q, want \"feature\"", beans[1].Type)
		}
		if beans[2].Type != "task" {
			t.Errorf("Third bean type = %q, want \"task\"", beans[2].Type)
		}
	})

	t.Run("sorts by title after type", func(t *testing.T) {
		beans := []*Bean{
			{ID: "1", Title: "Zebra", Status: "todo", Priority: "high", Type: "bug"},
			{ID: "2", Title: "Apple", Status: "todo", Priority: "high", Type: "bug"},
			{ID: "3", Title: "Mango", Status: "todo", Priority: "high", Type: "bug"},
		}

		SortByStatusPriorityAndType(beans, statusNames, priorityNames, typeNames)

		if beans[0].Title != "Apple" {
			t.Errorf("First bean title = %q, want \"Apple\"", beans[0].Title)
		}
		if beans[1].Title != "Mango" {
			t.Errorf("Second bean title = %q, want \"Mango\"", beans[1].Title)
		}
		if beans[2].Title != "Zebra" {
			t.Errorf("Third bean title = %q, want \"Zebra\"", beans[2].Title)
		}
	})

	t.Run("handles nil priority names gracefully", func(t *testing.T) {
		beans := []*Bean{
			{ID: "1", Title: "A", Status: "todo", Priority: "high"},
			{ID: "2", Title: "B", Status: "todo", Priority: ""},
		}

		// Should not panic with nil priorityNames
		SortByStatusPriorityAndType(beans, statusNames, nil, typeNames)

		// Both should be sorted by status, type, then title
		if beans[0].Title != "A" {
			t.Errorf("First bean title = %q, want \"A\"", beans[0].Title)
		}
	})
}

