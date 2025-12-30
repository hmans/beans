---
# beans-p63f
title: 'Task 8: Simplify detail.go and clean up keyboard handling'
status: todo
type: task
created_at: 2025-12-30T16:38:04Z
updated_at: 2025-12-30T16:38:04Z
parent: beans-pn6z
---

## Overview

Simplify detail.go to only handle its internal concerns. App now handles focus switching, navigation, and quit. Also ensure only `q` quits the app.

**Note:** This task consolidates what was previously Task 8 and Task 9 (beans-oms6 is now redundant).

## Files

- Modify: `internal/tui/detail.go`
- Modify: `internal/tui/tui.go`
- Modify: `internal/tui/list.go`

## Steps

### Step 1: Simplify detail.go Update method

The detail model should only handle:
- j/k for scrolling body or navigating links (based on focus params)
- Enter to emit selectBeanMsg when on a link (only when links focused)
- Edit shortcuts (p, s, t, P, b, e, y)
- / for filtering links (only when links focused)

Remove from detail.go Update():
- `case "esc", "backspace":` handler (App handles navigation)
- `case "tab":` handler (App handles focus switching)
- `case "q":` handler (App handles quit)

```go
func (m detailModel) Update(msg tea.Msg) (detailModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// ... existing resize handling ...

	case tea.KeyMsg:
		// If links focused and filtering, let link list handle all keys
		if m.linksFocused && m.linkList.FilterState() == list.Filtering {
			m.linkList, cmd = m.linkList.Update(msg)
			return m, cmd
		}

		switch msg.String() {
		case "enter":
			// Navigate to selected link (only when links focused)
			if m.linksFocused {
				if item, ok := m.linkList.SelectedItem().(linkItem); ok {
					targetBean := item.link.bean
					return m, func() tea.Msg {
						return selectBeanMsg{bean: targetBean}
					}
				}
			}

		// Edit shortcuts - always available
		case "p":
			return m, func() tea.Msg {
				return openParentPickerMsg{
					beanIDs:       []string{m.bean.ID},
					beanTitle:     m.bean.Title,
					beanTypes:     []string{m.bean.Type},
					currentParent: m.bean.Parent,
				}
			}
		case "s":
			return m, func() tea.Msg {
				return openStatusPickerMsg{
					beanIDs:       []string{m.bean.ID},
					beanTitle:     m.bean.Title,
					currentStatus: m.bean.Status,
				}
			}
		case "t":
			return m, func() tea.Msg {
				return openTypePickerMsg{
					beanIDs:     []string{m.bean.ID},
					beanTitle:   m.bean.Title,
					currentType: m.bean.Type,
				}
			}
		case "P":
			return m, func() tea.Msg {
				return openPriorityPickerMsg{
					beanIDs:         []string{m.bean.ID},
					beanTitle:       m.bean.Title,
					currentPriority: m.bean.Priority,
				}
			}
		case "b":
			return m, func() tea.Msg {
				return openBlockingPickerMsg{
					beanID:          m.bean.ID,
					beanTitle:       m.bean.Title,
					currentBlocking: m.bean.Blocking,
				}
			}
		case "e":
			return m, func() tea.Msg {
				return openEditorMsg{
					beanID:   m.bean.ID,
					beanPath: m.bean.Path,
				}
			}
		case "y":
			return m, func() tea.Msg {
				return copyBeanIDMsg{ids: []string{m.bean.ID}}
			}
		}
	}

	// Forward updates to the appropriate component based on focus
	if m.linksFocused && len(m.links) > 0 {
		m.linkList, cmd = m.linkList.Update(msg)
	} else if m.bodyFocused {
		m.viewport, cmd = m.viewport.Update(msg)
	}

	return m, cmd
}
```

### Step 2: Remove footer from detail.go View()

App renders the footer, so detail should not:

```go
func (m detailModel) View() string {
	if !m.ready {
		return "Loading..."
	}

	if m.bean == nil {
		return m.renderEmpty()
	}

	header := m.renderHeader()

	var linksSection string
	if len(m.links) > 0 {
		linksBorderColor := ui.ColorMuted
		if m.linksFocused {
			linksBorderColor = ui.ColorPrimary
		}
		linksBorder := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(linksBorderColor).
			Width(m.width - 4)
		linksSection = linksBorder.Render(m.linkList.View()) + "\n"
	}

	bodyBorderColor := ui.ColorMuted
	if m.bodyFocused {
		bodyBorderColor = ui.ColorPrimary
	}
	bodyBorder := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(bodyBorderColor).
		Width(m.width - 4)
	body := bodyBorder.Render(m.viewport.View())

	// No footer - App renders footer separately
	return header + "\n" + linksSection + body
}
```

### Step 3: Verify list.go Esc behavior

In list.go, Esc should only:
1. Clear selection if any beans selected
2. Clear filter if active

It should NOT quit. Verify this is the case (it already is).

### Step 4: Verify tui.go q handling

Already done in Task 6 (beans-lmk4), but verify q only quits when not filtering.

### Step 5: Build and test

Run: `mise build && mise beans`
Test:
- Press Esc in list with no selection/filter (nothing should happen)
- Press Esc in detail (nothing should happen)
- Press q anywhere (should quit)
- Press Esc with selection (should clear selection)
- Press Esc with filter (should clear filter)
- Edit shortcuts (p, s, t, P, b, e, y) work in detail

### Step 6: Commit

```bash
git add internal/tui/detail.go internal/tui/tui.go internal/tui/list.go
git commit -m "refactor(tui): simplify detail.go, only q quits

- Detail handles scrolling, link navigation, edit shortcuts only
- App handles focus switching (Tab), navigation (Backspace), quit (q)
- Footer rendered by App, not detail
- Esc is for cancel/clear only (list)

Refs: beans-pn6z"
```