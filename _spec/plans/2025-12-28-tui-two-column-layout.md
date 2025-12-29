# TUI Two-Column Layout Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a two-column TUI layout with bean list on the left and read-only detail preview on the right.

**Architecture:** Extend the existing Bubbletea TUI with responsive layout detection, compact list rendering, and a lightweight detail preview component. Cursor movement in the list automatically updates the preview. Enter opens the existing full-screen detail view.

**Tech Stack:** Go, Bubbletea, Lipgloss, existing internal/tui and internal/ui packages.

**Parent Bean:** beans-t0tv

---

## Phase 1: Compact List Format

**Bean:** beans-t0tv-p1

Add single-character type and status codes to make the list more compact. This is a prerequisite for the two-column layout where horizontal space is limited.

### Task 1.1: Add Single-Char Type/Status Helpers

**Files:**
- Modify: `internal/ui/styles.go`
- Test: `internal/ui/styles_test.go`

**Step 1: Write the failing tests**

Add to `internal/ui/styles_test.go`:

```go
func TestShortType(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"milestone", "M"},
		{"epic", "E"},
		{"bug", "B"},
		{"feature", "F"},
		{"task", "T"},
		{"unknown", "?"},
		{"", "?"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ShortType(tt.input)
			if result != tt.expected {
				t.Errorf("ShortType(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestShortStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"draft", "D"},
		{"todo", "T"},
		{"in-progress", "I"},
		{"completed", "C"},
		{"scrapped", "S"},
		{"unknown", "?"},
		{"", "?"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ShortStatus(tt.input)
			if result != tt.expected {
				t.Errorf("ShortStatus(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/ui/ -run "TestShort" -v`
Expected: FAIL with "undefined: ShortType" and "undefined: ShortStatus"

**Step 3: Implement the helpers**

Add to `internal/ui/styles.go`:

```go
// ShortType returns a single-character code for the bean type.
func ShortType(t string) string {
	switch t {
	case "milestone":
		return "M"
	case "epic":
		return "E"
	case "bug":
		return "B"
	case "feature":
		return "F"
	case "task":
		return "T"
	default:
		return "?"
	}
}

// ShortStatus returns a single-character code for the bean status.
func ShortStatus(s string) string {
	switch s {
	case "draft":
		return "D"
	case "todo":
		return "T"
	case "in-progress":
		return "I"
	case "completed":
		return "C"
	case "scrapped":
		return "S"
	default:
		return "?"
	}
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/ui/ -run "TestShort" -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/ui/styles.go internal/ui/styles_test.go
git commit -m "feat(ui): add ShortType and ShortStatus helpers

Single-character codes for compact list display:
- Types: M(ilestone), E(pic), B(ug), F(eature), T(ask)
- Statuses: D(raft), T(odo), I(n-progress), C(ompleted), S(crapped)

Refs: beans-t0tv"
```

### Task 1.2: Update List Rendering to Use Compact Format

**Files:**
- Modify: `internal/ui/styles.go` (RenderBeanRow function)
- Modify: `internal/tui/list.go` (column width calculations)

**Step 1: Understand current rendering**

Read `internal/ui/styles.go` to find `RenderBeanRow()` function. Note how type and status columns are rendered (currently full names like "feature", "in-progress").

**Step 2: Modify RenderBeanRow to use compact format**

In `internal/ui/styles.go`, find the type and status column rendering in `RenderBeanRow()` and replace with:

```go
// Type column - single character
typeStr := ShortType(opts.Type)
typeCol := typeStyle.Width(3).Render(typeStr)

// Status column - single character
statusStr := ShortStatus(opts.Status)
statusCol := statusStyle.Width(3).Render(statusStr)
```

**Step 3: Update column width constants**

In `internal/ui/styles.go`, find the column width constants and update:

```go
// Old values (approximately):
// StatusColWidth = 14
// TypeColWidth = 12

// New values:
StatusColWidth = 3
TypeColWidth = 3
```

**Step 4: Update responsive column calculation**

In `internal/ui/styles.go`, find `CalculateResponsiveColumns()` and update the base widths calculation to account for the smaller type/status columns.

**Step 5: Run existing tests**

Run: `go test ./internal/ui/ -v`
Run: `go test ./internal/tui/ -v`
Expected: PASS (or identify any tests that need updating)

**Step 6: Manual test**

Run: `mise beans` then `beans tui`
Verify the list shows single-character type and status codes.

**Step 7: Commit**

```bash
git add internal/ui/styles.go internal/tui/list.go
git commit -m "feat(ui): use compact single-char type/status in list

- Type column: 3 chars (M/E/B/F/T)
- Status column: 3 chars (D/T/I/C/S)
- Frees up ~20 chars per row for title

Refs: beans-t0tv"
```

---

## Phase 2: Detail Preview Component

**Bean:** beans-t0tv-p2

Create a lightweight, read-only detail preview that can be rendered in the right pane.

### Task 2.1: Create Detail Preview Model

**Files:**
- Create: `internal/tui/preview.go`
- Test: `internal/tui/preview_test.go`

**Step 1: Write the test for preview rendering**

Create `internal/tui/preview_test.go`:

```go
package tui

import (
	"strings"
	"testing"

	"github.com/your-org/beans/internal/bean"
)

func TestPreviewView(t *testing.T) {
	b := &bean.Bean{
		ID:     "beans-test",
		Title:  "Test Bean",
		Status: "todo",
		Type:   "feature",
		Body:   "## Summary\n\nThis is the body.",
	}

	preview := newPreviewModel(b, 60, 20)
	view := preview.View()

	// Should contain the title
	if !strings.Contains(view, "Test Bean") {
		t.Error("preview should contain bean title")
	}

	// Should contain status
	if !strings.Contains(view, "todo") {
		t.Error("preview should contain status")
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
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run "TestPreview" -v`
Expected: FAIL with "undefined: newPreviewModel"

**Step 3: Implement the preview model**

Create `internal/tui/preview.go`:

```go
package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/your-org/beans/internal/bean"
	"github.com/your-org/beans/internal/ui"
)

// previewModel is a read-only detail preview for the two-column layout.
// It has no focus, no interaction - just renders bean details.
type previewModel struct {
	bean   *bean.Bean
	width  int
	height int
}

func newPreviewModel(b *bean.Bean, width, height int) previewModel {
	return previewModel{
		bean:   b,
		width:  width,
		height: height,
	}
}

func (m previewModel) View() string {
	if m.bean == nil {
		return m.renderEmpty()
	}
	return m.renderBean()
}

func (m previewModel) renderEmpty() string {
	style := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center).
		Foreground(ui.ColorMuted)

	return style.Render("No bean selected")
}

func (m previewModel) renderBean() string {
	// Header: ID and Title
	idStyle := lipgloss.NewStyle().Foreground(ui.ColorPrimary).Bold(true)
	titleStyle := lipgloss.NewStyle().Bold(true)

	header := idStyle.Render(m.bean.ID) + "\n" + titleStyle.Render(m.bean.Title)

	// Metadata: Status, Type, Priority
	metaStyle := lipgloss.NewStyle().Foreground(ui.ColorMuted)
	meta := metaStyle.Render("Status: " + m.bean.Status + "  Type: " + m.bean.Type)
	if m.bean.Priority != "" && m.bean.Priority != "normal" {
		meta += metaStyle.Render("  Priority: " + m.bean.Priority)
	}

	// Body (truncated to fit)
	body := m.renderBody()

	// Compose
	content := lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		meta,
		"",
		body,
	)

	// Border
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorMuted).
		Width(m.width - 2).
		Height(m.height - 2)

	return borderStyle.Render(content)
}

func (m previewModel) renderBody() string {
	if m.bean.Body == "" {
		return lipgloss.NewStyle().Foreground(ui.ColorMuted).Render("No description")
	}

	// Render markdown (reuse existing glamour renderer from detail.go)
	rendered, err := renderMarkdown(m.bean.Body)
	if err != nil {
		return m.bean.Body
	}

	// Truncate to available height
	lines := strings.Split(rendered, "\n")
	availableLines := m.height - 8 // Account for header, meta, borders
	if len(lines) > availableLines {
		lines = lines[:availableLines]
		lines = append(lines, lipgloss.NewStyle().Foreground(ui.ColorMuted).Render("..."))
	}

	return strings.Join(lines, "\n")
}
```

Note: You may need to add `import "strings"` and export or reuse the `renderMarkdown` function from `detail.go`.

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/tui/ -run "TestPreview" -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/tui/preview.go internal/tui/preview_test.go
git commit -m "feat(tui): add previewModel for read-only detail preview

Lightweight component for two-column layout right pane:
- Shows bean ID, title, status, type, priority
- Renders markdown body (truncated to fit)
- Shows 'No bean selected' when empty

Refs: beans-t0tv"
```

---

## Phase 3: Two-Column Layout Composition

**Bean:** beans-t0tv-p3

Wire up the two-column layout in the main App, with responsive width detection.

### Task 3.1: Add Two-Column Width Threshold

**Files:**
- Modify: `internal/tui/tui.go`

**Step 1: Add constants**

Add to `internal/tui/tui.go` near the top:

```go
const (
	// TwoColumnMinWidth is the minimum terminal width for two-column layout
	TwoColumnMinWidth = 120
	// LeftPaneWidth is the fixed width of the list pane in two-column mode
	LeftPaneWidth = 55
)
```

**Step 2: Add helper method**

Add to `internal/tui/tui.go`:

```go
// isTwoColumnMode returns true if the terminal is wide enough for two-column layout
func (a *App) isTwoColumnMode() bool {
	return a.width >= TwoColumnMinWidth
}
```

**Step 3: Commit**

```bash
git add internal/tui/tui.go
git commit -m "feat(tui): add two-column width threshold constants

- TwoColumnMinWidth: 120 columns
- LeftPaneWidth: 55 characters
- isTwoColumnMode() helper method

Refs: beans-t0tv"
```

### Task 3.2: Add Preview State to App

**Files:**
- Modify: `internal/tui/tui.go`

**Step 1: Add preview field to App struct**

In `internal/tui/tui.go`, add to the `App` struct:

```go
type App struct {
	// ... existing fields ...

	// preview is the read-only detail preview for two-column mode
	preview previewModel
}
```

**Step 2: Initialize preview in New()**

In the `New()` function, initialize the preview:

```go
func New(core *beancore.Core, cfg *config.Config) *App {
	// ... existing code ...
	app := &App{
		// ... existing fields ...
		preview: newPreviewModel(nil, 0, 0),
	}
	return app
}
```

**Step 3: Commit**

```bash
git add internal/tui/tui.go
git commit -m "feat(tui): add preview field to App struct

Refs: beans-t0tv"
```

### Task 3.3: Implement Two-Column View Rendering

**Files:**
- Modify: `internal/tui/tui.go`

**Step 1: Modify View() for two-column composition**

In `internal/tui/tui.go`, find the `View()` method and modify the `viewList` case:

```go
func (a *App) View() string {
	switch a.state {
	case viewList:
		if a.isTwoColumnMode() {
			return a.renderTwoColumnView()
		}
		return a.list.View()
	// ... rest of cases unchanged ...
	}
}
```

**Step 2: Add renderTwoColumnView method**

Add to `internal/tui/tui.go`:

```go
func (a *App) renderTwoColumnView() string {
	// Calculate dimensions
	leftWidth := LeftPaneWidth
	rightWidth := a.width - leftWidth - 3 // 3 for border/separator
	height := a.height

	// Render left pane (list)
	// We need to constrain the list to leftWidth
	leftPane := a.list.ViewConstrained(leftWidth, height)

	// Render right pane (preview)
	a.preview.width = rightWidth
	a.preview.height = height - 2 // Account for footer
	rightPane := a.preview.View()

	// Compose horizontally
	return lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
}
```

**Step 3: Add ViewConstrained to listModel**

This will be implemented in Task 3.4.

**Step 4: Commit**

```bash
git add internal/tui/tui.go
git commit -m "feat(tui): implement two-column view rendering

- View() checks isTwoColumnMode() before rendering
- renderTwoColumnView() composes list + preview horizontally
- Falls back to single-column for narrow terminals

Refs: beans-t0tv"
```

### Task 3.4: Add ViewConstrained to List Model

**Files:**
- Modify: `internal/tui/list.go`

**Step 1: Add ViewConstrained method**

Add to `internal/tui/list.go`:

```go
// ViewConstrained renders the list constrained to the given width and height.
// Used for the left pane in two-column mode.
func (m listModel) ViewConstrained(width, height int) string {
	// Store original dimensions
	origWidth := m.width
	origHeight := m.height

	// Temporarily set constrained dimensions
	m.width = width
	m.height = height
	m.list.SetSize(width-2, height-4) // Account for border and footer

	// Recalculate columns for constrained width
	m.cols = ui.CalculateResponsiveColumns(width, m.hasTags)
	m.updateDelegate()

	// Render
	view := m.View()

	// Restore original dimensions (though this model is passed by value)
	m.width = origWidth
	m.height = origHeight

	return view
}
```

**Step 2: Test manually**

Run: `mise beans && beans tui`
In a wide terminal (≥120 cols), verify two-column layout appears.

**Step 3: Commit**

```bash
git add internal/tui/list.go
git commit -m "feat(tui): add ViewConstrained for two-column list pane

Renders list with constrained width for left pane in two-column mode.

Refs: beans-t0tv"
```

---

## Phase 4: Cursor Sync

**Bean:** beans-t0tv-p4

Detect cursor changes in the list and update the preview pane accordingly.

### Task 4.1: Add cursorChangedMsg Type

**Files:**
- Modify: `internal/tui/tui.go`

**Step 1: Add message type**

Add to `internal/tui/tui.go` with other message types:

```go
// cursorChangedMsg is sent when the list cursor moves to a different bean
type cursorChangedMsg struct {
	beanID string
}
```

**Step 2: Commit**

```bash
git add internal/tui/tui.go
git commit -m "feat(tui): add cursorChangedMsg type

Refs: beans-t0tv"
```

### Task 4.2: Emit cursorChangedMsg on Cursor Movement

**Files:**
- Modify: `internal/tui/list.go`

**Step 1: Track previous cursor index**

In `internal/tui/list.go`, find the `Update()` method. Before delegating to `m.list.Update(msg)`, capture the current index:

```go
func (m listModel) Update(msg tea.Msg) (listModel, tea.Cmd) {
	var cmds []tea.Cmd

	// Track cursor position before update
	prevIndex := m.list.Index()

	// ... existing key handling ...

	// Delegate to list component
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	// Check if cursor moved
	if m.list.Index() != prevIndex {
		if item, ok := m.list.SelectedItem().(beanItem); ok {
			cmds = append(cmds, func() tea.Msg {
				return cursorChangedMsg{beanID: item.bean.ID}
			})
		}
	}

	return m, tea.Batch(cmds...)
}
```

**Step 2: Commit**

```bash
git add internal/tui/list.go
git commit -m "feat(tui): emit cursorChangedMsg on cursor movement

Detects when list cursor moves and emits message with new bean ID.

Refs: beans-t0tv"
```

### Task 4.3: Handle cursorChangedMsg in App

**Files:**
- Modify: `internal/tui/tui.go`

**Step 1: Add handler in Update()**

In `internal/tui/tui.go`, find the `Update()` method and add a case for `cursorChangedMsg`:

```go
case cursorChangedMsg:
	// Update preview with the newly highlighted bean
	if msg.beanID != "" {
		bean, err := a.resolver.Query().Bean(context.Background(), msg.beanID)
		if err == nil && bean != nil {
			a.preview = newPreviewModel(bean, a.width-LeftPaneWidth-3, a.height-2)
		}
	} else {
		a.preview = newPreviewModel(nil, a.width-LeftPaneWidth-3, a.height-2)
	}
	return a, nil
```

**Step 2: Also update preview on beansLoadedMsg**

Find the `beansLoadedMsg` handler and add preview update:

```go
case beansLoadedMsg:
	// ... existing handling ...

	// Update preview with current cursor position
	if item, ok := a.list.list.SelectedItem().(beanItem); ok {
		a.preview = newPreviewModel(item.bean, a.width-LeftPaneWidth-3, a.height-2)
	}
```

**Step 3: Test manually**

Run: `mise beans && beans tui`
Move cursor with j/k - right pane should update.

**Step 4: Commit**

```bash
git add internal/tui/tui.go
git commit -m "feat(tui): handle cursorChangedMsg to update preview

- Updates preview when cursor moves in list
- Also updates preview when beans are loaded

Refs: beans-t0tv"
```

---

## Phase 5: Integration & Polish

**Bean:** beans-t0tv-p5

Final integration, help overlay updates, and edge case handling.

### Task 5.1: Update Help Overlay

**Files:**
- Modify: `internal/tui/help.go`

**Step 1: Update help text**

Find the help text in `internal/tui/help.go` and ensure it reflects:
- `enter` - view bean details (opens full-screen detail)
- Remove any hierarchy drilling references
- Keep all existing shortcuts

**Step 2: Commit**

```bash
git add internal/tui/help.go
git commit -m "docs(tui): update help overlay for two-column layout

Refs: beans-t0tv"
```

### Task 5.2: Handle Window Resize in Two-Column Mode

**Files:**
- Modify: `internal/tui/tui.go`

**Step 1: Update preview dimensions on resize**

In the `tea.WindowSizeMsg` handler, add preview dimension update:

```go
case tea.WindowSizeMsg:
	a.width = msg.Width
	a.height = msg.Height

	// Update preview dimensions if in two-column mode
	if a.isTwoColumnMode() {
		a.preview.width = a.width - LeftPaneWidth - 3
		a.preview.height = a.height - 2
	}

	// ... existing list/detail updates ...
```

**Step 2: Commit**

```bash
git add internal/tui/tui.go
git commit -m "fix(tui): update preview dimensions on window resize

Refs: beans-t0tv"
```

### Task 5.3: Handle Empty List State

**Files:**
- Modify: `internal/tui/tui.go`

**Step 1: Clear preview when list is empty**

In the `beansLoadedMsg` handler, check for empty list:

```go
case beansLoadedMsg:
	// ... existing handling ...

	// Update preview
	if len(msg.items) == 0 {
		a.preview = newPreviewModel(nil, a.width-LeftPaneWidth-3, a.height-2)
	} else if item, ok := a.list.list.SelectedItem().(beanItem); ok {
		a.preview = newPreviewModel(item.bean, a.width-LeftPaneWidth-3, a.height-2)
	}
```

**Step 2: Commit**

```bash
git add internal/tui/tui.go
git commit -m "fix(tui): show empty preview when list is empty

Refs: beans-t0tv"
```

### Task 5.4: Final Testing

**Step 1: Run all tests**

```bash
go test ./internal/tui/ -v
go test ./internal/ui/ -v
```

**Step 2: Manual testing checklist**

- [ ] Wide terminal (≥120 cols): two-column layout appears
- [ ] Narrow terminal (<120 cols): single-column layout
- [ ] Cursor movement updates preview
- [ ] Enter opens full-screen detail
- [ ] Escape from detail returns to two-column
- [ ] Tag filter works
- [ ] Text filter works
- [ ] Multi-select works
- [ ] All shortcuts (p/s/t/P/b/e/y/c) work
- [ ] Resize from wide to narrow and back
- [ ] Empty list shows "No bean selected"

**Step 3: Final commit**

```bash
git add .
git commit -m "feat(tui): complete two-column layout implementation

Two-column TUI layout with:
- Left pane: compact bean list (55 chars)
- Right pane: read-only detail preview
- Cursor movement updates preview automatically
- Enter opens full-screen detail view
- Responsive collapse below 120 columns
- Compact single-char type/status codes

Refs: beans-t0tv"
```

---

## Summary

| Phase | Bean | Description |
|-------|------|-------------|
| 1 | beans-t0tv-p1 | Compact list format (single-char type/status) |
| 2 | beans-t0tv-p2 | Detail preview component |
| 3 | beans-t0tv-p3 | Two-column layout composition |
| 4 | beans-t0tv-p4 | Cursor sync |
| 5 | beans-t0tv-p5 | Integration & polish |
