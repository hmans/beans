---
# beans-xg8t
title: RemoveLinksTo and FixBrokenLinks bypass Update's worktree routing
status: todo
type: bug
priority: low
created_at: 2026-10-12T13:59:03Z
updated_at: 2026-10-13T11:49:22Z
parent: beans-oyic
---

`Core.RemoveLinksTo` and `Core.FixBrokenLinks` (`pkg/beancore/links.go`) call `saveToDisk` for every bean they change. They ignore the routing that `Core.Update` applies: `Update` writes a worktree-linked bean into its worktree's `.beans/` and keeps a runtime-only (dirty) bean out of the main `.beans/`.

## Effect

In practice a bean is dirty only when the worktree watcher linked it to a worktree (`pkg/beancore/worktree_watcher.go`). No caller uses `WithPersist(false)`.

Example: bean X is linked to worktree W and blocks bean Y. Deleting Y in `beans serve` runs `RemoveLinksTo(Y)`, which writes W's version of X into the main `.beans/`. The main watcher then loads that file and clears X's dirty flag. `worktreeLinks[X]` still points at W. Main now carries the worktree's unmerged state of X, and the Beans UI no longer shows X as dirty.

## Events

Except for `Archive`, no write method of `Core` calls `fanOut`, `Update` included. Subscribers receive a persisted write as the echo of the file watcher, about 100 ms later. A write routed into a worktree echoes through the worktree watcher. So `RemoveLinksTo` and `FixBrokenLinks` need no event of their own once they route like `Update`.

## Todo

- [ ] Route each changed bean like `Core.Update` does (worktree link, dirty flag, persist); preferably through one write helper that `Update` uses too
- [ ] Tests: a worktree-linked bean that links to a deleted bean is written to its worktree, and the main file stays unchanged; same for `FixBrokenLinks`
- [ ] `go test ./...`

Related: beans-qypq adds a file lock to every write path; one shared write helper leaves fewer paths to lock. `Delete`, `Archive` and `Unarchive` have the same routing gap for worktree-linked beans; that is beans-i0ag.

## Reachability

The bug is latent. `RemoveLinksTo` routes wrongly only in a `Core` whose `worktreeLinks` is set, i.e. in `beans serve`. The CLI never watches worktrees.

- Beans UI: `DeleteBean` exists in `operations.graphql`, but no component calls it.
- `beans delete` and `beans check --fix` run in a CLI `Core`.
- A hand-written `deleteBean` request to `beans serve` reaches the bug.

The bug becomes live once the UI calls `deleteBean`, or once `beans serve` calls `FixBrokenLinks`.
