---
# beans-9bmv
title: 'TUI: toggle to hide completed/scrapped beans'
status: completed
type: feature
priority: normal
created_at: 2026-10-01T11:24:36Z
updated_at: 2026-10-01T11:26:23Z
---

Add a toggle key (h) in the TUI list view that hides beans with an archive status (completed, scrapped), including beans whose ancestor has such a status. Visible by default; esc does not reset it; title shows 'Beans (active)' while active.

- [x] hideClosed toggle in listModel + loadBeans filter
- [x] title shows (active)
- [x] footer + help entry
- [x] tests

## Summary of Changes

- `listModel.hideClosed`, toggled with `h`; `loadBeans` excludes archive statuses (from config) and beans with a closed ancestor
- Title helper `listModel.title()`: `Beans (active) [tag: x]`
- Footer (`h hide closed`/`show closed`) and help overlay entry
- `esc`/`clearFilter` leaves the mode untouched
- Tests in `internal/tui/list_hide_closed_test.go`
