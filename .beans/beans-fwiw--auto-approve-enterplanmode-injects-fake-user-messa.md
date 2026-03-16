---
# beans-fwiw
title: Auto-approve EnterPlanMode injects fake user message
status: completed
type: bug
priority: normal
created_at: 2026-03-16T18:21:05Z
updated_at: 2026-03-16T18:22:11Z
---

When the agent requests EnterPlanMode, autoApproveModeSwitch calls SendMessage with 'yes, proceed', which creates a fake user message in the conversation history. The fix: just toggle plan mode and respawn directly without creating any message.

## Summary of Changes

Fixed autoApproveModeSwitch in internal/agent/claude.go to directly call spawnAndRun instead of SendMessage. This prevents a fake 'yes, proceed' user message from appearing in the conversation history when entering plan mode. The mode switch is now transparent to the user — no message is persisted, the session just toggles PlanMode and respawns.
