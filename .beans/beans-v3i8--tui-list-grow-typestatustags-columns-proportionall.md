---
# beans-v3i8
title: 'TUI list: grow type/status/tags columns proportionally instead of jumping'
status: completed
type: feature
priority: normal
created_at: 2026-10-02T06:52:54Z
updated_at: 2026-10-02T06:55:25Z
---

Bean rows jump in layout as the pane widens: type/status switch from 3 to 12 columns at 120, the tags column appears at 140 and steps through fixed widths.

Target layout (pane width W):
- W < 120: type/status as one letter (3 columns each)
- W = 120: type/status names cut hard to 5 columns (`miles in-pr`)
- 120..160: type/status widths grow proportionally to full width (12)
- W = 140: tags column appears once (24 columns), then grows proportionally to 70 at W = 200

## Todo
- [x] CalculateResponsiveColumns: proportional widths
- [x] RenderBeanRow: render type/status at given column width
- [x] Callers (list, detail, tree) and docs (okf/tui-taxonomy.md)
- [x] Tests: widths monotone, rows fit pane

## Summary of Changes

- `CalculateResponsiveColumns` (internal/ui/styles.go): type/status widths interpolate 5 -> 12 between pane widths 120 and 160, taking turns so the title never loses more than it gains; the tags column appears at 140 with 24 columns and grows to 70 between 160 and 220. `MaxTags` follows from the tags width. `UseFullTypeStatus` is gone.
- `RenderBeanRow`: `TypeColWidth`/`StatusColWidth` replace `UseFullNames`; below 5 columns the one-letter code, otherwise the name cut hard to the column width.
- list.go passes the computed widths, detail.go the full widths; okf/tui-taxonomy.md updated.
- Tests: column table, title width never shrinks except at 120 and 140, row text per width, rows fit the pane at widths 80..250 with tags.
