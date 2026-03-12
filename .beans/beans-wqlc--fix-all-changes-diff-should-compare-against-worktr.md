---
# beans-wqlc
title: 'Fix: All Changes diff should compare against worktree base branch, not remote upstream'
status: completed
type: bug
priority: normal
created_at: 2026-03-12T14:59:25Z
updated_at: 2026-03-12T15:02:43Z
---

The All Changes tab in the changes pane for worktree spaces compares against origin/main (the remote default branch) instead of the configured worktree base branch. MergeBase() uses DefaultRemoteBranch() which resolves to origin/main, but it should use the base_ref from .beans.yml config (defaulting to 'main').

## Summary of Changes

- Changed `MergeBase()` to accept a `baseRef` parameter instead of hardcoding `origin/main` via `DefaultRemoteBranch()`
- Updated `AllChangesVsUpstream()` and `AllFileDiff()` to accept and pass the base ref
- Updated GraphQL resolvers to pass `WorktreeMgr.BaseRef()` (from `.beans.yml` config, defaults to `main`)
- Also fixed `HasUnmergedCommits` call in agent action resolver to use configured base ref
- Added `BaseRef()` getter to `worktree.Manager`
- Added test for fallback behavior when baseRef is empty
