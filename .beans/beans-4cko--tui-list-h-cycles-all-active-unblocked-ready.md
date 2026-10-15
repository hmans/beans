---
# beans-4cko
title: 'TUI list: h cycles all → active → unblocked → ready'
status: completed
type: feature
priority: normal
created_at: 2026-10-02T07:54:53Z
updated_at: 2026-10-02T07:57:39Z
---

The 'active' view (h) still shows blocked beans. Replace the hideClosed toggle with a view mode that h cycles through:

- all: every bean
- active: hides archive statuses and beans below a closed ancestor
- unblocked: active, minus beans for which Core.IsBlocked holds
- ready: same filter as `beans list --ready` (shared helper)

- [x] Shared ready filter helper in pkg/beangraph, used by CLI --ready
- [x] TUI view mode enum, h cycles, title and footer help
- [x] Tests

## Summary of Changes

- `beangraph.AddReadyFilter` holds the `--ready` filter; `beans list --ready` uses it (output unchanged).
- TUI: `viewMode` (all, active, unblocked, ready) replaces the `hideClosed` toggle; `h` cycles through it, each mode applies the previous mode's restrictions plus its own. Border title shows the mode, footer help shows the next one, help overlay lists the cycle.
- `okf/tui-taxonomy.md`: view modes and the ⊘ blocked mark.
- Tests: `list_hide_closed_test.go` became `list_view_mode_test.go` with one case per mode and the full cycle.
