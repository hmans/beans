package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/beancore"
	"github.com/hmans/beans/internal/config"
)

// =============================================================================
// Test Helpers
// =============================================================================

// setupTestApp creates an App with test beans in a temp directory
func setupTestApp(t *testing.T, beans []*bean.Bean) *App {
	t.Helper()

	tmpDir := t.TempDir()
	beansDir := filepath.Join(tmpDir, beancore.BeansDir)
	if err := os.MkdirAll(beansDir, 0755); err != nil {
		t.Fatalf("failed to create test .beans dir: %v", err)
	}

	cfg := config.Default()
	core := beancore.New(beansDir, cfg)
	core.SetWarnWriter(nil) // suppress warnings in tests
	if err := core.Load(); err != nil {
		t.Fatalf("failed to load core: %v", err)
	}

	// Create test beans
	for _, b := range beans {
		if err := core.Create(b); err != nil {
			t.Fatalf("failed to create test bean %s: %v", b.ID, err)
		}
	}

	return New(core, cfg)
}

// captureAndQuit waits for condition, captures output, then quits the program
func captureAndQuit(t *testing.T, tm *teatest.TestModel, condition func(string) bool) []byte {
	t.Helper()

	var out []byte
	teatest.WaitFor(t, tm.Output(),
		func(bts []byte) bool {
			s := string(bts)
			if condition(s) {
				out = bts
				return true
			}
			return false
		},
		teatest.WithDuration(3*time.Second),
	)

	// Quit cleanly
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))

	return out
}

// listReadyCondition returns true when the list footer is visible
func listReadyCondition(s string) bool {
	return strings.Contains(s, "space") && strings.Contains(s, "select")
}

// detailReadyCondition returns true when the detail footer is visible
func detailReadyCondition(s string) bool {
	return strings.Contains(s, "esc back")
}

// bodyFocusCondition returns true when body is focused (scroll in footer)
func bodyFocusCondition(s string) bool {
	return strings.Contains(s, "scroll")
}

// =============================================================================
// Test Fixtures
// =============================================================================

// testBeanSimple returns a simple bean with no links
func testBeanSimple() *bean.Bean {
	return &bean.Bean{
		ID:     "test-1234",
		Slug:   "simple-task",
		Title:  "Simple Task",
		Status: "todo",
		Type:   "task",
		Body:   "This is a simple task with no links.",
	}
}

// testBeanWithFewLinks returns beans with parent/child and blocking relationships
func testBeanWithFewLinks() []*bean.Bean {
	return []*bean.Bean{
		{
			ID:       "parent-01",
			Slug:     "parent-feature",
			Title:    "Parent Feature",
			Status:   "in-progress",
			Type:     "feature",
			Body:     "This is the parent feature.",
			Blocking: []string{"child-01"},
		},
		{
			ID:     "child-01",
			Slug:   "child-task",
			Title:  "Child Task",
			Status: "todo",
			Type:   "task",
			Parent: "parent-01",
			Body:   "This task is a child of the parent feature.",
		},
		{
			ID:       "blocker-01",
			Slug:     "blocking-bug",
			Title:    "Blocking Bug",
			Status:   "in-progress",
			Type:     "bug",
			Blocking: []string{"child-01"},
			Body:     "This bug blocks the child task.",
		},
	}
}

// testBeanWithManyLinks returns an epic with 8 children to trigger pagination
func testBeanWithManyLinks() []*bean.Bean {
	beans := []*bean.Bean{
		{
			ID:     "main-bean",
			Slug:   "main-epic",
			Title:  "Main Epic with Many Children",
			Status: "in-progress",
			Type:   "epic",
			Body:   "This epic has many child tasks to test pagination in the links section.",
		},
	}

	// Add 8 child beans to trigger pagination (max visible is 5)
	for i := 1; i <= 8; i++ {
		beans = append(beans, &bean.Bean{
			ID:     fmt.Sprintf("child-%02d", i),
			Slug:   "child-task",
			Title:  "Child Task " + string(rune('A'+i-1)),
			Status: "todo",
			Type:   "task",
			Parent: "main-bean",
			Body:   "Child task for testing.",
		})
	}

	return beans
}

// testBeanLongTitle returns a bean with a very long title
func testBeanLongTitle() *bean.Bean {
	return &bean.Bean{
		ID:     "long-1234",
		Slug:   "very-long-title",
		Title:  "This is a very long title that should be truncated when displayed in narrow terminals to ensure proper layout",
		Status: "todo",
		Type:   "feature",
		Body:   "Body content for the long title bean.",
	}
}

// =============================================================================
// Layout Tests
// =============================================================================

// TestLayoutWideTerminal tests the two-column layout at wide terminal width
func TestLayoutWideTerminal(t *testing.T) {
	beans := testBeanWithFewLinks()
	app := setupTestApp(t, beans)

	tm := teatest.NewTestModel(t, app,
		teatest.WithInitialTermSize(120, 40))

	out := captureAndQuit(t, tm, listReadyCondition)
	teatest.RequireEqualOutput(t, out)
}

// TestLayoutNarrowTerminal tests the single-column layout at narrow terminal width
func TestLayoutNarrowTerminal(t *testing.T) {
	beans := []*bean.Bean{testBeanSimple()}
	app := setupTestApp(t, beans)

	tm := teatest.NewTestModel(t, app,
		teatest.WithInitialTermSize(80, 24))

	out := captureAndQuit(t, tm, listReadyCondition)
	teatest.RequireEqualOutput(t, out)
}

// TestLayoutDetailViewWithLinks tests the detail view when links section is present
func TestLayoutDetailViewWithLinks(t *testing.T) {
	beans := testBeanWithFewLinks()
	app := setupTestApp(t, beans)

	tm := teatest.NewTestModel(t, app,
		teatest.WithInitialTermSize(120, 40))

	// Wait for list to load
	teatest.WaitFor(t, tm.Output(),
		func(bts []byte) bool { return listReadyCondition(string(bts)) },
		teatest.WithDuration(3*time.Second),
	)

	// Navigate to child-01 which has parent and blocked-by links
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	time.Sleep(50 * time.Millisecond)

	// Press Enter to focus detail
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	out := captureAndQuit(t, tm, detailReadyCondition)
	teatest.RequireEqualOutput(t, out)
}

// TestLayoutDetailViewManyLinks tests layout with many links (pagination)
func TestLayoutDetailViewManyLinks(t *testing.T) {
	beans := testBeanWithManyLinks()
	app := setupTestApp(t, beans)

	tm := teatest.NewTestModel(t, app,
		teatest.WithInitialTermSize(120, 40))

	// Wait for list to load
	teatest.WaitFor(t, tm.Output(),
		func(bts []byte) bool { return listReadyCondition(string(bts)) },
		teatest.WithDuration(3*time.Second),
	)

	// Press Enter to focus detail (the main-bean has 8 children)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	out := captureAndQuit(t, tm, detailReadyCondition)
	teatest.RequireEqualOutput(t, out)
}

// TestLayoutResizeFromWideToNarrow tests layout adjustment when resizing
func TestLayoutResizeFromWideToNarrow(t *testing.T) {
	beans := []*bean.Bean{testBeanSimple()}
	app := setupTestApp(t, beans)

	// Start wide
	tm := teatest.NewTestModel(t, app,
		teatest.WithInitialTermSize(120, 40))

	// Wait for list to load
	teatest.WaitFor(t, tm.Output(),
		func(bts []byte) bool { return listReadyCondition(string(bts)) },
		teatest.WithDuration(3*time.Second),
	)

	// Resize to narrow
	tm.Send(tea.WindowSizeMsg{Width: 80, Height: 24})
	time.Sleep(100 * time.Millisecond)

	out := captureAndQuit(t, tm, listReadyCondition)
	teatest.RequireEqualOutput(t, out)
}

// TestLayoutLongTitle tests that long titles are truncated properly
func TestLayoutLongTitle(t *testing.T) {
	beans := []*bean.Bean{testBeanLongTitle()}
	app := setupTestApp(t, beans)

	tm := teatest.NewTestModel(t, app,
		teatest.WithInitialTermSize(100, 30))

	out := captureAndQuit(t, tm, listReadyCondition)
	teatest.RequireEqualOutput(t, out)
}

// =============================================================================
// Focus Transition Tests
// =============================================================================

// TestFocusTransitionListToDetail tests entering detail view from list
func TestFocusTransitionListToDetail(t *testing.T) {
	beans := testBeanWithFewLinks()
	app := setupTestApp(t, beans)

	tm := teatest.NewTestModel(t, app,
		teatest.WithInitialTermSize(120, 40))

	// Wait for list to load
	teatest.WaitFor(t, tm.Output(),
		func(bts []byte) bool { return listReadyCondition(string(bts)) },
		teatest.WithDuration(3*time.Second),
	)

	// Press Enter to focus detail
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	out := captureAndQuit(t, tm, detailReadyCondition)
	teatest.RequireEqualOutput(t, out)
}

// TestFocusTransitionBackToList tests pressing backspace to return to list
func TestFocusTransitionBackToList(t *testing.T) {
	beans := []*bean.Bean{testBeanSimple()}
	app := setupTestApp(t, beans)

	tm := teatest.NewTestModel(t, app,
		teatest.WithInitialTermSize(120, 40))

	// Wait for list to load
	teatest.WaitFor(t, tm.Output(),
		func(bts []byte) bool { return listReadyCondition(string(bts)) },
		teatest.WithDuration(3*time.Second),
	)

	// Press Enter to focus detail (body since no links)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Wait for detail
	teatest.WaitFor(t, tm.Output(),
		func(bts []byte) bool { return detailReadyCondition(string(bts)) },
		teatest.WithDuration(3*time.Second),
	)

	// Press Backspace to return to list
	tm.Send(tea.KeyMsg{Type: tea.KeyBackspace})

	// Capture when back in list (no backspace in footer)
	out := captureAndQuit(t, tm, func(s string) bool {
		return listReadyCondition(s) && !strings.Contains(s, "backspace")
	})
	teatest.RequireEqualOutput(t, out)
}

// TestDetailTabSwitchBetweenLinksAndBody tests Tab key cycling between links and body
func TestDetailTabSwitchBetweenLinksAndBody(t *testing.T) {
	beans := testBeanWithFewLinks()
	app := setupTestApp(t, beans)

	tm := teatest.NewTestModel(t, app,
		teatest.WithInitialTermSize(120, 40))

	// Wait for list to load
	teatest.WaitFor(t, tm.Output(),
		func(bts []byte) bool { return listReadyCondition(string(bts)) },
		teatest.WithDuration(3*time.Second),
	)

	// Navigate to child-01 (has links)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	time.Sleep(50 * time.Millisecond)

	// Press Enter to focus detail (starts in links since bean has links)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Wait for detail
	teatest.WaitFor(t, tm.Output(),
		func(bts []byte) bool { return detailReadyCondition(string(bts)) },
		teatest.WithDuration(3*time.Second),
	)

	// Press Tab to switch to body
	tm.Send(tea.KeyMsg{Type: tea.KeyTab})

	out := captureAndQuit(t, tm, bodyFocusCondition)
	teatest.RequireEqualOutput(t, out)
}
