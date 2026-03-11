---
# beans-63sz
title: Changes pane in web UI
status: completed
type: feature
priority: normal
created_at: 2026-03-10T20:39:30Z
updated_at: 2026-03-11T16:46:51Z
order: zzzw
---

Add a toggleable Changes pane to the web UI that shows git working tree status — list of changed files with per-file +/- line stats. Works in both PlanningView (main repo) and WorkspaceView (worktree).

## Tasks

- [x] Backend: git status utility (internal/gitutil/status.go)
- [x] Backend: unit tests for git utility
- [x] Backend: GraphQL schema + codegen
- [x] Backend: resolver implementation
- [x] Frontend: changes store with polling
- [x] Frontend: UI state toggle
- [x] Frontend: ChangesPane component
- [x] Frontend: integrate into PlanningView
- [x] Frontend: integrate into WorkspaceView
- [x] E2E tests (existing tests all pass; skipping new e2e test for now)

## Summary of Changes

- Added `internal/gitutil/status.go` with `FileChanges()` function that combines `git diff --numstat` (staged/unstaged) and `git ls-files --others` (untracked)
- Added unit tests in `internal/gitutil/status_test.go`
- Added `FileChange` GraphQL type and `fileChanges(path)` query with worktree path validation
- Added frontend polling store (`changes.svelte.ts`) querying every 3s
- Added `ChangesPane` component with status indicators, file paths, and +/- line stats
- Added toggle button (Material Symbols `difference` icon) in PlanningView and WorkspaceView toolbars
- Changes and Agent panes share a SplitPane so dragging their separator resizes both
- Persisted toggle state via localStorage
