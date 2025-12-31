---
# beans-lmk4
title: 'Task 5: Update keyboard routing for granular focus states'
status: completed
type: task
priority: normal
created_at: 2025-12-30T16:36:43Z
updated_at: 2025-12-30T17:43:01Z
parent: beans-pn6z
---

## Overview

Update tui.go Update() to route keyboard events based on the new granular view states.

## Files

- Modify: `internal/tui/tui.go` (Update method)

## Steps

### Step 1: Update global key handling

In `Update()`, update the switch for global keys (ctrl+c, ?, q):

```go
case tea.KeyMsg:
	// Clear status messages on any keypress
	a.list.statusMessage = ""
	a.detail.statusMessage = ""

	switch msg.String() {
	case "ctrl+c":
		return a, tea.Quit
	case "q":
		// q always quits (except when filtering)
		if a.state == viewListFocused && a.list.list.FilterState() == list.Filtering {
			break // let list handle it
		}
		if a.state == viewDetailLinksFocused && a.detail.linkList.FilterState() == list.Filtering {
			break // let detail handle it
		}
		return a, tea.Quit
	case "?":
		// Open help overlay from list or detail states
		if a.state == viewListFocused || a.state == viewDetailLinksFocused || a.state == viewDetailBodyFocused {
			a.previousState = a.state
			a.helpOverlay = newHelpOverlayModel(a.width, a.height)
			a.state = viewHelpOverlay
			return a, a.helpOverlay.Init()
		}
	}
```

### Step 2: Add Enter handler for viewListFocused

When Enter is pressed in list, focus the detail pane:

```go
case tea.KeyMsg:
	// ... after global keys ...
	
	// Handle Enter from list - focus detail
	if a.state == viewListFocused && msg.String() == "enter" {
		if a.list.list.FilterState() \!= list.Filtering {
			if item, ok := a.list.list.SelectedItem().(beanItem); ok {
				// Focus links if bean has links, else focus body
				if len(a.detail.links) > 0 {
					a.state = viewDetailLinksFocused
				} else {
					a.state = viewDetailBodyFocused
				}
				return a, nil
			}
		}
	}
```

### Step 3: Add Tab handler for detail states

```go
	// Handle Tab in detail - toggle between links and body
	if msg.String() == "tab" {
		if a.state == viewDetailLinksFocused {
			a.state = viewDetailBodyFocused
			return a, nil
		} else if a.state == viewDetailBodyFocused && len(a.detail.links) > 0 {
			a.state = viewDetailLinksFocused
			return a, nil
		}
	}
```

### Step 4: Add moveCursorToBean helper method

Add this helper method to App (needed for history navigation and link following):

```go
// moveCursorToBean moves the list cursor to the bean with the given ID.
// This triggers cursorChangedMsg which updates the detail pane.
func (a *App) moveCursorToBean(beanID string) {
	items := a.list.list.Items()
	for i, item := range items {
		if bi, ok := item.(beanItem); ok && bi.bean.ID == beanID {
			a.list.list.Select(i)
			return
		}
	}
}
```

### Step 5: Add Backspace handler for detail states

```go
	// Handle Backspace in detail - navigate history or return to list
	if msg.String() == "backspace" {
		if a.state == viewDetailLinksFocused || a.state == viewDetailBodyFocused {
			// Check history first
			if len(a.history) > 0 {
				// Pop from history, move cursor to that bean
				prevBeanID := a.history[len(a.history)-1]
				a.history = a.history[:len(a.history)-1]
				// Move list cursor to that bean (will trigger cursor change and detail update)
				a.moveCursorToBean(prevBeanID)
				// Stay in detail focus
				return a, nil
			}
			// No history - return to list
			a.state = viewListFocused
			return a, nil
		}
	}
```

### Step 6: Remove old viewList/viewDetail case handling

Remove or update the old switch cases:
- Remove `case "q":` checks for `viewDetail` (now handled above)
- Update the message forwarding at the bottom of Update()

### Step 7: Update message forwarding

At the bottom of Update(), route messages to appropriate model:

```go
// Forward all messages to the current view
switch a.state {
case viewListFocused:
	a.list, cmd = a.list.Update(msg)
case viewDetailLinksFocused, viewDetailBodyFocused:
	a.detail, cmd = a.detail.Update(msg)
case viewTagPicker:
	a.tagPicker, cmd = a.tagPicker.Update(msg)
// ... rest unchanged
}
```

### Step 8: Build and test

Run: `mise build`
Expected: Build succeeds

### Step 9: Commit

```bash
git add internal/tui/tui.go
git commit -m "feat(tui): route keyboard events based on granular focus states

- Enter from list focuses detail (links if present, else body)
- Tab toggles between links and body in detail
- Backspace navigates history then returns to list
- q quits from any state (except when filtering)
- ? opens help from list/detail states

Refs: beans-pn6z"
```