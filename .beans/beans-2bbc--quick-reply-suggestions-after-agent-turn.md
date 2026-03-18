---
# beans-2bbc
title: Quick reply suggestions after agent turn
status: in-progress
type: feature
created_at: 2026-03-17T15:20:43Z
updated_at: 2026-03-17T15:20:43Z
---

When an agent turn ends, send the last assistant message to Haiku via claude --print --model haiku to generate 3-4 short suggested replies. Display them as clickable buttons above the message composer. Clicking one sends that text as a message.

## Tasks
- [x] Backend: Add GenerateQuickReplies function (internal/agent/quickreplies.go)
- [x] Backend: Add QuickReplies field to Session struct and snapshot
- [x] Backend: Trigger async quick reply generation on eventResult
- [x] Backend: Clear quick replies when user sends new message
- [x] Backend: Add quickReplies to GraphQL schema and model mapping
- [x] Frontend: Add quickReplies to AgentSessionFields fragment
- [x] Frontend: Run codegen
- [x] Frontend: Show quick reply chips in AgentComposer
- [x] Write tests
- [ ] Improve latency (currently ~6s due to claude CLI startup overhead)
