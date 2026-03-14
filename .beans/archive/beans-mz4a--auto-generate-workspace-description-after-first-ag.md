---
# beans-mz4a
title: Auto-generate workspace description after first agent message
status: completed
type: feature
priority: normal
created_at: 2026-03-13T17:57:43Z
updated_at: 2026-03-13T18:05:10Z
---

After the first agent message in a workspace, fire off a lightweight Claude CLI call (claude --print -m haiku) to generate a brief summary of what the workspace is about. Store it as a Description field on the worktree metadata and expose it via GraphQL/subscriptions so the UI can display it.

## Tasks

- [x] Add `Description` field to worktree metadata struct and persistence
- [x] Add `description` field to GraphQL Worktree type
- [x] Implement description generation via `claude --print -m haiku`
- [x] Trigger generation after first assistant response in a workspace
- [x] Wire up subscription so UI gets the update
- [x] Display description in the frontend workspace UI
- [x] Write tests


## Summary of Changes

Added auto-generated workspace descriptions. After an agent completes its first turn in a workspace, a lightweight Claude Haiku call summarizes what the workspace is doing in 3-8 words. The description is stored in worktree metadata, exposed via GraphQL, and displayed as a subtitle under the workspace name in the sidebar.

### Files changed

**Backend:**
- `internal/worktree/worktree.go` — Added `Description` field to `Worktree` struct and `worktreeMeta`, plus `UpdateDescription()` method
- `internal/agent/manager.go` — Added `OnFirstResponseFunc` callback type and `SetOnFirstResponse()` method
- `internal/agent/claude.go` — Track `isFirstSpawn` and fire callback after first `eventResult`
- `internal/agent/describe.go` — New file: `GenerateDescription()` runs `claude --print -m haiku` to summarize conversations
- `internal/commands/serve.go` — Wire up the callback to generate descriptions for new workspaces
- `internal/graph/schema.graphqls` — Added `description: String` to `Worktree` type
- `internal/graph/resolver.go` — Map `Description` field in `worktreeToModel`

**Frontend:**
- `frontend/src/lib/worktrees.svelte.ts` — Added `description` to `Worktree` interface and GraphQL fields
- `frontend/src/lib/components/Sidebar.svelte` — Display description as subtitle under workspace name

**Tests:**
- `internal/agent/describe_test.go` — Tests for first-response callback firing/not-firing
- `internal/worktree/worktree_test.go` — Test for `UpdateDescription` with notification and persistence
