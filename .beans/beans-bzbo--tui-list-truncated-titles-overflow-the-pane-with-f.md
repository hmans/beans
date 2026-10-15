---
# beans-bzbo
title: 'TUI list: truncated titles overflow the pane with full type/status names'
status: completed
type: bug
created_at: 2026-10-01T13:55:48Z
updated_at: 2026-10-01T13:55:48Z
---

CalculateResponsiveColumns reserved 10 columns for the full type name, RenderBeanRow rendered it 12 wide. Truncated rows were 2 columns too wide and the pane border wrapped the "..." onto its own line. Visible since the stacked layout gives the list >= 120 columns.

- [x] Shared ColWidthStatusFull/ColWidthTypeFull constants for calculation and rendering
- [x] TestListRowsFitPane

## Summary of Changes

- internal/ui/styles.go: ColWidthStatusFull = ColWidthTypeFull = 12, used by CalculateResponsiveColumns and RenderBeanRow.
- TestListRowsFitPane renders the list pane at 80/120/124/160 columns and requires every line to be exactly the pane width.
