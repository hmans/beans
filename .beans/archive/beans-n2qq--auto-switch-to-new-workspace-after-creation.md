---
# beans-n2qq
title: Auto-switch to new workspace after creation
status: completed
type: bug
priority: normal
created_at: 2026-03-12T14:28:42Z
updated_at: 2026-03-12T14:29:23Z
---

When clicking the + button to create a new workspace, the view should automatically switch to that workspace. Currently there's a race condition: the navigateTo fires but the worktree subscription hasn't updated the store yet, so the fallback effect in +layout.svelte redirects back to planning.

## Summary of Changes

Fixed race condition in worktree creation by eagerly adding the new worktree to local state in `WorktreeStore.createWorktree()` before the subscription delivers the update. This prevents the layout guard from redirecting back to planning view.
