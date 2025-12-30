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

Comment out these specific blocks (will be replaced in Task 5 - beans-s65d):

**In `tea.WindowSizeMsg` handler (~line 162-167):**
```go
// Comment out:
// if a.isTwoColumnMode() {
//     _, rightWidth := calculatePaneWidths(a.width)
//     a.preview.width = rightWidth
//     a.preview.height = a.height - 2
// }
```

**In `cursorChangedMsg` handler (~line 220-231):**
```go
// Comment out entire case:
// case cursorChangedMsg:
//     _, rightWidth := calculatePaneWidths(a.width)
//     if msg.beanID != "" {
//         bean, err := a.resolver.Query().Bean(context.Background(), msg.beanID)
//         if err == nil && bean != nil {
//             a.preview = newPreviewModel(bean, rightWidth, a.height-2)
//         }
//     } else {
//         a.preview = newPreviewModel(nil, rightWidth, a.height-2)
//     }
//     return a, nil
```

**In `beansLoadedMsg` handler (~line 237-242):**
```go
// Comment out preview update (keep the list update):
// _, rightWidth := calculatePaneWidths(a.width)
// if len(msg.items) == 0 {
//     a.preview = newPreviewModel(nil, rightWidth, a.height-2)
// } else if item, ok := a.list.list.SelectedItem().(beanItem); ok {
//     a.preview = newPreviewModel(item.bean, rightWidth, a.height-2)
// }
```

**In `renderTwoColumnView()` (~line 641-644):**
```go
// Comment out preview rendering:
// a.preview.width = rightWidth
// a.preview.height = contentHeight
// rightPane := a.preview.View()

// Temporarily replace with placeholder:
rightPane := "Detail placeholder"
```

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