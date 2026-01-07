---
# beans-l4ht
title: Fix missing space between bean ID and type in detail view
status: completed
type: bug
priority: normal
created_at: 2026-01-07T18:37:16Z
updated_at: 2026-01-07T18:38:00Z
---

In internal/tui/detail.go, when displaying linked beans, there's no space between the bean ID column and the type column in the rendered output. This makes the display look cramped.

## Checklist
- [x] Add space separator between idCol and typeCol in RenderBeanRow function
- [x] Add space separator between typeCol and statusCol for consistency
- [x] Test the fix in the TUI