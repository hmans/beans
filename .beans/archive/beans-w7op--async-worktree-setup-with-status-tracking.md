---
# beans-w7op
title: Async worktree setup with status tracking
status: completed
type: feature
priority: normal
created_at: 2026-03-13T19:32:13Z
updated_at: 2026-03-13T19:39:06Z
---

Run worktree setup command asynchronously instead of blocking creation. Track setup status and notify the user via agent chat when setup completes or fails.

## Summary of Changes

### Backend
- Added `RoleInfo` message role to agent types — info messages are visible in the chat UI but never sent to Claude
- Added `AddInfoMessage(beanID, content)` to the agent Manager for posting system info messages
- Added `SetupStatus` enum (`running`, `done`, `failed`) and `SetupError` fields to the Worktree struct
- Made the setup command in `Manager.Create()` run **asynchronously** in a goroutine instead of blocking
- Added `SetOnSetupDone` callback to the worktree Manager
- Wired the callback in serve.go to post an info message to the workspace's agent chat on setup completion/failure

### GraphQL
- Added `WorktreeSetupStatus` enum and `setupStatus`/`setupError` fields to the Worktree type
- Added `INFO` to the `AgentMessageRole` enum

### Frontend
- Added `INFO` role to AgentMessage type and renders info messages with a distinct styled container (border, italic, muted)
- Added `setupStatus`/`setupError` to the Worktree interface and GraphQL fragment
- Shows "Setting up..." animated indicator in the sidebar for workspaces with running setup

### Tests
- Updated `TestCreateRunsSetupCommand` to handle async setup via callback and verify setup status tracking
