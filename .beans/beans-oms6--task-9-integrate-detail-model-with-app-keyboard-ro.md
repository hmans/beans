---
# beans-oms6
title: 'Task 9: Integrate detail model with App keyboard routing'
status: todo
type: task
created_at: 2025-12-30T16:38:33Z
updated_at: 2025-12-30T16:38:33Z
parent: beans-pn6z
---

## Overview

Ensure the detail model works correctly when keyboard events are routed from App. The detail model should handle only its internal concerns (scrolling, link navigation, edit shortcuts), while App handles focus switching and navigation.

## Files

- Modify: `internal/tui/detail.go`
- Modify: `internal/tui/tui.go`

## Steps

### Step 1: Simplify detail.go Update method

The detail model should handle:
- j/k for scrolling body or navigating links (based on focus params)
- Enter to emit selectBeanMsg when on a link (only when links focused)
- Edit shortcuts (p, s, t, P, b, e, y)
- / for filtering links (only when links focused)

Remove from detail.go Update():
- Tab handling (App does this)
- Esc/Backspace handling (App does this)
- q handling (App does this)

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
		// ... other edit shortcuts unchanged ...
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

The detail View() should NOT render its own footer - App handles this:

```go
func (m detailModel) View() string {
	if !m.ready {
		return "Loading..."
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

### Step 3: Adjust height calculations

Since footer is now rendered by App, update height calculations in detail:

```go
func newDetailModel(b *bean.Bean, resolver *graph.Resolver, cfg *config.Config, width, height int, linksFocused, bodyFocused bool) detailModel {
	// height is now the full content area (App already subtracted footer)
	// ... rest unchanged
}
```

### Step 4: Build and test

Run: `mise build && mise beans`
Test all interactions work correctly.

### Step 5: Commit

```bash
git add internal/tui/detail.go internal/tui/tui.go
git commit -m "refactor(tui): integrate detail model with App keyboard routing

- Detail handles scrolling, link navigation, edit shortcuts
- App handles focus switching (Tab), navigation (Backspace), quit (q)
- Footer rendered by App, not detail
- Focus params control border colors

Refs: beans-pn6z"
```