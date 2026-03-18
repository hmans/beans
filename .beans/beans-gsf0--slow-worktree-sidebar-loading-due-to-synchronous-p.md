---
# beans-gsf0
title: Slow worktree sidebar loading due to synchronous PR lookups
status: completed
type: bug
priority: normal
created_at: 2026-03-17T18:04:59Z
updated_at: 2026-03-17T18:07:26Z
---

The WorktreesChanged subscription resolver calls populatePR() synchronously for every worktree before emitting. Each populatePR runs 2-3 gh CLI subprocesses hitting the GitHub API. This blocks the initial emission, causing the sidebar to take seconds to appear.

## Tasks
- [x] Refactor WorktreesChanged resolver to emit without PR data first
- [x] Add async PR population that re-emits after PR data is fetched
- [x] Parallelize PR lookups across worktrees
- [x] Do the same for change notifications (the for loop at line 1353)
- [x] Write tests (all existing tests pass)

## Summary of Changes

Refactored the `WorktreesChanged` subscription resolver in `internal/graph/schema.resolvers.go`:

- **Emit worktree list immediately without PR data** — the sidebar now appears instantly with worktree names, branches, and bean associations
- **Fetch PR data asynchronously** — PR info (status, checks, mergeable) is populated in the background and re-emitted when ready
- **Parallelize PR lookups** — all worktrees' PR data is fetched concurrently using goroutines instead of sequentially

This eliminates the blocking `gh` CLI calls (2-3 per worktree, each hitting GitHub API) from the critical path of the initial subscription emission.
