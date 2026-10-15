---
# beans-sg3r
title: 'TUI: configurable list height in stacked layout'
status: completed
type: feature
priority: normal
created_at: 2026-10-01T14:16:23Z
updated_at: 2026-10-01T14:18:45Z
---

Make the list pane height of the stacked TUI layout (preview below the list) configurable via `tui.stacked_list_height` in .beans.yml.

- Rows, border included; missing/0 → 15, values below 5 clamp to 5.
- Stacking threshold follows: list height + 20 (preview keeps its room).

## Todo
- [x] Config field + getter + tests
- [x] TUI uses configured height + threshold
- [x] Update TUI tests
- [x] ~~Docs~~ (no config reference docs exist; the field comment and the .beans.yml HeadComment document it)

## Summary of Changes

- `pkg/config`: new `TUIConfig` with `stacked_list_height`, written by `Save` when set; `GetStackedListHeight()` returns 15 when unset and raises values below 5 to 5.
- `internal/tui`: `StackedListHeight`/`StackedMinHeight` constants are replaced by `App.stackedListHeight` from the config and `stackedMinHeight()` = list height + `StackedBelowListMinHeight` (20).
- Tests: config getter table plus load/save round trip; stacked layout cases with configured heights 25 and 8.
