---
# beans-o1if
title: beans update rejects the etag that beans show reports
status: completed
type: bug
priority: normal
created_at: 2026-10-08T13:51:47Z
updated_at: 2026-10-08T14:45:07Z
---

`beans update --if-match` rejects the etag that `beans show` reports, for every bean whose file differs from its rendered form. The cause is that `updateBean` modifies the stored bean before the etag check, and `Core.Update` works around that by hashing a different input than `Bean.ETag()`.

## Reproduction

A bean created without `-p` has no `priority:` line in its file:

```
$ beans create "probe" -t task          # -> beans-xxxx
$ beans update beans-xxxx --if-match "$(beans show beans-xxxx --etag-only)" -s todo
Error: etag mismatch: provided a685f981b655e800, current is 79b12faf276b6c8d
```

Observed on beans-4x0o (2026-10-07) and beans-q7ye (2026-10-06).

## Root cause

Two generators compute the etag of one bean:

| Path | Generator |
|---|---|
| reads: `Bean.ETag()` (`pkg/bean/bean.go:301`), used by `show`, `--json`, GraphQL | FNV-1a of `Render()` |
| `If-Match` check in `Core.Update` (`pkg/beancore/core.go:563-574`), clean beans | FNV-1a of the file bytes |

The loader fills defaults (`pkg/beancore/core.go:205-210`, e.g. `priority: normal`), so the rendered form and the file differ.

The file-byte hash exists because of the order in `UpdateBean` (`pkg/beangraph/mutations.go:109` onward):

1. `Core.Get` returns the pointer stored in `c.beans[id]` (`core.go:315`), not a copy.
2. `UpdateBean` writes the input fields into that object, without a lock.
3. `Core.Update` takes the write lock and only then compares the etag. `storedBean` is the modified object, so its `ETag()` can never match.

The check ran before the modification originally (PR #59, `validateETag` in `pkg/beangraph/resolver.go:40`). PR #72 moved it under the write lock to fix a race (two requests both passing the check, the second overwriting the first), which put it after the modification. `validateETag` has had no callers since; step 2 of the design reuses it.

Further effects of the same order, inferred from the code, not reproduced:

- The `If-Match` check on a dirty bean always fails: `core.go:576-579` hashes the already-modified stored object.
- A rejected update still changes `c.beans[id]`. In `beans serve`, queries and subscriptions show the rejected change until the bean is reloaded.
- Step 2 races with readers that hold `RLock`.

## Design

Optimistic concurrency with a compare-and-swap as the only transaction:

1. **Read and clone.** Deep-copy `c.beans[id]`, including `Tags`, `Blocking`, `BlockedBy`, `Properties`. `base := clone.ETag()`.
2. **Early reject.** If `ifMatch` is given and differs from `base`, return `ETagMismatchError`. Nothing has changed.
3. **Modify the clone.** No lock is held, so validations such as `DetectCycle` can call `Core`.
4. **Compare-and-swap.** `Core.Update(clone, base)` under the write lock: write only if `c.beans[id].ETag() == base`; save to disk (or worktree, or mark dirty) and store the clone in `c.beans[id]`. Otherwise return `ETagMismatchError`.

Step 4 compares against `base` also when the client sent no `ifMatch`, so concurrent updates without an etag no longer overwrite each other.

Invariant: no code modifies a bean stored in `c.beans`. Every writer uses clone, modify, swap.

Generator: both sides of step 4 come from the in-memory bean, so `Bean.ETag()` (rendered hash) serves reads and the check, and the file-byte branch goes away. It is weak with respect to the file: a hand edit that renders identically (whitespace, field order, a missing default) does not change the etag, and a hand edit stays invisible until the file watcher reloads the bean. The alternative closes both gaps: `beancore` records the hash of the bytes it last read or wrote, and a dirty bean uses its rendered hash.

## Todo

- [x] Decide the generator: rendered hash everywhere (`Bean.ETag()`); the file-byte branch goes away
- [x] Add a deep `Clone()` to `bean.Bean` with a test that mutating the clone leaves the original unchanged
- [x] `Core.Get` returns a clone, or its callers are audited to treat the result as read-only
- [x] Every caller of `Core.Update` clones, rejects early, and modifies the clone: `UpdateBean` and the five relationship mutations (`pkg/beangraph/mutations.go:236,306,340,356,389,405`), and the TUI (`internal/tui/tui.go:658`)
- [x] `Core.Update`: compare-and-swap against `base` under the write lock; remove the file-byte branch (or replace it by the recorded hash)
- [x] Step 2 (early reject) uses `validateETag` (`pkg/beangraph/resolver.go:40`), as `.claude/rules/backend.md` asks of resolvers
- [x] Tests: `show` etag accepted by `update` for a file without `priority:`; dirty bean with `If-Match`; rejected update leaves the stored bean unchanged; two concurrent updates with the same `base`, exactly one succeeds
- [x] `mise test` (Go part: `go test ./...`; `mise` is not installed in the implementing environment)

## Summary of Changes

- `Bean.Clone()` (`pkg/bean/bean.go`) deep-copies a bean: `Tags`, `Blocking`, `BlockedBy`, `Properties`, and the timestamp pointers.
- `Core.Update(b, base, opts...)` (`pkg/beancore/core.go`) is the compare-and-swap. `base` is the ETag of the stored bean the caller cloned. Update writes only if the stored bean still has that ETag, and returns `ETagMismatchError` with `Concurrent: true` otherwise, also when the client sent no `ifMatch`. `Concurrent` selects the message "bean changed concurrently", so the error does not claim an etag the client never sent. The file-byte branch is gone; `Bean.ETag()` serves reads and the check.
- `CoreResolver.cloneForUpdate` (`pkg/beangraph/resolver.go`) runs steps 1 and 2: get, clone, `base := clone.ETag()`, and `validateETag`. `validateETag` also holds the `require_if_match` check and returns the `beancore` error types. The duplicate `beangraph.ETag*Error` types are removed.
- `UpdateBean` and the five relationship mutations modify the clone. The relationship validators change only the bean they are given, never the target bean.
- `Core.DetectCycleWith(pending, ...)` (`pkg/beancore/links.go`) traverses the store with the clone in place of its stored version. Without it, a request that adds `blocking: X` and `blocked_by: X` together passes the cycle check, because the stored bean no longer carries the first edge.
- The TUI editor path (`internal/tui/tui.go`) updates a clone of the reloaded bean.
- `Bean.ApplyDefaults()` (`pkg/bean/bean.go`) fills `type`, `priority`, `tags` and `blocking`. `loadBean`, `loadBeanFrom`, `Create` and `Update` all call it, so a stored bean renders the same before and after a reload and keeps its ETag.

`Core.Get` still returns the stored pointer. Its callers are audited: none outside `beancore` modifies the result.

Open: `Core.Archive`, `Unarchive`, `LoadAndUnarchive`, `RemoveLinksTo` and `FixBrokenLinks` still modify stored beans in place under the write lock. A reader that holds a pointer from `Get` reads the same fields without a lock, so these writers race with it: `cloneForUpdate`, the GraphQL field resolvers and `output.NewBean` all read that way. beans-6we9 makes them clone, modify and swap.

## Review (opencode, 6a80b60)

- `Create`/`Update` stored a bean without the defaults `loadBean` fills, so its ETag changed on the next reload (`create --json` etag rejected by `update --if-match`; watcher reload failing a CAS without `ifMatch`): applied; `Bean.ApplyDefaults` in all four paths, `TestStoredETagSurvivesReload` fails without it.
- Summary claimed in-place writers do not race with readers: applied; sentence corrected above. The code fix is beans-6we9.
- Rewrite lost the empty-`ifMatch` case under `require_if_match`: applied; case added to `TestRequireIfMatchConfig`.
- Test case "file identical to its rendered form" was not identical (missing `# probe-1`): applied; content fixed, and the test asserts the claim.
- CAS mismatch reported an etag the client never sent: applied; `ETagMismatchError.Concurrent` selects "bean changed concurrently", asserted in both directions.
- Cycle check across two beans is check-then-act (two requests adding `X blocking Y` and `Y blocking X`): rejected for this bean; the race predates the fix, since `DetectCycle` never ran under the write lock. Recorded in beans-6we9.
