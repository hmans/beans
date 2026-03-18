---
# beans-sa92
title: Remove automatic reordering of workspaces in sidebar
status: completed
type: task
priority: normal
created_at: 2026-03-18T10:10:19Z
updated_at: 2026-03-18T10:11:01Z
---

Keep workspaces in creation order (oldest first) instead of reordering by last activity

## Summary of Changes

Removed the LastActiveAt-based sort in `worktree.Manager.List()`, so workspaces now stay in git's native creation order (oldest first) instead of reordering by last activity. Updated the corresponding test.
