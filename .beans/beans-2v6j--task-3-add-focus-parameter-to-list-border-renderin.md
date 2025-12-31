---
# beans-2v6j
title: 'Task 3: Add focus parameter to list border rendering'
status: completed
type: task
priority: normal
created_at: 2025-12-30T16:36:00Z
updated_at: 2025-12-30T17:18:32Z
parent: beans-pn6z
---

## Overview

Update list.go to accept a focus parameter that controls border color.

## Files

- Modify: `internal/tui/list.go:504-514` (viewContent method)
- Modify: `internal/tui/list.go:582-603` (ViewConstrained method)

## Steps

### Step 1: Add focused parameter to viewContent

Update the `viewContent` method signature and implementation:

```go
// viewContent renders just the bordered list without footer.
// innerHeight is the content height inside the border (not including border lines).
// focused determines the border color (primary when focused, muted when not).
func (m listModel) viewContent(innerHeight int, focused bool) string {
	borderColor := ui.ColorMuted
	if focused {
		borderColor = ui.ColorPrimary
	}
	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(m.width - 2).
		Height(innerHeight)

	return border.Render(m.list.View())
}
```

### Step 2: Update View() to pass focused=true

In `View()`, update the call:
```go
return m.viewContent(m.height-4, true) + "\n" + m.Footer()
```

### Step 3: Add focused parameter to ViewConstrained

Update signature:
```go
func (m listModel) ViewConstrained(width, height int, focused bool) string {
```

Update the return:
```go
return m.viewContent(innerHeight, focused)
```

### Step 4: Update call site in tui.go

In `renderTwoColumnView()`, update the call (will be refactored more later):
```go
leftPane := a.list.ViewConstrained(leftWidth, contentHeight, true) // TODO: pass actual focus state
```

### Step 5: Build and verify

Run: `mise build`
Expected: Build succeeds

### Step 6: Commit

```bash
git add internal/tui/list.go internal/tui/tui.go
git commit -m "feat(tui): add focus parameter to list border rendering

Border color changes based on focus state:
- Primary (cyan) when focused
- Muted (gray) when not focused

Refs: beans-pn6z"
```