---
# beans-36gu
title: Remove git worktree prune from worktree deletion
status: completed
type: bug
priority: high
created_at: 2026-03-12T08:17:34Z
updated_at: 2026-03-12T08:18:58Z
---

When deleting a worktree through the sidebar, `git worktree prune` can remove more than just the targeted worktree. The `Remove()` method in `internal/worktree/worktree.go` has two fallback paths that run `git worktree prune`, which is a global operation that removes ALL stale worktree entries — not just the one being deleted. This means if any other worktrees had missing directories, they'd get silently cleaned up too.

## Fix
- Remove all `git worktree prune` calls from `Remove()`
- When the worktree ID isn't found in the active list, return a clear error instead of pruning
- When `git worktree remove` fails with 'is not a working tree', return an error instead of pruning
- If we need prune functionality, it should be a separate explicit operation, never triggered implicitly by a single deletion

## Summary of Changes

- Removed both `git worktree prune` calls from `Remove()` in `internal/worktree/worktree.go`
- When worktree ID is not found in the active list, `Remove()` now returns a clear error instead of pruning
- When `git worktree remove` fails (e.g. stale directory), the error is propagated instead of falling back to prune
- Updated `TestRemoveStaleWorktree` to expect an error for stale worktrees
- Added `TestRemoveNonexistent` to verify error on unknown worktree IDs
