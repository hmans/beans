---
# beans-0jew
title: 'TUI: left/right page jumps skip beans in the stacked and two-column layouts'
status: completed
type: bug
priority: normal
created_at: 2026-10-01T14:32:49Z
updated_at: 2026-10-01T14:35:03Z
---

The bubbles list paginates with PerPage computed from the full terminal height (listModel WindowSizeMsg), while ViewConstrained renders a smaller pane. left/right therefore jump by more rows than are visible in the stacked layout (and by one row off in two columns).

## Todo
- [x] Size the real list model to the rendered pane after each update
- [x] Test: next page lands on the first bean after the visible page

## Summary of Changes

- `App.listPaneSize()` returns the rendered list pane size; the mouse hit test uses it too.
- `App.fitListToPane()` sizes the bubbles list to that pane before every forwarded message and again after a resize, so `PerPage` matches the visible rows in all three layouts, also after a resize while another view was open.
- `list_paging_test.go`: right/left move by exactly one visible page in single-column, two-column and stacked layouts.
