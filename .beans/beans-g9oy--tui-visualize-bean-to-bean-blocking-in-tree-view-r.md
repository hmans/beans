---
# beans-g9oy
title: 'TUI: visualize bean-to-bean blocking in tree view + ''r'' to hide blocked'
status: completed
type: feature
priority: normal
created_at: 2026-05-16T14:00:39Z
updated_at: 2026-05-16T14:06:05Z
---

Make bean-to-bean blocking dependencies visible in the TUI tree view, plus a keybind to hide blocked beans.

Closes hmans/beans#91 ("Ability to see a tree view" / "How can we see what blocks what?").

Three behavior changes:

1. **Topo-reorder siblings within their existing sort bucket** so that, when sibling A is an active blocker of sibling B, A appears before B. Tiebreaker on top of the status/priority/type sort — preserves the existing order for unrelated siblings.

2. **Annotation on blocked beans.** Beans with active (non-archive) blockers get a `⊘` glyph rendered after the title, similar to the existing `↑<implicit status>` annotation.

3. **`r` keybind toggles "ready only"** — when on, hide beans that are explicitly blocked. Uses the existing GraphQL `isExplicitlyBlocked: false` filter so blocked beans don't appear in the matched set, but their parent epics still show as ancestors-for-context.

Aligns with hmans's stance in #78 (bean-to-bean blocking is the dependency mechanism; no need for a new sibling-scoped `dependsOn` field). Zero schema change.

## Todo

- [x] Pipe an `activeBlockerIDs` map into `BuildTree`
- [x] Stable Kahn-topo pass `reorderByActiveBlockers` applied after `sortFn` per sibling group (only honors edges within the group)
- [x] Added `Blocked bool` to `TreeNode` and `FlatItem`
- [x] Added `Blocked bool` to `BeanRowConfig`; renderer appends a muted ` ⊘` after title (skipped when row is dimmed, same as `ImplicitStatus`)
- [x] `loadBeans` and CLI tree path both compute the set via `Core.FindActiveBlockers`
- [x] Added `readyOnly bool`; `buildFilter()` sets `IsExplicitlyBlocked: &false`; composes with existing tag filter
- [x] Added `r` toggle + reload
- [x] `buildTitle()` composes `[ready]` with `[tag: x]`
- [x] Help overlay + dynamic footer label (`ready only` / `show blocked`)
- [x] 5 topo cases in `TestBuildTreeReorderByActiveBlockers` (sibling reorder, no-edge passthrough, non-sibling edge ignored, already-correct chain, full reversal); 3 `TestListModelBuildFilterReadyOnly`; 4 `TestListModelBuildTitleReadyOnly`
- [x] Full `go test ./...` green (with web dist stubbed for embed)
- [x] Commit + push branch to chrisjkuch/beans
- [x] PR opened against upstream/main referencing #91

## Out of scope

- New `dependsOn` field (rejected per #78 maintainer direction)
- Auto-create blockedBy on manual reorder (footgun)
- Cross-parent blockedBy visualization beyond what already exists (parent-context dim)
- Dimming blocked beans (the glyph alone is sufficient; overloading existing `Dimmed` would conflate with ancestor-for-context)


## Summary of Changes

- `BuildTree` gains an `activeBlockers map[string][]string` parameter (nil = old behavior). After the existing sibling sort, `reorderByActiveBlockers` runs a Kahn topo-sort on the within-group blocker edges, preserving incoming order for unrelated pairs (stable).
- `TreeNode.Blocked` and `FlatItem.Blocked` carry a boolean: true when the bean has at least one entry in `activeBlockers` (matches `Core.IsExplicitlyBlocked`).
- `BeanRowConfig.Blocked` adds a muted ` ⊘` glyph after the title, skipped when the row is dimmed (same gate the existing `↑<status>` annotation uses).
- TUI `listModel`: new `readyOnly` flag, `r` keybind that toggles + reloads, `[ready]` title indicator composing with `[tag: x]`. Uses the existing GraphQL `isExplicitlyBlocked: false` filter — blocked beans drop out of the matched set while their parent epics remain as ancestor-context.
- Same `activeBlockers` computation added to `internal/commands/list.go` so the `beans list` tree view picks up the same ordering and annotation.
- No schema changes. All edges flow through the existing `blockedBy`/`blocking` model — aligns with hmans's direction in #78.
