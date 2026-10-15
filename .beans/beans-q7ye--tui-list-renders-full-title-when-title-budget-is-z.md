---
# beans-q7ye
title: TUI list renders full title when title budget is zero
status: todo
type: bug
priority: normal
created_at: 2026-10-06T13:19:37Z
updated_at: 2026-10-07T07:19:20Z
---

`RenderBeanRow` (`internal/ui/styles.go:585-589`) truncates a title only when `MaxTitleWidth > 0`. The list delegate computes the budget as `max(0, m.Width()-baseWidth)` (`internal/tui/list.go:72`). When the ID, status, type and tag columns already fill the row, the budget is 0, and the title is rendered in full.

Example: in the two-column layout at 120x27 the list is 37 columns wide. With tree depth 6 the ID column is 30 wide, `baseWidth` is 40, and the budget is 0. A 41-character title then makes the row 81 columns wide.

The pane clip in `viewContent` (beans-1q77) keeps the layout intact, but cuts such rows hard at the border without `...`.

## Todo

- [x] Decide what a row shows when the title budget is 0 or less (see Decision)
- [ ] `isTwoColumnMode` (`internal/tui/tui.go`) chooses two columns only if the left pane gives a title budget of at least `MinTitleWidth`
- [ ] `isTwoColumnMode` and the list delegate both compute the budget with `ui.TitleBudget` (beans-4x0o)
- [ ] `RenderBeanRow`: a budget of 0 or less no longer means "no truncation"; the list delegate clamps the budget to `MinTitleWidth`; check the TUI detail view caller (`internal/tui/detail.go:137`)
- [ ] Table-driven layout test: a tree depth that gives the 120x27 left pane a budget below 3 selects below or list-only
- [ ] Table-driven test: deep tree in a narrow pane; with a budget of at least `MinTitleWidth` every row is exactly the list width and ends in `...` or a fully shown title, below it the pane keeps its size
- [ ] `mise test`

## Decision

`MinTitleWidth := 3`, the smallest title budget a list row may get. The title budget is the list inner width minus the ID, status, type and tags columns and 4 for cursor and padding.

- Preview on the right only if the left pane gives a title budget of at least `MinTitleWidth`.
- Otherwise the preview goes below, if the terminal is tall enough (`stackedMinHeight`, unchanged).
- Otherwise only the bean list remains.

This holds for a `preview.position: right` setting too. The ID column grows by 3 per tree level, so the layout depends on the tree depth of the loaded beans, not only on the terminal size.

The list alone has the full terminal width and no further fallback. There a budget below `MinTitleWidth` is clamped to `MinTitleWidth`, and the pane clip in `viewContent` cuts the row at the border (at depth 6, a terminal under 45 columns).
