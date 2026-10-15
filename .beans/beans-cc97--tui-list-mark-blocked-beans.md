---
# beans-cc97
title: 'TUI list: mark blocked beans'
status: completed
type: feature
priority: normal
created_at: 2026-10-02T07:36:59Z
updated_at: 2026-10-02T07:39:00Z
---

The TUI list does not show that a bean is blocked (directly or via its parent chain). Mark such rows with a warning-colored ⊘ in the separator slot before the status column, analogous to the red ↑ for a closed ancestor; ↑ takes precedence.

- [x] Pre-compute blocked state in loadBeans
- [x] Pass it via beansLoadedMsg/beanItem to BeanRowConfig.Blocked
- [x] Render ⊘ in RenderBeanRow
- [x] Tests

## Summary of Changes

- `ui.BeanRowConfig.Blocked`: `RenderBeanRow` puts a warning-colored ⊘ in the separator slot before the status; the closed-ancestor ↑ takes precedence, dimmed context rows stay unmarked.
- TUI `loadBeans` computes the set of open beans for which `Core.IsBlocked` holds (direct `blocked_by`, incoming `blocking` links, blocked ancestors) and passes it to the list items via `beansLoadedMsg`. `BuildTree`/`FlatItem` stay unchanged, so the CLI tree output is unaffected.
- Tests: `TestRenderBeanRow_BlockedMark`, `TestListLoadBeansBlocked`.
