---
# beans-9s9h
title: Replace optimistic workspace removal with destroying state indicator
status: completed
type: bug
priority: normal
created_at: 2026-03-18T10:20:34Z
updated_at: 2026-03-18T10:22:23Z
---

When closing a workspace, the optimistic removal causes it to flash (disappear then reappear from subscription, then disappear again). Replace with a 'destroying' visual state (low opacity) and let the backend subscription handle the actual removal.

## Summary of Changes

Replaced optimistic workspace removal with a 'destroying' state indicator:

- **worktrees.svelte.ts**: Added a `destroying` Set to track workspaces being destroyed. Instead of eagerly filtering them out of the list, they stay visible while the backend processes the removal. The subscription handler clears the destroying flag once the worktree disappears from the backend's list.
- **Sidebar.svelte**: Workspaces being destroyed render at 30% opacity with `pointer-events-none`, preventing interaction while providing clear visual feedback.
- **WorkspaceView.svelte**: The destroy button is disabled and shows 'Destroying...' tooltip during destruction.
