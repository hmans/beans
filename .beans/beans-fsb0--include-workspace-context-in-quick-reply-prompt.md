---
# beans-fsb0
title: Include workspace context in quick reply prompt
status: completed
type: task
priority: normal
created_at: 2026-03-18T09:40:19Z
updated_at: 2026-03-18T09:43:07Z
parent: beans-2bbc
---

The Haiku prompt for generating quick reply suggestions should include workspace status (PR state, uncommitted changes, unpushed commits, etc.) so the suggestions are more contextually relevant. E.g. if there's an open PR with passing checks, suggest 'merge the PR' instead of generic replies.

## Summary of Changes

- Added `QuickReplyContextFunc` callback type to the agent Manager (follows existing provider pattern)
- Updated `GenerateQuickReplies` to accept optional workspace context string
- Enhanced the Haiku prompt to consider workspace status when suggesting replies
- Wired up context provider in `serve.go` that gathers: branch name, uncommitted changes, unmerged/unpushed commits, conflicts, and PR status (number, state, CI checks, review approval)

### Files modified
- `internal/agent/manager.go` — new `QuickReplyContextFunc` type and `SetQuickReplyContext` method
- `internal/agent/quickreplies.go` — updated prompt and `GenerateQuickReplies` signature
- `internal/commands/serve.go` — workspace context provider registration
