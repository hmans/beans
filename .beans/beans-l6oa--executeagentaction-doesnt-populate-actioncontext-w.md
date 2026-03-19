---
# beans-l6oa
title: ExecuteAgentAction doesn't populate actionContext with PR/git state
status: completed
type: bug
priority: normal
created_at: 2026-03-19T17:43:46Z
updated_at: 2026-03-19T17:45:08Z
---

The ExecuteAgentAction mutation resolver only sets WorktreeID, WorkDir, MainRepoPath, and ForgeCLI on the actionContext. It never populates PullRequest, HasChanges, HasNewCommits, HasUnpushedCommits, etc. This means prPrompt() always sees ctx.PullRequest == nil and returns the 'Create a pull request' prompt regardless of actual state.

## Summary of Changes

Fixed the `ExecuteAgentAction` mutation resolver to populate the full `actionContext` (git state and PR state) before calling `action.PromptFunc()`. Previously it only set `WorktreeID`, `WorkDir`, `MainRepoPath`, and `ForgeCLI`, causing `prPrompt()` to always see `ctx.PullRequest == nil` and return the "Create a pull request" prompt regardless of actual state.

Changes in `internal/graph/schema.resolvers.go`:
- Added `gitutil.HasChanges()`, `gitutil.HasUnmergedCommits()`, `gitutil.HasUnpushedCommits()` calls
- Added `gitutil.CurrentBranch()` + `Forge.FindPR()` to populate PR state
