---
# beans-xbqz
title: 'TUI: mouse support (wheel scroll in detail view, click to select in list)'
status: completed
type: feature
priority: normal
created_at: 2026-10-01T12:01:20Z
updated_at: 2026-10-01T12:48:03Z
---

Under tmux the TUI ignores the mouse wheel, because the program never requests mouse events.

- [x] Enable mouse cell motion in tea.NewProgram
- [x] Click on a list item selects it (no wheel scrolling in the list)
- [x] Tests
- [x] Wheel scrolls the body in the two-column preview pane
- [x] Manual check under tmux

## Summary of Changes

- Enable mouse events (`tea.WithMouseCellMotion`), so the TUI receives the wheel under tmux.
- Detail view: the mouse wheel always scrolls the body, also while the links list is focused.
- List/tree: left click selects the clicked bean; the wheel does nothing there. Hit-testing paginates a copy at the rendered pane size, so it matches both layouts.
- Two-column preview: the wheel over the preview scrolls the body (header stays fixed); the scroll resets on cursor change and survives a reload of the same bean.
- Tests: list_mouse_test.go, TestPreviewScroll, TestPreviewWheel.
