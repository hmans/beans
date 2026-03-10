---
# beans-cbpd
title: Permission approval UI for agent chat (act mode)
status: completed
type: feature
priority: normal
created_at: 2026-03-10T10:37:28Z
updated_at: 2026-03-10T10:46:36Z
---

Implement PermissionRequest hook-based permission approval in the web UI, enabling a real 'act' mode where Claude Code requests are mediated through HTTP hooks and the user approves/denies in the browser.

## Summary of Changes

### Backend (`internal/agent/`)
- **types.go**: Added `InteractionPermission` type, `PermissionRequest` struct (with `ToolName`, `ToolInput`, response channel), `PermissionResponse` struct, and `SetResponseCh` method
- **manager.go**: Changed `NewManager` to accept `ManagerConfig` (with `BeansDir` + `ServerPort`); added `SetPermissionRequest()` and `ResolvePermission()` methods
- **claude.go**: Extended `buildClaudeArgs` to inject `--settings` with PermissionRequest HTTP hook config in act mode (not plan, not yolo)

### HTTP Endpoint (`internal/commands/serve.go`)
- Added `POST /api/permissions/:beanId` endpoint that receives Claude Code's PermissionRequest hook calls
- Blocks until user responds via GraphQL mutation or 110s timeout (auto-deny)
- Passes server port to `ManagerConfig` so hook URLs are correct

### GraphQL (`internal/graph/`)
- **schema.graphqls**: Added `PERMISSION_REQUEST` to `InteractionType` enum; added `toolName` and `toolInput` fields to `PendingInteraction`; added `resolvePermission` mutation
- **schema.resolvers.go**: Implemented `ResolvePermission` resolver
- **agent_helpers.go**: Maps `InteractionPermission` to `PERMISSION_REQUEST` and populates tool name/input fields

### Frontend
- **agentChat.svelte.ts**: Added `PERMISSION_REQUEST` to `InteractionType`; added `toolName`/`toolInput` to `PendingInteraction` interface; added `RESOLVE_PERMISSION` mutation and `resolvePermission()` method
- **AgentChat.svelte**: Added permission request UI banner with tool name, formatted input preview, and Allow/Always Allow/Deny buttons; `formatToolInput()` shows context-appropriate summaries per tool type

### Tests
- Added unit tests for `SetPermissionRequest`, `ResolvePermission` (success, no session, no pending), and `buildClaudeArgs` act mode behavior
- Updated all existing tests for `ManagerConfig` signature change
