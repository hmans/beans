---
# beans-jwqn
title: 'Upgrade changes view: all changes vs upstream + tab switcher'
status: completed
type: feature
priority: normal
created_at: 2026-03-12T14:41:06Z
updated_at: 2026-03-12T14:46:38Z
---

Show all changes compared to the upstream branch (committed + staged + unstaged) and add a tab switcher between 'Unstaged Changes' and 'All Changes'.

## Summary of Changes

### Backend (Go)
- Added `MergeBase()`, `AllChangesVsUpstream()`, and `AllFileDiff()` to `internal/gitutil/status.go`
- Added `allFileChanges` and `allFileDiff` GraphQL queries to schema
- Added resolvers with worktree path validation and path traversal protection
- Added 7 new tests covering merge-base detection, committed/unstaged/untracked combinations, and diff generation

### Frontend (Svelte)
- Updated `changes.svelte.ts` to fetch both regular and all-changes data in parallel
- Redesigned `ChangesPane.svelte` with a tab switcher (All Changes / Unstaged)
- All Changes tab shows combined committed+staged+unstaged+untracked changes vs upstream
- Unstaged tab shows the previous staged/unstaged view (disabled when no working tree changes)
- Renamed pane title from "Status" to "Changes" across all views
- Clicking a file in All Changes mode fetches diff vs merge-base; in Unstaged mode uses the existing staged/unstaged diff
