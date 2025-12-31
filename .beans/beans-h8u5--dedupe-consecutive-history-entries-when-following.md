---
# beans-h8u5
title: Dedupe consecutive history entries when following links
status: draft
type: task
priority: low
created_at: 2025-12-30T18:30:03Z
updated_at: 2025-12-30T18:34:12Z
parent: beans-pn6z
---

## Problem

If a user repeatedly follows the same link back and forth, the history stack contains duplicate consecutive entries. For example:

1. User is viewing bean A
2. User follows link to bean B (history: [A])
3. User follows link back to bean A (history: [A, B])
4. User follows link to bean B again (history: [A, B, A])

This creates a longer history than necessary.

## Current Code

```go
case selectBeanMsg:
    // Push current bean ID to history before navigating
    if a.detail.bean != nil {
        a.history = append(a.history, a.detail.bean.ID)
    }
    // Move cursor to new bean (will trigger cursorChangedMsg)
    a.moveCursorToBean(msg.bean.ID)
    return a, nil
```

## Suggested Fix

Check if the new bean is the same as the last history entry before pushing:

```go
case selectBeanMsg:
    if a.detail.bean != nil {
        // Avoid duplicate consecutive entries
        if len(a.history) == 0 || a.history[len(a.history)-1] != a.detail.bean.ID {
            a.history = append(a.history, a.detail.bean.ID)
        }
    }
    a.moveCursorToBean(msg.bean.ID)
    return a, nil
```

## Impact

- Works correctly without this fix, just creates unnecessary history entries
- Minor optimization, not required for correctness

## Files

- `internal/tui/tui.go` (selectBeanMsg handler)

## Context

Identified during code review of beans-pn6z (unified detail view refactoring).