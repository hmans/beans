---
# beans-uk55
title: '@ mention file/directory attachment in agent chat'
status: completed
type: feature
priority: normal
created_at: 2026-03-21T09:33:57Z
updated_at: 2026-03-21T09:46:22Z
---

Allow users to type @ in the agent composer to autocomplete and attach files/directories from the codebase as context. File contents are injected into the prompt sent to Claude Code.

## Summary of Changes

Implemented @-mention file/directory attachment in the agent chat composer:

### Backend
- Added `listFiles` GraphQL query that uses `git ls-files` to list tracked files, filtered by prefix, with directory deduplication (one level of depth)
- Extended `sendAgentMessage` mutation to accept `attachments: [FileAttachmentInput!]` — attached paths are prepended as context hints in the user message
- Added `FileEntry` type and `FileAttachmentInput` input to the GraphQL schema
- Added unit tests for `ListFiles` resolver

### Frontend
- Added @-detection in the composer textarea — typing `@` (preceded by whitespace or start-of-text) opens an autocomplete dropdown
- Dropdown queries the backend via `ListFiles` with debounced input (100ms)
- Keyboard navigation: ArrowUp/Down to navigate, Enter/Tab to select, Escape to close
- Selecting a file adds it as a pill below the textarea and removes the @query text
- Selecting a directory replaces the query to drill deeper (keeps dropdown open)
- Pending attachments shown as removable pills with file/folder icons
- Attachments are passed through to the GraphQL mutation via the store
