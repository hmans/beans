---
# beans-jmqu
title: Worktree status info in changes panel
status: completed
type: feature
priority: normal
created_at: 2026-03-14T10:12:57Z
updated_at: 2026-03-14T10:17:46Z
---

Add status information above the All Changes/Unstaged tab pill: commits behind main (with rebase button), merge conflict indicator, and change summary (added/deleted/modified counts).


## Summary of Changes

### Backend
- Added `CommitsBehind(dir, baseBranch)` and `HasConflicts(dir, baseBranch)` to `internal/gitutil/status.go`
  - `CommitsBehind` counts commits on base branch not reachable from HEAD via `git rev-list --count`
  - `HasConflicts` uses `git merge-tree --write-tree` to check for conflicts without modifying the working tree
- Added `BranchStatus` GraphQL type with `commitsBehind` and `hasConflicts` fields
- Added `branchStatus(path)` query to the GraphQL schema
- Populated `commitsBehind` and `hasConflicts` on the `Worktree` type as well
- Added 4 unit tests for the new git utility functions

### Frontend
- Extended `ChangesStore` to poll `branchStatus` alongside file changes
- Added status bar above the tab switcher in `ChangesPane` showing:
  - Commits behind count with warning/danger coloring (danger when conflicts expected)
  - "Rebase" button that sends a message to the agent asking it to rebase
  - Change summary (added/modified/deleted/renamed file counts) with color-coded labels
- Status bar only appears for worktree workspaces (not the main workspace) and only when there's info to show
