---
# beans-u9vk
title: Add error handling to moveCursorToBean helper
status: draft
type: task
priority: low
created_at: 2025-12-30T18:30:02Z
updated_at: 2025-12-30T18:34:12Z
parent: beans-pn6z
---

## Problem

The `moveCursorToBean()` function in `internal/tui/tui.go` (lines 678-686) silently fails if the bean is not found in the list. This could happen if:

- The bean was deleted but still in history
- The list is filtered and the bean is not visible
- The bean hasn't been loaded yet

When this happens, the cursor doesn't move, no `cursorChangedMsg` is triggered, and the detail pane shows stale data.

## Current Code

```go
func (a *App) moveCursorToBean(beanID string) {
    items := a.list.list.Items()
    for i, item := range items {
        if bi, ok := item.(beanItem); ok && bi.bean.ID == beanID {
            a.list.list.Select(i)
            return
        }
    }
    // Silently returns if bean not found
}
```

## Suggested Approaches

1. Return a boolean indicating success/failure and handle accordingly in callers
2. Log a debug message when bean is not found
3. Add fallback behavior (e.g., clear detail pane, select first item, or skip to next history item)

## Mitigating Factors

- The `beansChangedMsg` handler already clears history when a bean is deleted
- In normal usage, beans in history should exist in the list
- The code doesn't crash, just shows stale data

## Files

- `internal/tui/tui.go`

## Context

Identified during code review of beans-pn6z (unified detail view refactoring).