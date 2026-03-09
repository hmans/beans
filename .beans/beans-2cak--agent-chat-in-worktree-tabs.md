---
# beans-2cak
title: Agent chat in worktree tabs
status: in-progress
type: feature
priority: normal
created_at: 2026-03-08T15:58:35Z
updated_at: 2026-03-08T16:32:44Z
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

## JSONL Persistence

- [x] Create `internal/agent/store.go` with JSONL read/write
- [x] Create `.beans/conversations/` directory with `.gitignore`
- [x] Persist user messages on send
- [x] Persist assistant messages on turn completion
- [x] Persist session ID for --resume
- [x] Load conversations from disk on server restart
- [x] Add store tests (`store_test.go`)
