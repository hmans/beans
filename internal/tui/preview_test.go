package tui

import (
	"strings"
	"testing"

	"github.com/hmans/beans/internal/bean"
)

func TestPreviewView(t *testing.T) {
	b := &bean.Bean{
		ID:       "beans-test",
		Title:    "Test Bean",
		Status:   "todo",
		Type:     "feature",
		Priority: "high",
		Tags:     []string{"frontend", "design"},
		Body:     "## Summary\n\nThis is the body.",
	}

	preview := newPreviewModel(b, 60, 20)
	view := preview.View()

	// Should contain the title
	if !strings.Contains(view, "Test Bean") {
		t.Error("preview should contain bean title")
	}

	// Should contain the ID
	if !strings.Contains(view, "beans-test") {
		t.Error("preview should contain bean ID")
	}

	// Should contain status
	if !strings.Contains(view, "todo") {
		t.Error("preview should contain status")
	}

	// Should contain type
	if !strings.Contains(view, "feature") {
		t.Error("preview should contain type")
	}

	// Should contain body content
	if !strings.Contains(view, "Summary") {
		t.Error("preview should contain body")
	}
}

func TestPreviewViewEmpty(t *testing.T) {
	preview := newPreviewModel(nil, 60, 20)
	view := preview.View()

	if !strings.Contains(view, "No bean selected") {
		t.Error("empty preview should show 'No bean selected'")
	}
}

func TestPreviewViewWithTags(t *testing.T) {
	b := &bean.Bean{
		ID:     "beans-test",
		Title:  "Bean with Tags",
		Status: "in-progress",
		Type:   "bug",
		Tags:   []string{"urgent", "backend"},
		Body:   "Test body",
	}

	preview := newPreviewModel(b, 60, 20)
	view := preview.View()

	// Should show tags
	if !strings.Contains(view, "urgent") || !strings.Contains(view, "backend") {
		t.Error("preview should display tags")
	}
}

func TestPreviewViewWithPriority(t *testing.T) {
	b := &bean.Bean{
		ID:       "beans-test",
		Title:    "High Priority Bean",
		Status:   "todo",
		Type:     "task",
		Priority: "critical",
		Body:     "Important work",
	}

	preview := newPreviewModel(b, 60, 20)
	view := preview.View()

	// Should show priority
	if !strings.Contains(view, "critical") {
		t.Error("preview should display priority when not normal")
	}
}

func TestPreviewViewEmptyBody(t *testing.T) {
	b := &bean.Bean{
		ID:     "beans-test",
		Title:  "Bean without body",
		Status: "todo",
		Type:   "task",
		Body:   "",
	}

	preview := newPreviewModel(b, 60, 20)
	view := preview.View()

	// Should show placeholder for empty body
	if !strings.Contains(view, "No description") {
		t.Error("preview should show 'No description' for empty body")
	}
}
