---
# beans-mvme
title: 'Task 1: Update view states to granular focus states'
status: completed
type: task
priority: normal
created_at: 2025-12-30T16:35:29Z
updated_at: 2025-12-30T16:59:55Z
parent: beans-pn6z
---

## Overview

Replace `viewList` and `viewDetail` with granular focus states.

## Files

- Modify: `internal/tui/tui.go:22-35`

## Steps

### Step 1: Update viewState constants

Replace the current view states:

```go
// viewState represents which view is currently active
type viewState int

const (
	viewListFocused viewState = iota
	viewDetailLinksFocused
	viewDetailBodyFocused
	viewTagPicker
	viewParentPicker
	viewStatusPicker
	viewTypePicker
	viewBlockingPicker
	viewPriorityPicker
	viewCreateModal
	viewHelpOverlay
)
```

### Step 2: Update App.state initialization

In `New()`, change:
```go
state: viewListFocused,
```

### Step 3: Build and verify no compile errors

Run: `mise build`
Expected: Build succeeds (there will be runtime issues until we update the rest)

### Step 4: Commit

```bash
git add internal/tui/tui.go
git commit -m "refactor(tui): replace viewList/viewDetail with granular focus states

Refs: beans-pn6z"
```