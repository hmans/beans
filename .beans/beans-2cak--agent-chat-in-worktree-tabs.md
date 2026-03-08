---
# beans-2cak
title: Agent chat in worktree tabs
status: in-progress
type: feature
priority: normal
created_at: 2026-03-08T15:58:35Z
updated_at: 2026-03-08T16:06:46Z
---

Implement agent chat within worktree tabs in the web UI. Spawn and manage Claude Code CLI sessions from the Go backend, stream output via GraphQL subscriptions, and provide a chat composer UI.

## Tasks

- [x] Create `internal/agent/` package (types, manager, claude runner, parser)
- [x] Add agent session types to GraphQL schema and run codegen
- [x] Implement GraphQL resolvers (query, mutation, subscription)
- [x] Wire agent manager into serve command
- [x] Create frontend agent chat store (`agentChat.svelte.ts`)
- [x] Create AgentChat Svelte component
- [x] Update worktree page to use AgentChat
- [x] Write backend unit tests (parsing, manager lifecycle)
- [x] Verify full stack works end-to-end (build passes)
