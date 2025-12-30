---
# beans-7vrp
title: 'Task 7: Update history navigation for unified view'
status: completed
type: task
priority: normal
created_at: 2025-12-30T16:37:42Z
updated_at: 2025-12-30T18:18:57Z
parent: beans-pn6z
---

## Overview

Update the history stack to store bean IDs instead of detailModel instances, and implement navigation when following links.

## Files

- Modify: `internal/tui/tui.go`

## Steps

### Step 1: Change history type in App struct

```go
type App struct {
	// ... other fields ...
	history []string // stack of bean IDs for back navigation
	// Remove: history []detailModel
}
```

### Step 2: Add helper method to move cursor to a bean

```go
// moveCursorToBean moves the list cursor to the bean with the given ID
func (a *App) moveCursorToBean(beanID string) {
	items := a.list.list.Items()
	for i, item := range items {
		if bi, ok := item.(beanItem); ok && bi.bean.ID == beanID {
			a.list.list.Select(i)
			break
		}
	}
}
```

### Step 3: Update link navigation (Enter on link)

In the detail models Update() or in App Update(), when a link is followed:

```go
// In App.Update(), handle selectBeanMsg differently
case selectBeanMsg:
	// Push current bean to history before navigating
	if a.detail.bean \!= nil {
		a.history = append(a.history, a.detail.bean.ID)
	}
	// Move cursor to new bean (will trigger cursorChangedMsg)
	a.moveCursorToBean(msg.bean.ID)
	// Stay in detail focus (links or body based on new bean)
	// The detail will be recreated via cursorChangedMsg
	return a, nil
```

### Step 4: Update Backspace handling

Already done in Task 5, but verify:

```go
if msg.String() == "backspace" {
	if a.state == viewDetailLinksFocused || a.state == viewDetailBodyFocused {
		if len(a.history) > 0 {
			prevBeanID := a.history[len(a.history)-1]
			a.history = a.history[:len(a.history)-1]
			a.moveCursorToBean(prevBeanID)
			// Stay in current detail focus state
			return a, nil
		}
		a.state = viewListFocused
		return a, nil
	}
}
```

### Step 5: Update cursorChangedMsg handler

When cursor changes, recreate detailModel:

```go
case cursorChangedMsg:
	if msg.beanID \!= "" {
		bean, err := a.resolver.Query().Bean(context.Background(), msg.beanID)
		if err == nil && bean \!= nil {
			// Recreate detail with current focus state
			linksFocused := a.state == viewDetailLinksFocused
			bodyFocused := a.state == viewDetailBodyFocused
			_, rightWidth := calculatePaneWidths(a.width)
			a.detail = newDetailModel(bean, a.resolver, a.config, rightWidth, a.height-2, linksFocused, bodyFocused)
		}
	}
	return a, nil
```

### Step 6: Remove old backToListMsg handling

Remove the `backToListMsg` handler that used to pop from history - we now handle this differently.

### Step 7: Build and test

Run: `mise build && mise beans`
Test:
- Enter a bean with links
- Follow a link (Enter on linked bean)
- Press Backspace (should go back to previous bean)
- Press Backspace again (should return to list)

### Step 8: Commit

```bash
git add internal/tui/tui.go
git commit -m "feat(tui): update history navigation for unified view

- History stores bean IDs instead of detailModel instances
- Following links pushes current bean to history
- Backspace navigates history, then returns to list
- moveCursorToBean helper for navigation

Refs: beans-pn6z"
```