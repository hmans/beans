---
# beans-e8qv
title: 'TUI: show preview below the list on narrow, tall terminals'
status: completed
type: feature
priority: normal
created_at: 2026-10-01T13:13:55Z
updated_at: 2026-10-01T13:17:26Z
---

When the terminal is too narrow for two columns (< TwoColumnMinWidth) but at least 35 rows tall, show the preview below the list instead of hiding it.

- List pane fixed at 15 rows, preview gets the rest above the footer
- Preview spans the terminal width, net text width capped at 120

- [x] Stacked layout rendering
- [x] Mouse: click selects in list, wheel scrolls preview
- [x] Tests

## Summary of Changes

- isStackedMode/renderStackedView in tui.go: below TwoColumnMinWidth and at >= StackedMinHeight (35) rows, the list renders at StackedListHeight (15) rows with the preview below it.
- previewSize() is the single source of the preview's size for both layouts; mouse handling shares one over-preview branch for wheel scrolling.
- Stacked preview width is min(terminal width, StackedPreviewMaxWidth = 120 text + 8).
- Tests: TestStackedLayout, stacked cases in TestListClickSelectsItem.
