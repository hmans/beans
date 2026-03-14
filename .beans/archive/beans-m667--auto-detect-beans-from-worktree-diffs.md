---
# beans-m667
title: Auto-detect beans from worktree diffs
status: completed
type: feature
priority: normal
created_at: 2026-03-13T14:53:07Z
updated_at: 2026-03-13T15:07:09Z
---

Expose beans field on Worktree GraphQL type by analyzing git diff vs base branch for .beans/*.md changes

## Tasks

- [x] Add BeanIDs field + DetectBeanIDs() to worktree Manager
- [x] Populate BeanIDs in List()
- [x] Update GraphQL schema with beans field on Worktree
- [x] Regenerate GraphQL code and implement resolver
- [x] Update frontend worktree store and Sidebar
- [x] Write tests

## Summary of Changes

- Added `BeanIDs []string` field to `Worktree` struct and `DetectBeanIDs()` method to worktree Manager
- `DetectBeanIDs()` uses `gitutil.AllChangesVsUpstream()` to find `.beans/*.md` file changes, filtering out subdirs
- `List()` now populates `BeanIDs` for each worktree
- Added `beans: [Bean!]!` field to `Worktree` GraphQL type, resolved via `beancore.Core.Get()`
- Updated `worktreeToModel()` to accept `*beancore.Core` and resolve bean IDs to full Bean objects
- Updated frontend `Worktree` interface and subscription to include `beans { id title status type }`
- Sidebar renders detected beans nested under each worktree, clicking navigates to planning view with bean selected
- Added `selectBeanById()` method to UIState
- Added 3 tests: committed changes, no changes, and untracked changes
