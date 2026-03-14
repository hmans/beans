---
# beans-3blw
title: Warn on destroy workspace if uncommitted/unmerged changes
status: completed
type: feature
priority: normal
created_at: 2026-03-13T19:45:31Z
updated_at: 2026-03-13T19:47:50Z
---

When clicking the destroy workspace button, warn the user if the workspace has uncommitted changes or unmerged commits. Add hasChanges and hasUnmergedCommits fields to the Worktree GraphQL type and show warnings in the confirmation modal.

## Summary of Changes

- Added `hasChanges` and `hasUnmergedCommits` boolean fields to the GraphQL `Worktree` type
- Updated `worktreeToModel` in `resolver.go` to populate these fields using existing `gitutil.HasChanges()` and `gitutil.HasUnmergedCommits()` functions
- Updated the frontend `Worktree` interface and subscription to include the new fields
- Updated the destroy workspace confirmation modal in `Sidebar.svelte` to warn users when the workspace has uncommitted changes and/or unmerged commits
