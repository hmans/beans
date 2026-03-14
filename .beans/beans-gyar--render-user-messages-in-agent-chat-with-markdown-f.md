---
# beans-gyar
title: Render user messages in agent chat with markdown formatting
status: completed
type: bug
priority: normal
created_at: 2026-03-14T18:28:25Z
updated_at: 2026-03-14T18:29:03Z
---

User messages in the agent chat are rendered as plain text (whitespace-pre-wrap) while assistant messages get full markdown rendering via renderMarkdown(). User messages should also be markdown-formatted for consistency.

## Summary of Changes

Modified `AgentMessages.svelte` to render user messages with markdown formatting (same as assistant messages):

- Extended the markdown rendering `$effect` to process both `USER` and `ASSISTANT` messages
- Updated `getRenderedContent()` to return rendered HTML for user messages
- Updated the user message template to use `{@html}` with `agent-prose prose` classes when rendered content is available, falling back to plain text while rendering
