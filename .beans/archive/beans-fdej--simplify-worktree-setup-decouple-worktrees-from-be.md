---
# beans-fdej
title: 'Simplify worktree setup: decouple worktrees from beans'
status: completed
type: feature
priority: normal
created_at: 2026-03-11T22:30:12Z
updated_at: 2026-03-11T22:41:54Z
---

Worktrees are no longer bound to specific beans. The primary action is to create a worktree. 'Start work' on a bean creates a worktree and seeds the agent conversation with a prompt like 'start working on bean-xxxx'.

## Tasks

- [x] Unify worktree creation: remove startWork/stopWork mutations, use createWorktree/removeWorktree only
- [x] Rename Worktree.BeanID to Worktree.ID (it's just a worktree identifier now)
- [x] Update BeanDetail 'Start Work' button to create worktree + send initial agent message
- [x] Rework agent action prompts to be bean-agnostic (remove BeanID from action context)
- [x] Remove 'start-work' action (replaced by initial prompt on worktree creation)
- [x] Update sidebar to reflect new model
- [x] Update WorkspaceView to handle worktrees without bean binding
- [x] Update GraphQL schema
- [x] Update tests
- [x] Build and verify no warnings

## Summary of Changes

- Removed `startWork`/`stopWork` GraphQL mutations; all worktrees are now created via `createWorktree(name)` which generates a `wt-xxxx` ID
- Renamed `Worktree.BeanID` to `Worktree.ID` across Go structs, GraphQL schema, and frontend types
- Removed the `bean` field from the GraphQL `Worktree` type (no more structural binding)
- BeanDetail "Start Work" button now creates a worktree named after the bean title, sends `Start working on bean <id>` as the initial agent message, and navigates to the workspace
- Removed the "start-work" agent action (replaced by the initial prompt)
- Made "integrate" action prompt bean-agnostic ("mark any associated beans as completed" instead of referencing a specific bean ID)
- Simplified `actionContext` by removing `BeanID`, `BeanStatus`, and `InWorktree` fields
- WorkspaceView no longer takes a `bean` prop or shows a bean detail pane
- Sidebar workspace items use worktree name/branch instead of looking up bean titles
- Updated all Go and frontend tests
