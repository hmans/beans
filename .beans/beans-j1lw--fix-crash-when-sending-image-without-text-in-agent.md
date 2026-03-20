---
# beans-j1lw
title: Fix crash when sending image without text in agent chat
status: completed
type: bug
priority: normal
created_at: 2026-03-20T14:11:55Z
updated_at: 2026-03-20T14:13:41Z
---

Sending an image with no text creates an empty text content block ({"type": "text", "text": ""}), which the Anthropic API rejects with 'messages: text content blocks must be non-empty'. This can destroy an entire conversation.

## Summary of Changes

Fixed a bug where sending an image with no text in agent chat would create an empty text content block (`{"type": "text", "text": ""}`), causing the Anthropic API to reject the message. The fix conditionally includes the text block only when the message is non-empty.

**Files changed:**
- `internal/agent/claude.go` — Skip empty text block in `sendToProcess` when message is empty
- `internal/agent/claude_test.go` — Added two tests: image-only (no empty text block) and image-with-text (both blocks present)
