---
# beans-a37d
title: Sort workspaces by last agent activity
status: completed
type: feature
priority: normal
created_at: 2026-03-14T15:05:50Z
updated_at: 2026-03-14T15:09:29Z
---

Sort workspaces in the sidebar by when their agent last completed a turn, so most recently active workspaces appear at the top. Uses a LastActiveAt timestamp persisted in worktree metadata.

## Summary of Changes

- Added `LastActiveAt` timestamp to `worktreeMeta` (persisted to `.meta.json` files)
- Added `LastActiveAt` field to the `Worktree` struct
- `Create()` now sets `LastActiveAt` at creation time so new worktrees sort to the top
- Added `TouchLastActive(id)` method that updates the timestamp and notifies subscribers
- `List()` now sorts worktrees by `LastActiveAt` descending (most recently active first, no-activity worktrees at the end)
- Added `OnTurnCompleteFunc` callback to agent manager, fired on `eventResult` (agent turn completion)
- Wired callback in `serve.go` to call `TouchLastActive` when a workspace agent finishes a turn
- No frontend changes needed — sidebar renders in the order the backend provides
- Added 3 new tests: `TestTouchLastActive`, `TestListSortsByLastActiveAt`, `TestCreateSetsLastActiveAt`
