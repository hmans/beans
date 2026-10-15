---
# beans-5fp6
title: Warn about open beans under a closed ancestor
status: completed
type: feature
priority: normal
created_at: 2026-10-01T09:40:43Z
updated_at: 2026-10-01T10:04:31Z
---

The implicit-status annotation is shown for every bean below a completed/scrapped ancestor, as a muted ` ↑<status>` suffix. For a bean that is itself completed/scrapped this repeats what its status column already says; for an open bean under a closed ancestor it is an inconsistency that deserves a warning.

## Rules
- Bean itself completed/scrapped: no ancestor lookup, no marker.
- Bean open, ancestor completed/scrapped:
  - list/tree (CLI + TUI): red `↑` directly before the status (`↑T`, wide: `↑todo`); the title suffix goes away.
  - `beans show`: direct parent closed → `parent: <id> !completed` (red). Closed ancestor further up → `parent: <id>` stays, plus `ancestor: <id> !completed` (red). The header suffix `↑completed (from …)` goes away.

## Todo
- [x] Core helper that skips the walk for resolved beans
- [x] List/tree rendering (RenderBeanRow) + tests
- [x] TUI and CLI list use the helper
- [x] `beans show` relationship lines + tests
- [x] Verify with go test and manual CLI run

## Summary of Changes

- `Core.ClosedAncestor` (pkg/beancore/links.go): returns the nearest closed ancestor of an open bean; returns nothing for completed/scrapped beans without walking the parent chain. `ImplicitStatus`, the GraphQL `implicitStatus` field and the `excludeImplicitTerminal` filter are unchanged.
- `RenderBeanRow`: red `↑` replaces the separator space left of the status, so the status stays aligned with unmarked rows; the ` ↑<status>` title suffix is gone, so the row width no longer depends on the marker. Dimmed context rows get no marker.
- TUI list and `beans list` tree use `ClosedAncestor`.
- `beans show`: header suffix removed; `parent: <id> !<status>` in red when the parent is closed, otherwise an extra `ancestor: <id> !<status>` line.
- Tests: `TestClosedAncestor`, `TestRenderBeanRow_ClosedAncestorMark`, `TestFormatRelationships_ClosedAncestor`.

Not covered: the web UI, which reads the GraphQL `implicitStatus` field directly.
