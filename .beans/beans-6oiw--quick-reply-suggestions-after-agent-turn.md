---
# beans-6oiw
title: Quick reply suggestions after agent turn
status: scrapped
type: feature
priority: normal
created_at: 2026-03-17T15:20:34Z
updated_at: 2026-03-17T15:25:04Z
---

When an agent turn ends, send the last assistant message to Haiku via claude --print --model haiku to generate 3-4 short suggested replies. Display them as clickable buttons above the message composer. Clicking one sends that text as a message.

## Tasks
- [ ] Backend: Add GenerateQuickReplies function (internal/agent/quickreplies.go)
- [ ] Backend: Add QuickReplies field to Session struct and snapshot
- [ ] Backend: Trigger async quick reply generation on eventResult
- [ ] Backend: Clear quick replies when user sends new message
- [ ] Backend: Add quickReplies to GraphQL schema and model mapping
- [ ] Frontend: Add quickReplies to AgentSessionFields fragment
- [ ] Frontend: Run codegen
- [ ] Frontend: Show quick reply chips in AgentComposer
- [ ] Write tests
