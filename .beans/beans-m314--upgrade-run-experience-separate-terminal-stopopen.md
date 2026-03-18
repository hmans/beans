---
# beans-m314
title: 'Upgrade Run experience: separate terminal, Stop/Open buttons'
status: completed
type: feature
priority: normal
created_at: 2026-03-17T19:36:50Z
updated_at: 2026-03-18T09:37:04Z
---

- [x] Run command should use a separate terminal session from the main shell
- [x] Run button becomes Stop when app is running
- [x] Open button appears next to Stop, launching http://localhost:$BEANS_WORKSPACE_PORT/
- [x] Terminal pane has tabs (Shell / Run) when run session exists
- [x] Backend: startRun, stopRun, isRunning, workspacePort GraphQL operations
- [x] Frontend: new GraphQL operations and codegen
- [x] E2E tests for run experience

## Summary of Changes

Upgraded the Run experience in workspace toolbars:

**Backend:**
- Added `CreateWithCommand()` to terminal Manager — runs a specific command via `shell -l -c` instead of an interactive login shell, session exits when command finishes
- Added `startRun`, `stopRun` mutations and `isRunning`, `workspacePort` queries to GraphQL schema
- Run sessions use `{workspaceId}__run` convention for terminal session IDs
- Updated port allocator env injection to strip `__run` suffix so run sessions share the workspace port

**Frontend:**
- Run button becomes Stop (red, with square icon) when a run session is active
- Open button appears next to Stop, linking to `http://localhost:{port}/`
- Terminal pane shows Shell/Run tabs when a run session exists
- Run tab auto-activates on start, reverts to Shell when run ends
- Initial run state is checked on workspace mount via `isRunning` query
- TerminalPane gains `onSessionEnd` callback and `hideToolbar` prop

**Tests:**
- 4 unit tests for `CreateWithCommand` (output, exit behavior, session replacement, env injection)
- 5 e2e tests covering Run/Stop/Open button visibility, state transitions, and port display
