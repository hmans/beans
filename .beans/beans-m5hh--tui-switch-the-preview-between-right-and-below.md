---
# beans-m5hh
title: 'TUI: switch the preview between right and below'
status: completed
type: feature
priority: normal
created_at: 2026-10-02T05:47:32Z
updated_at: 2026-10-02T05:49:44Z
---

Make the TUI preview position switchable between right of the list and below the list.

- Key `v` in the list view toggles right/below for the session (not persisted).
- Config `tui.preview_position: auto | right | below` (default auto = size-based selection).
- If the chosen layout does not fit, fall back to the other one, else list only.

## Todo
- [x] Config option tui.preview_position
- [x] Layout selection honours the preference, with fallback
- [x] Key v toggles, help overlay + footer hint
- [x] Tests

## Summary of Changes

- `pkg/config`: `tui.preview_position` (`auto`, `right`, `below`), written by `Save` when set; `GetPreviewPosition()` returns `auto` when unset or invalid.
- `internal/tui`: `isTwoColumnMode` honours the position; `right` needs `TwoColumnMinWidth`, `below` needs `stackedMinHeight()`, otherwise the other layout applies, then the list alone. `auto` keeps the size-based selection.
- Key `v` in the list view toggles right/below for the session (not persisted); listed in the help overlay and the footer.
- Tests: layout table per position, toggle test, config getter and save round trip.
