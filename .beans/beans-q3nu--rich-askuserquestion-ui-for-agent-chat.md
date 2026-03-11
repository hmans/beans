---
# beans-q3nu
title: Rich AskUserQuestion UI for agent chat
status: completed
type: feature
priority: normal
created_at: 2026-03-11T12:40:34Z
updated_at: 2026-03-11T16:46:51Z
order: zzzzV
---

Capture AskUserQuestion tool input (questions, options, multiSelect) from the Claude stream and render a proper interactive UI in the agent chat instead of the generic 'type your reply below' message.

## Tasks

- [x] Add Go types (AskUserQuestion, AskUserOption) to types.go
- [x] Add parseAskUserInput() to parse.go with tests
- [x] Implement deferred blocking in claude.go readOutput()
- [x] Extend GraphQL schema with question types
- [x] Run mise codegen
- [x] Update agent_helpers.go to map new fields
- [x] Update frontend TS types and subscription query
- [x] Build rich question UI in AgentChat.svelte
- [x] Run mise test
- [x] Run pnpm build (check warnings)
- [x] Run mise test:e2e

## Summary of Changes

Implemented rich AskUserQuestion UI for the agent chat:

- **Backend**: Added deferred blocking for AskUserQuestion in the stream reader so tool input JSON is fully accumulated before triggering the blocking interaction. Added `AskUserQuestion` and `AskUserOption` types, `parseAskUserInput()` parser with tests.
- **GraphQL**: Extended `PendingInteraction` with `questions` field containing structured question/option data.
- **Frontend**: Replaced the generic "type your reply below" banner with an interactive UI showing question headers, question text, and clickable option buttons. Single-select sends immediately; multi-select allows toggling options with a submit button. Falls back to the generic message if no structured data is available.
