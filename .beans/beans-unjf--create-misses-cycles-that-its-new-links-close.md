---
# beans-unjf
title: Create misses cycles that its new links close
status: scrapped
type: bug
priority: low
created_at: 2026-10-12T10:03:37Z
updated_at: 2026-10-13T09:30:03Z
---

`CreateBean` (`pkg/beangraph/mutations.go:13`) rejects only the trivial cycle on a new bean: it blocks and is blocked by the same bean. It misses longer cycles. Example: `T blocking U` is stored; `beans create N --blocking T --blocked-by U` stores the cycle N → T → U → N.

The relationship validators on updates (`ValidateAndAddBlocking`, `ValidateAndAddBlockedBy` in `pkg/beangraph/resolver.go`) check each link alone, with `DetectCycleWith`, and add it only after its check. Two links added in one request therefore are not checked against each other. `Core.Update` catches this since beans-6we9 (`addedLinkCycleLocked` sees all added links at once), with the error from beancore. A create has no such backstop, because `Core.Create` runs no cycle check.

A new bean cannot race with a concurrent writer into a cycle (no stored bean links to an ID that does not exist yet, unless a broken link already points at it), so this is a validation gap, not a concurrency bug.

## Todo

- [ ] Decide where the check lives: `CreateBean` calls `DetectCycleWith` with the pending bean for each link, or `Core.Create` runs `addedLinkCycleLocked(&bean.Bean{}, b)` under its write lock (one check for both paths, also covers a broken link that already points at the new ID)
- [ ] Test: `create --blocking T --blocked-by U` with `T blocking U` stored is rejected; same for parent
- [ ] Optional: make the update validators see all links of one request, so the resolver error appears before `Core.Update` and names the request
- [ ] `go test ./...`

## Reasons for Scrapping

Merged into beans-26we as gap 3. beans-26we collects every check that relates a bean to other beans and moves it under the write lock; the create-time cycle check is one of them.
