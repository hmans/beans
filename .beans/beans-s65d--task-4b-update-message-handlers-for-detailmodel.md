---
# beans-s65d
title: 'Task 4b: Update message handlers for detailModel'
status: completed
type: task
created_at: 2025-12-30T16:46:56Z
updated_at: 2025-12-30T17:15:00Z
parent: beans-pn6z
---

## Overview

Update the message handlers in tui.go that were using previewModel to work with detailModel instead.

**Note:** Do this task after Task 4 (beans-ozhq) and before Task 5 (beans-lmk4).

## Files

- Modify: `internal/tui/tui.go`

## Steps

### Step 1: Update beansLoadedMsg handler

When beans are first loaded, create the initial detailModel:

```go
case beansLoadedMsg:
	// Forward to list view
	a.list, cmd = a.list.Update(msg)
	
	// Create initial detailModel for selected bean
	_, rightWidth := calculatePaneWidths(a.width)
	if len(msg.items) == 0 {
		// Empty list - create detail with nil bean
		a.detail = newDetailModel(nil, a.resolver, a.config, rightWidth, a.height-2, false, false)
	} else if item, ok := a.list.list.SelectedItem().(beanItem); ok {
		// Default to body focused (links focused if bean has links will be set on Enter)
		a.detail = newDetailModel(item.bean, a.resolver, a.config, rightWidth, a.height-2, false, false)
	}
	return a, cmd
```

### Step 2: Update beansChangedMsg handler

Update to check for new view states:

```go
case beansChangedMsg:
	// Beans changed on disk - refresh
	if a.state == viewDetailLinksFocused || a.state == viewDetailBodyFocused {
		// Try to reload the current bean via GraphQL
		if a.detail.bean \!= nil {
			updatedBean, err := a.resolver.Query().Bean(context.Background(), a.detail.bean.ID)
			if err \!= nil || updatedBean == nil {
				// Bean was deleted - return to list
				a.state = viewListFocused
				a.history = nil
			} else {
				// Recreate detail view with fresh bean data
				linksFocused := a.state == viewDetailLinksFocused
				bodyFocused := a.state == viewDetailBodyFocused
				_, rightWidth := calculatePaneWidths(a.width)
				a.detail = newDetailModel(updatedBean, a.resolver, a.config, rightWidth, a.height-2, linksFocused, bodyFocused)
			}
		}
	}
	// Trigger list refresh
	return a, a.list.loadBeans
```

### Step 3: Update WindowSizeMsg handler

Update detail dimensions on resize:

```go
case tea.WindowSizeMsg:
	a.width = msg.Width
	a.height = msg.Height

	// Update detail dimensions
	_, rightWidth := calculatePaneWidths(a.width)
	if a.detail.bean \!= nil {
		// Preserve focus state when resizing
		linksFocused := a.state == viewDetailLinksFocused
		bodyFocused := a.state == viewDetailBodyFocused
		a.detail = newDetailModel(a.detail.bean, a.resolver, a.config, rightWidth, a.height-2, linksFocused, bodyFocused)
	}
```

### Step 4: Handle empty list in Enter handler

In the Enter handler (Task 5), add a guard:

```go
if a.state == viewListFocused && msg.String() == "enter" {
	if a.list.list.FilterState() \!= list.Filtering {
		// Guard: do nothing if no bean selected
		item, ok := a.list.list.SelectedItem().(beanItem)
		if \!ok || item.bean == nil {
			return a, nil
		}
		// ... rest of Enter handling
	}
}
```

### Step 5: Update detail.go to handle nil bean

In newDetailModel, handle nil bean gracefully:

```go
func newDetailModel(b *bean.Bean, resolver *graph.Resolver, cfg *config.Config, width, height int, linksFocused, bodyFocused bool) detailModel {
	m := detailModel{
		bean:         b,
		resolver:     resolver,
		config:       cfg,
		width:        width,
		height:       height,
		ready:        true,
		linksFocused: linksFocused,
		bodyFocused:  bodyFocused,
	}
	
	if b == nil {
		// Empty state - no links, empty viewport
		return m
	}
	
	// ... rest of initialization for non-nil bean
}
```

And update View() to show empty state:

```go
func (m detailModel) View() string {
	if m.bean == nil {
		return m.renderEmpty()
	}
	// ... rest of View
}

func (m detailModel) renderEmpty() string {
	style := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center).
		Foreground(ui.ColorMuted)
	return style.Render("No bean selected")
}
```

### Step 6: Build and verify

Run: `mise build`
Expected: Build succeeds

### Step 7: Commit

```bash
git add internal/tui/tui.go internal/tui/detail.go
git commit -m "fix(tui): update message handlers for detailModel

- beansLoadedMsg creates initial detailModel
- beansChangedMsg checks new view states
- WindowSizeMsg updates detail dimensions
- Handle nil bean gracefully (empty list)

Refs: beans-pn6z"
```