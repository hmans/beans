---
# beans-unjz
title: 'Task 6: Update View() for two-column rendering with focus'
status: todo
type: task
created_at: 2025-12-30T16:37:12Z
updated_at: 2025-12-30T16:37:12Z
parent: beans-pn6z
---

## Overview

Update the View() method to render both panes with correct focus indication, and handle narrow mode (single pane visible).

## Files

- Modify: `internal/tui/tui.go` (View method, renderTwoColumnView)

## Steps

### Step 1: Update renderTwoColumnView to pass focus state

```go
// renderTwoColumnView renders the list and detail side by side with app-global footer
func (a *App) renderTwoColumnView() string {
	leftWidth, rightWidth := calculatePaneWidths(a.width)
	contentHeight := a.height - 1 // Reserve 1 line for footer

	// Determine focus states
	listFocused := a.state == viewListFocused
	linksFocused := a.state == viewDetailLinksFocused
	bodyFocused := a.state == viewDetailBodyFocused

	// Render left pane (list) with focus-dependent border
	leftPane := a.list.ViewConstrained(leftWidth, contentHeight, listFocused)

	// Render right pane (detail) with focus-dependent borders
	a.detail.linksFocused = linksFocused
	a.detail.bodyFocused = bodyFocused
	a.detail.width = rightWidth
	a.detail.height = contentHeight
	rightPane := a.detail.View()

	// Compose columns
	columns := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)

	// App-global footer based on focused area
	footer := a.renderFooter()

	return columns + "\n" + footer
}
```

### Step 2: Add renderFooter method

```go
// renderFooter returns the footer help text based on current focus state
func (a *App) renderFooter() string {
	switch a.state {
	case viewListFocused:
		return a.list.Footer()
	case viewDetailLinksFocused:
		return a.renderDetailLinksFooter()
	case viewDetailBodyFocused:
		return a.renderDetailBodyFooter()
	default:
		return ""
	}
}

func (a *App) renderDetailLinksFooter() string {
	return helpKeyStyle.Render("tab") + " " + helpStyle.Render("switch") + "  " +
		helpKeyStyle.Render("/") + " " + helpStyle.Render("filter") + "  " +
		helpKeyStyle.Render("enter") + " " + helpStyle.Render("go to") + "  " +
		helpKeyStyle.Render("j/k") + " " + helpStyle.Render("navigate") + "  " +
		helpKeyStyle.Render("backspace") + " " + helpStyle.Render("back") + "  " +
		helpKeyStyle.Render("b") + " " + helpStyle.Render("blocking") + "  " +
		helpKeyStyle.Render("e") + " " + helpStyle.Render("edit") + "  " +
		helpKeyStyle.Render("p") + " " + helpStyle.Render("parent") + "  " +
		helpKeyStyle.Render("P") + " " + helpStyle.Render("priority") + "  " +
		helpKeyStyle.Render("s") + " " + helpStyle.Render("status") + "  " +
		helpKeyStyle.Render("t") + " " + helpStyle.Render("type") + "  " +
		helpKeyStyle.Render("y") + " " + helpStyle.Render("copy id") + "  " +
		helpKeyStyle.Render("?") + " " + helpStyle.Render("help") + "  " +
		helpKeyStyle.Render("q") + " " + helpStyle.Render("quit")
}

func (a *App) renderDetailBodyFooter() string {
	footer := helpKeyStyle.Render("tab") + " " + helpStyle.Render("switch") + "  " +
		helpKeyStyle.Render("j/k") + " " + helpStyle.Render("scroll") + "  " +
		helpKeyStyle.Render("backspace") + " " + helpStyle.Render("back") + "  " +
		helpKeyStyle.Render("b") + " " + helpStyle.Render("blocking") + "  " +
		helpKeyStyle.Render("e") + " " + helpStyle.Render("edit") + "  " +
		helpKeyStyle.Render("p") + " " + helpStyle.Render("parent") + "  " +
		helpKeyStyle.Render("P") + " " + helpStyle.Render("priority") + "  " +
		helpKeyStyle.Render("s") + " " + helpStyle.Render("status") + "  " +
		helpKeyStyle.Render("t") + " " + helpStyle.Render("type") + "  " +
		helpKeyStyle.Render("y") + " " + helpStyle.Render("copy id") + "  " +
		helpKeyStyle.Render("?") + " " + helpStyle.Render("help") + "  " +
		helpKeyStyle.Render("q") + " " + helpStyle.Render("quit")
	// Only show tab switch if there are links
	if len(a.detail.links) == 0 {
		footer = helpKeyStyle.Render("j/k") + " " + helpStyle.Render("scroll") + "  " +
			helpKeyStyle.Render("backspace") + " " + helpStyle.Render("back") + "  " +
			// ... rest of shortcuts without tab
	}
	return footer
}
```

### Step 3: Update View() for narrow mode

```go
func (a *App) View() string {
	switch a.state {
	case viewListFocused:
		if a.isTwoColumnMode() {
			return a.renderTwoColumnView()
		}
		return a.list.View()
		
	case viewDetailLinksFocused, viewDetailBodyFocused:
		if a.isTwoColumnMode() {
			return a.renderTwoColumnView()
		}
		// Narrow mode: show only detail at full width
		a.detail.linksFocused = a.state == viewDetailLinksFocused
		a.detail.bodyFocused = a.state == viewDetailBodyFocused
		a.detail.width = a.width
		a.detail.height = a.height - 1
		return a.detail.View() + "\n" + a.renderFooter()
		
	case viewTagPicker:
		return a.tagPicker.View()
	// ... rest of picker cases unchanged, but update getBackgroundView
	}
	return ""
}
```

### Step 4: Update getBackgroundView

```go
func (a *App) getBackgroundView() string {
	switch a.previousState {
	case viewListFocused:
		if a.isTwoColumnMode() {
			return a.renderTwoColumnView()
		}
		return a.list.View()
	case viewDetailLinksFocused, viewDetailBodyFocused:
		if a.isTwoColumnMode() {
			return a.renderTwoColumnView()
		}
		return a.detail.View()
	default:
		return a.list.View()
	}
}
```

### Step 5: Build and test manually

Run: `mise build && mise beans`
Test: 
- Navigate with j/k (list should have primary border)
- Press Enter (detail should get primary border, list muted)
- Press Tab (toggle between links and body focus)
- Press Backspace (return to list)
- Resize terminal below 120 cols (should show single pane)

### Step 6: Commit

```bash
git add internal/tui/tui.go
git commit -m "feat(tui): two-column view with focus-based borders and footers

- Render both panes with focus-dependent border colors
- Footer changes based on which area is focused
- Narrow mode shows single pane at full width

Refs: beans-pn6z"
```