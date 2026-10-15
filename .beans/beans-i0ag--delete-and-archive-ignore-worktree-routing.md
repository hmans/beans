---
# beans-i0ag
title: Delete and Archive ignore worktree routing
status: todo
type: bug
priority: normal
created_at: 2026-10-13T09:29:55Z
updated_at: 2026-10-13T11:53:44Z
parent: beans-oyic
---

`Core.Delete`, `Core.Archive`, `Core.Unarchive` and `Core.LoadAndUnarchive` (`pkg/beancore/core.go`) act on `c.root/<bean.Path>`, the main `.beans/`. They ignore `worktreeLinks`. For a worktree-linked bean, `Path` is relative to the worktree's `.beans/` (`loadBeanFrom`, `pkg/beancore/worktree_watcher.go`), and usually equals the main filename.

## Cases

Inferred from the code, not reproduced.

1. **Bean exists in main and in worktree W.** `Delete` removes the main file. W's file stays. The main watcher drops the bean from runtime state; the next write in W brings it back, and the PR of W re-creates it on main.
2. **Bean exists only in W.** `Delete` fails with "no such file", because no main file exists.
3. **`Archive` of a worktree-linked bean.** Case 1: the main file moves to `archive/`, W's file stays in place, and the runtime bean has an archive path although W still edits the unarchived file. Case 2: the rename fails.

## Open question

What should deleting or archiving a bean that a worktree changes mean? Options:

- (a) act on the worktree file, so the change travels with W's PR, like `Update`
- (b) act on both main and worktree
- (c) reject the operation while the bean is worktree-linked

## Todo

- [ ] Reproduce cases 1–3 with a worktree in a test project
- [ ] Decide between (a), (b) and (c)
- [ ] Implement for `Delete`, `Archive`, `Unarchive`, `LoadAndUnarchive`
- [ ] Tests for each case
- [ ] `go test ./...`

Related: beans-xg8t covers the same routing gap in `RemoveLinksTo` and `FixBrokenLinks`.
