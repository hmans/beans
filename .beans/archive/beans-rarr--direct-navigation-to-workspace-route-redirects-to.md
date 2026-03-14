---
# beans-rarr
title: Direct navigation to workspace route redirects to planning
status: completed
type: bug
priority: normal
created_at: 2026-03-12T15:37:17Z
updated_at: 2026-03-12T15:38:05Z
---

Race condition: the layout guard effect checks worktreeStore.hasWorktree() before the subscription has delivered data, so it always redirects to planning on cold page loads.

## Summary of Changes

- Added `initialized` flag to `WorktreeStore` that becomes `true` after the first subscription result
- Updated the layout guard `$effect` to wait for `worktreeStore.initialized` before deciding to redirect
- This prevents the race condition where a cold page load to a workspace route would redirect to planning before the worktree subscription delivers data
