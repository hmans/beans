---
# beans-238n
title: 'Task 2: Delete preview.go and update App struct'
status: todo
type: task
created_at: 2025-12-30T16:35:44Z
updated_at: 2025-12-30T16:35:44Z
parent: beans-pn6z
---

## Overview

Remove the preview model since we will use detailModel in the right pane.

## Files

- Delete: `internal/tui/preview.go`
- Modify: `internal/tui/tui.go:98-128` (App struct)

## Steps

### Step 1: Delete preview.go

```bash
rm internal/tui/preview.go
```

### Step 2: Remove preview field from App struct

In `tui.go`, remove from App struct:
```go
// Remove this line:
preview        previewModel
```

### Step 3: Remove preview initialization in New()

Remove:
```go
preview:  newPreviewModel(nil, 0, 0),
```

### Step 4: Comment out preview-related code temporarily

In `Update()` and `View()`, comment out any code referencing `a.preview` - we will fix these in later tasks. Look for:
- `cursorChangedMsg` handler updating preview
- `beansLoadedMsg` handler updating preview  
- `tea.WindowSizeMsg` handler updating preview dimensions
- `renderTwoColumnView()` using preview

### Step 5: Build and verify

Run: `mise build`
Expected: Build succeeds (commented code will be replaced later)

### Step 6: Commit

```bash
git add -A
git commit -m "refactor(tui): remove preview.go, use detailModel in right pane

- Delete preview.go (no longer needed)
- Remove preview field from App struct
- Comment out preview references (will be replaced with detail)

Refs: beans-pn6z"
```