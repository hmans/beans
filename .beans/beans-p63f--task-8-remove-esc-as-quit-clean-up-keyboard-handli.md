---
# beans-p63f
title: 'Task 8: Remove Esc as quit, clean up keyboard handling'
status: todo
type: task
created_at: 2025-12-30T16:38:04Z
updated_at: 2025-12-30T16:38:04Z
parent: beans-pn6z
---

## Overview

Ensure only `q` quits the app. Remove Esc from quit handling and ensure it only does cancel/clear actions.

## Files

- Modify: `internal/tui/tui.go`
- Modify: `internal/tui/detail.go`
- Modify: `internal/tui/list.go`

## Steps

### Step 1: Remove Esc quit from tui.go global handling

In `Update()`, ensure Esc is not in the quit handling:

```go
case "q":
	// q quits from list or detail (except when filtering)
	if a.state == viewListFocused && a.list.list.FilterState() == list.Filtering {
		break
	}
	if a.state == viewDetailLinksFocused && a.detail.linkList.FilterState() == list.Filtering {
		break
	}
	return a, tea.Quit
// Remove any case "esc" that returns tea.Quit
```

### Step 2: Update detail.go - remove esc/backspace quit

In detail.go Update(), remove:
```go
case "esc", "backspace":
	return m, func() tea.Msg {
		return backToListMsg{}
	}
```

The detail model should not handle navigation - that is now App's responsibility. Detail only handles:
- Internal scrolling (j/k when body focused)
- Link list navigation (j/k when links focused)
- Enter to emit selectBeanMsg when on a link
- Edit shortcuts (p, s, t, P, b, e, y)

### Step 3: Update detail.go - remove q quit

Remove the `case "q":` that quits - App handles this now.

### Step 4: Verify list.go Esc behavior

In list.go, Esc should only:
1. Clear selection if any beans selected
2. Clear filter if active

It should NOT quit. Verify this is the case.

### Step 5: Build and test

Run: `mise build && mise beans`
Test:
- Press Esc in list with no selection/filter (nothing should happen)
- Press Esc in detail (nothing should happen)
- Press q anywhere (should quit)
- Press Esc with selection (should clear selection)
- Press Esc with filter (should clear filter)

### Step 6: Commit

```bash
git add internal/tui/tui.go internal/tui/detail.go internal/tui/list.go
git commit -m "fix(tui): only q quits, esc is for cancel/clear only

- Remove esc from quit handling
- q is the only way to quit the app
- esc clears selection/filter in list only

Refs: beans-pn6z"
```