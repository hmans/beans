---
# beans-vbap
title: Image attachments in agent conversation
status: completed
type: feature
priority: normal
created_at: 2026-03-11T17:38:55Z
updated_at: 2026-03-11T21:51:10Z
order: zzzk
---

Allow users to attach images in the agent chat conversation. Images should be sent as part of the message context to the agent and displayed inline in the chat history.

## Implementation Tasks

- [x] Backend types (types.go) — ImageRef, ImageUpload, Message.Images
- [x] Backend JSONL persistence (store.go) — image storage, pruning, cleanup
- [x] Backend manager (manager.go) — SendMessage with images, pruneOrphanedAttachments
- [x] Backend Claude Code integration (claude.go) — content blocks with images
- [x] GraphQL schema + codegen
- [x] GraphQL resolvers (schema.resolvers.go, agent_helpers.go)
- [x] HTTP attachment endpoint (serve.go)
- [x] Frontend types and store (agentChat.svelte.ts)
- [x] Frontend composer (AgentComposer.svelte) — paste, drag-drop, file picker
- [x] Frontend messages (AgentMessages.svelte) — inline image display
- [x] Frontend wiring (AgentChat.svelte)
- [x] Tests

## Summary of Changes

- **Backend types** (`internal/agent/types.go`): Added `ImageRef`, `ImageUpload` structs and `Images []ImageRef` to `Message` with deep-copy support in `snapshot()`
- **JSONL persistence** (`internal/agent/store.go`): Image file storage at `.conversations/attachments/<beanId>/<uuid>.<ext>`, save/load/clear/prune methods with path traversal guards, type allowlisting, and 5MB size limit
- **Manager** (`internal/agent/manager.go`): Updated `SendMessage` to accept image uploads, added `AttachmentPath` public accessor and `pruneOrphanedAttachments` for compact cleanup
- **Claude Code integration** (`internal/agent/claude.go`): Content block array construction with base64-encoded images, compact-triggered orphan pruning
- **GraphQL schema** (`internal/graph/schema.graphqls`): Added `ImageInput`, `AgentMessageImage` types; extended `sendAgentMessage` mutation and `AgentMessage` type
- **GraphQL resolvers**: Base64 decoding of uploaded images, URL generation for image refs (`/api/attachments/<beanId>/<id>`)
- **HTTP endpoint** (`internal/commands/serve.go`): `GET /api/attachments/:beanId/:filename` with existence check, Content-Disposition, and X-Content-Type-Options headers
- **Frontend store** (`agentChat.svelte.ts`): Image types, subscription query update, mutation with images parameter
- **Frontend composer** (`AgentComposer.svelte`): Paste, drag-and-drop, and file picker for image attachment; thumbnail previews with remove buttons
- **Frontend messages** (`AgentMessages.svelte`): Inline image display in user messages
- **Tests**: 7 new Go unit tests for image storage, validation, round-trip, clear, prune, and path traversal; 1 new Playwright e2e test for inline image display
