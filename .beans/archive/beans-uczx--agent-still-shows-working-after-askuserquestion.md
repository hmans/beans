---
# beans-uczx
title: Agent still shows 'working' after AskUserQuestion
status: completed
type: bug
priority: normal
created_at: 2026-03-13T20:20:01Z
updated_at: 2026-03-13T20:21:57Z
---

When the agent invokes AskUserQuestion, readOutput continues processing events after handleBlockingTool sets status to IDLE. The ensureRunning() helper then flips status back to RUNNING, causing the frontend to show 'Agent is working...' alongside the pending interaction UI.


## Summary of Changes

After `handleBlockingTool` sets session status to IDLE and signals the process, the `readOutput` loop continues processing remaining stdout events. The `ensureRunning()` helper would then flip the status back to RUNNING, causing the frontend to show "Agent is working..." alongside the AskUserQuestion UI.

**Fix:** Added a `blocked` flag in `readOutput` that is set after `handleBlockingTool` is called. `ensureRunning()` checks this flag and returns early, preventing the status from flipping back to RUNNING while the process is winding down.

- `internal/agent/claude.go`: Added `blocked` flag, set it after all `handleBlockingTool` calls within the scanner loop, check it in `ensureRunning()`
- `internal/agent/claude_test.go`: Added `TestReadOutputAskUserQuestionStaysIdle` test
