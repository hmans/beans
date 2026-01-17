---
# beans-vn93
title: Responsive type/status column expansion in TUI
status: completed
type: task
priority: normal
created_at: 2025-12-29T18:30:52Z
updated_at: 2025-12-29T18:39:15Z
parent: beans-t0tv
---

Show full type/status names (e.g., 'feature', 'in-progress') when terminal is wide enough (≥120 cols), single-letter abbreviations (F, I) when space is tight.

**Scope:** TUI only (not CLI output).

## Plan

1. Add `UseFullTypeStatus bool` to `ResponsiveColumns` struct
2. Update `CalculateResponsiveColumns()` to set flag when width ≥ 120
3. Update list delegate to pass `UseFullNames: d.cols.UseFullTypeStatus` to `RenderBeanRow`

## Files
- `internal/ui/styles.go` - ResponsiveColumns, CalculateResponsiveColumns
- `internal/tui/list.go` - itemDelegate.Render