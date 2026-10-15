---
# beans-9sxr
title: beans check reports corrupted cycle paths
status: todo
type: bug
priority: low
created_at: 2026-10-12T10:03:37Z
updated_at: 2026-10-12T10:03:37Z
---

`beans check` can report wrong cycle paths. `Core.findCycles` (`pkg/beancore/links.go:283`) builds a cycle path that shares its backing array with the slice that the DFS keeps appending to.

## Cause

- `cyclePath := append(path[cycleStart:], id)` (`links.go:301`) aliases `path`.
- `dfs(target, append(path, id))` (`links.go:342`) passes a slice to sibling calls whose appends can overwrite the elements of an already recorded `cyclePath`.

A randomized check over 300 seeds found 32 invalid cycles, for example `[b1 b3 b6 b5 b3]`: the path does not close on its start, or it contains edges that do not exist. `findPathToTarget` uses the same `append(path, t)` pattern but returns on the first hit, and the same check found no invalid path there.

## Todo

- [ ] Copy the path before each append in `findCycles` (`slices.Clone`, or `append(slices.Clone(...), id)`)
- [ ] Test: every cycle that `CheckAllLinks` reports is a closed walk over existing links, on a graph with branching (the randomized case, made deterministic with a fixed seed or a fixed graph that reproduces it)
- [ ] `go test ./pkg/beancore/`
