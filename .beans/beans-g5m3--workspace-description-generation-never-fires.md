---
# beans-g5m3
title: Workspace description generation never fires
status: completed
type: bug
priority: normal
created_at: 2026-03-14T09:26:36Z
updated_at: 2026-03-14T09:27:41Z
---

The onFirstUserMessage callback in agent.Manager.SendMessage checks `!ok` (whether the session was just created), but AddInfoMessage pre-creates the session when workspace setup completes. This means the callback never fires for workspaces that go through setup, which is all of them.


## Summary of Changes

- Changed `SendMessage` in `internal/agent/manager.go` to detect first user message by counting user-role messages instead of checking session existence (`!ok`)
- Added `countUserMessages` helper function
- Added test `TestSendMessageCallbackFiresWithInfoOnlySession` covering the exact broken scenario (session pre-created by `AddInfoMessage` from workspace setup)
