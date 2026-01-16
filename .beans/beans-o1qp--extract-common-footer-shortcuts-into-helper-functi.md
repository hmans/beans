---
# beans-o1qp
title: Extract common footer shortcuts into helper function
status: draft
type: task
priority: low
created_at: 2025-12-30T18:30:01Z
updated_at: 2025-12-30T18:34:12Z
parent: beans-pn6z
---

## Problem

The footer rendering functions in `internal/tui/tui.go` (lines 752-790) have repetitive code listing all the shortcuts. The only differences are tab/filter/navigation keys.

## Current State

- `renderDetailLinksFooter()` and `renderDetailBodyFooter()` repeat most shortcuts
- Common shortcuts: backspace, b, e, p, P, s, t, y, ?, q

## Suggested Fix

Extract common shortcuts into a helper function to reduce duplication:

```go
func (a *App) renderCommonDetailShortcuts() string {
    return helpKeyStyle.Render("backspace") + " " + helpStyle.Render("back") + "  " +
        helpKeyStyle.Render("b") + " " + helpStyle.Render("blocking") + "  " +
        // ... etc
}
```

## Files

- `internal/tui/tui.go`

## Context

Identified during code review of beans-pn6z (unified detail view refactoring).