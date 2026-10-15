---
# beans-26we
title: Checks that relate a bean to other beans run outside the write lock
status: todo
type: bug
priority: high
created_at: 2026-03-14T15:06:18Z
updated_at: 2026-10-13T09:30:03Z
parent: beans-5txp
---

Some checks that relate a bean to other beans run in `pkg/beangraph` before `Core` takes its write lock `Core.mu`. A concurrent writer can change the other beans between the check and the write, so the stored state can violate the check. Two checks already run again under the lock in `Core.Update`: the ETag compare-and-swap (beans-o1if) and the cycle check `addedLinkCycleLocked` (beans-6we9). This bean covers the remaining ones, and `Core.Create`, which runs no check at all.

## Gaps

1. **Link target exists.** `ValidateAndAddBlocking` and `ValidateAndAddBlockedBy` (`pkg/beangraph/resolver.go`) and `CreateBean` (`pkg/beangraph/mutations.go`) call `Core.Get` on each target. Example: request A adds `X blocking Y`, request B deletes Y. A's check passes, B deletes Y, A's update stores a broken link.
2. **`DeleteBean` is not atomic.** `DeleteBean` (`pkg/beangraph/mutations.go`) calls `Core.RemoveLinksTo(id)` and then `Core.Delete(id)`, each in its own lock section. An update between the two calls can add a link to the bean that is being deleted, and that link stays broken.
3. **`Create` misses cycles that its new links close.** `CreateBean` rejects only the trivial cycle: the new bean both blocks and is blocked by the same bean. Example: `T blocking U` is stored; `beans create N --blocking T --blocked-by U` stores the cycle N → T → U → N. `Core.Create` has no backstop like `addedLinkCycleLocked`. A new bean cannot race into a cycle, unless a broken link already points at its ID, so this is a validation gap rather than a race.
4. **Parent type.** `ValidateParent` (`pkg/beancore/links.go`) runs only when a request sets a parent, and only before the lock. A concurrent type change of the parent can make the stored link invalid. Even without concurrency, no check runs when a type change makes an existing parent link invalid: the child's type, or the parent's type, changes.

Gaps 1 and 2 leave broken links, which readers tolerate and `beans check --fix` removes. Gaps 3 and 4 leave a state that nothing repairs.

## Design

`Core.Create`, `Core.Update` and a new atomic delete run every check that relates beans under the write lock, against the stored beans:

- target exists, for every link the bean adds
- cycle check, with `addedLinkCycleLocked(&bean.Bean{}, b)` for a create
- parent type, for an added parent link

The resolver validators stay. They give an early error message that names the request. The checks under the lock are the authority.

`Core` gets one method that removes the incoming links of a bean and deletes it, in one lock section. `DeleteBean` calls it.

## Todo

- [ ] `Core.Create` and `Core.Update`: target-exists and parent-type checks for added links, under the lock
- [ ] `Core.Create`: cycle check with `addedLinkCycleLocked`
- [ ] Atomic delete in `Core`; `DeleteBean` uses it
- [ ] Decide gap 4 beyond the parent-set case: reject a type change that invalidates a parent link (own or children's), or leave it and file a follow-up bean
- [ ] Tests: create with `--blocking T --blocked-by U` and `T blocking U` stored is rejected, same for parent; update adding a link to a bean deleted concurrently is rejected; update concurrent with delete leaves no broken link
- [ ] `go test -race ./...`

Related: beans-qypq makes the stored beans that these checks read current across processes. beans-unjf was merged into this bean as gap 3.
