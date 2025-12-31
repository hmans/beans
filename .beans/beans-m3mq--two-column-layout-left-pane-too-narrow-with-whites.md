---
# beans-m3mq
title: 'Two-column layout: left pane too narrow with whitespace gap'
status: completed
type: bug
priority: normal
created_at: 2025-12-28T19:20:10Z
updated_at: 2025-12-28T19:49:19Z
parent: beans-t0tv
---

The left pane only takes up ~40 chars instead of the intended 55 chars. There's significant whitespace between the left and right panes.

## Screenshot

```
╭─────────────────────────────────────────────────────╮                              ╭─────────────────────────────
│ Beans                                               │                              │ beans-18db
│                                                     │                              │ beans milestones command
│ beans-f11p        M  I  Milestone 0.4.0             │                              │
│ ├─ beans-hz87     F  T  Add blocked-by relatio...   │                              │ Status: todo  Type: task
```

## Root Cause (investigation notes)

The issue is in `list.View()` (list.go:500-567):
- The border box is constrained to `m.width - 2`
- BUT the footer is appended as `content + "\n" + footer` without width constraint
- The footer line extends to full terminal width
- `lipgloss.JoinHorizontal` sees the left pane width as the footer width (unbounded), not the box width

## Fix

The footer needs to be constrained to the same width as the border box, or the entire View() output needs width clamping.