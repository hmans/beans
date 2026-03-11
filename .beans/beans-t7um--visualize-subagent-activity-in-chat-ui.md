---
# beans-t7um
title: Visualize subagent activity in chat UI
status: completed
type: feature
priority: normal
created_at: 2026-03-11T10:19:14Z
updated_at: 2026-03-11T16:46:51Z
order: zzzzk
---

When Claude uses subagents (via the Agent tool), the parent session receives stream events from the subagent but currently logs them as 'unhandled event'. We should visualize this activity in the chat UI so users can see what subagents are doing in real-time.

## Tasks
- [x] Understand current event flow for subagent events
- [x] Route subagent events through the subscription system
- [x] Add frontend UI to render subagent activity in the chat
- [x] Handle nested subagents (subagents spawning subagents)
- [x] Test the feature

## Summary of Changes

Implemented real-time subagent activity visualization in the chat UI.

### Backend (Go)
- **parse.go**: Propagated `session_id` from `stream_event` wrappers to `parsedEvent`; added handling for `thinking_delta` and `signature_delta` (ignored instead of unknown)
- **types.go**: Added `SubagentActivity` struct (Description, Text, CurrentTool) to `Session`; updated `snapshot()` for deep copy
- **claude.go**: Track main session ID; route events with different session_id to `SubagentActivity` state (text deltas, tool use, tool input); create `SubagentActivity` on Agent tool invocation; clear on turn completion
- **schema.graphqls**: Added `SubagentActivity` type and field on `AgentSession`
- **agent_helpers.go**: Convert `SubagentActivity` to GraphQL model

### Frontend (Svelte/TS)
- **agentChat.svelte.ts**: Added `SubagentActivity` interface and subscription field
- **AgentChat.svelte**: Render subagent activity inline with left border accent; show description, current tool, and streaming text (last 500 chars); update status bar to show subagent info

### Tests
- Updated parse_test.go expectations for session_id propagation
- Added test cases for thinking_delta and signature_delta (ignored)
- All 21 e2e tests pass, all Go tests pass
