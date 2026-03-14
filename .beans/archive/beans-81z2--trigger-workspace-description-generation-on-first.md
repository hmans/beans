---
# beans-81z2
title: Trigger workspace description generation on first user message instead of first agent turn
status: completed
type: bug
priority: normal
created_at: 2026-03-13T19:46:02Z
updated_at: 2026-03-13T19:58:57Z
---

Description generation currently fires after the agent's first turn completes. It should fire immediately after the first user message is sent, using the user message as the primary input.

## Summary of Changes

- Changed `OnFirstResponseFunc` → `OnFirstUserMessageFunc` in `manager.go` — callback now receives `(beanID, message string)` instead of `(beanID, []Message)`
- Moved callback trigger from `readOutput` (after first agent turn) to `SendMessage` (when creating a new session)
- Simplified `describe.go`: `buildDescribePrompt` and `GenerateDescription` now accept a single `string` instead of `[]Message`
- Updated prompt text from "conversation" framing to "first user message" framing
- Removed unused `isFirstSpawn` parameter from `readOutput` and `spawnAndRun`
- Updated all tests to match new API
