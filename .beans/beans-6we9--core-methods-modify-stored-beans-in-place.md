---
# beans-6we9
title: Core methods modify stored beans in place
status: completed
type: bug
priority: normal
created_at: 2026-10-08T15:40:29Z
updated_at: 2026-10-12T13:59:12Z
---

Five `Core` methods modify beans stored in `c.beans` in place, under the write lock. `Core.Get` returns the stored pointer and releases its read lock, so a reader that holds that pointer reads the same fields without a lock. The two accesses race.

| Method | Writes |
|---|---|
| `Core.Archive` | `Path` |
| `Core.Unarchive` | `Path` |
| `Core.LoadAndUnarchive` | `Path` |
| `Core.RemoveLinksTo` (`pkg/beancore/links.go`) | `Parent`, `Blocking`, `BlockedBy` |
| `Core.FixBrokenLinks` (`pkg/beancore/links.go`) | `Parent`, `Blocking`, `BlockedBy` |

Readers without a lock: `CoreResolver.cloneForUpdate` (`pkg/beangraph/resolver.go`), the GraphQL field resolvers, `output.NewBean`.

Example: `deleteBean(Y)` and `updateBean(X)` run concurrently, and X blocks Y. `RemoveLinksTo` reassigns `X.Blocking` while `Bean.Clone` runs `slices.Clone(X.Blocking)`. The race detector flags it, and a torn slice header can read out of bounds.

beans-o1if made `Core.Update` a compare-and-swap with the invariant "no code modifies a bean stored in `c.beans`". These five methods are the remaining exceptions.

## Cross-bean cycle race

The relationship validators run `DetectCycleWith` before `Core.Update` takes the write lock. The compare-and-swap protects one bean only. Request A adds `X blocking Y` on X, request B adds `Y blocking X` on Y, concurrently: each check sees only its own pending edge, both swaps succeed, and a blocking cycle is stored.

## Todo

- [x] The five methods clone each bean they change, modify the clone, and store the clone in `c.beans`
- [x] Test under `-race`: `RemoveLinksTo` concurrent with `Clone` of a bean it changes
- [x] Decide the cycle race: `Core.Update` re-checks the edges the bean adds, under the write lock, with the lock-free `findPathToTarget`
- [x] `Core.Update` rejects an added edge that closes a cycle; test with two crossing updates
- [x] `SaveDirty`/`SaveBean` store a clone when `saveToDisk` assigns `Path`
- [x] `go test -race ./...` (the Go part of `just test`; codegen and e2e not run, no schema or frontend change)

Related: beans-qypq covers concurrent writers across processes. The `Core.mu` lock that this bean relies on orders goroutines inside one process only.

## Summary of Changes

- `Archive`, `Unarchive`, `LoadAndUnarchive` (`pkg/beancore/core.go`) store a clone with the new `Path`; `Archive` sends the clone to subscribers, `LoadAndUnarchive` returns it.
- `RemoveLinksTo` (`pkg/beancore/links.go`) clones only the beans that link to the target; `FixBrokenLinks` deep-copies only the beans it changes. Both store the clone after `saveToDisk` succeeds, so a failed save leaves the store unchanged.
- `SaveDirty` and `SaveBean` save a clone, because `saveToDisk` assigns `Path` to a bean without one, and store it.
- `Core.Update` re-checks, under the write lock, every parent, blocking and blocked_by link the bean adds over its stored version, with the lock-free `findPathToTarget`, and returns `CycleError` if one closes a cycle. Each added blocking or blocked_by link is checked in both representations (`X blocking Y` and `Y blocked_by X`), and `CycleError.LinkType` names the added link. Two concurrent updates on different beans can no longer store a cycle together. Links the bean already had are not checked, so a cycle already on disk does not block unrelated updates. The check sees all links an update adds at once, so it also rejects an update that adds `--blocking` and `--blocked-by` closing a cycle together, which the relationship validators let through.
- Tests: `TestWritersLeaveHeldBeansUnchanged` (all seven writers, fails on the old code); `TestRemoveLinksToConcurrentWithClone` (race detector reports a DATA RACE on the old code); `TestUpdateRejectsAddedLinkThatClosesCycle` (crossing blocking, blocked_by and parent writers; a cycle that closes only in the inverse representation; both links in one update); `TestUpdateAllowsStoredCycleWithoutAddedLinks`.

Not changed: `RemoveLinksTo` and `FixBrokenLinks` still write every changed bean to the main `.beans/`, also a dirty or worktree-linked one, and send no subscriber events.

Follow-ups: beans-9sxr (`findCycles` returns corrupted cycle paths), beans-unjf (`Create` misses cycles that its new links close).

Follow-up: beans-xg8t (routing and events of `RemoveLinksTo`/`FixBrokenLinks`).
