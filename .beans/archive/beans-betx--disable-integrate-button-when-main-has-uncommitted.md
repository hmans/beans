---
# beans-betx
title: Disable Integrate button when main has uncommitted changes
status: completed
type: feature
priority: normal
created_at: 2026-03-13T18:21:26Z
updated_at: 2026-03-13T18:24:10Z
---

Show the Integrate action button always, but disable it when the main workspace has uncommitted changes. Add disabled/disabledReason fields to AgentAction GraphQL type.

## Summary of Changes

- Added `RepoRoot()` method to worktree Manager
- Added `disabled` and `disabledReason` fields to `AgentAction` GraphQL type
- Added `Disabled` function to `agentActionDef` struct and wired it for the Integrate action
- Extended `actionContext` with `MainRepoHasChanges` field, populated in the resolver
- Updated frontend to query and respect `disabled`/`disabledReason`, showing reason as tooltip
- Added unit test `TestIntegrateActionDisabledWhenMainHasChanges`
