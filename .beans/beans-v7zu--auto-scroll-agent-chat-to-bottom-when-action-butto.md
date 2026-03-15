---
# beans-v7zu
title: Auto-scroll agent chat to bottom when action buttons are clicked
status: completed
type: bug
priority: normal
created_at: 2026-03-15T17:12:07Z
updated_at: 2026-03-15T17:13:41Z
---

When a user clicks action buttons (Integrate, Rebase, etc.) and the agent chat is scrolled up, the chat doesn't scroll to the bottom to show the new message.

## Summary of Changes

Added scroll-to-bottom triggering when action buttons (Integrate, Rebase, etc.) or the Rebase button in ChangesPane are clicked:

- `AgentMessages.svelte`: Added `scrollToBottomTrigger` prop with effect that forces `stuckToBottom = true` and scrolls when triggered
- `AgentChat.svelte`: Passes through external trigger + triggers scroll on composer send
- `AgentActions.svelte`: Added `onExecute` callback fired when an action is executed
- `ChangesPane.svelte`: Added `onAgentMessage` callback fired when rebase message is sent
- `WorkspaceView.svelte`: Wires everything together with a shared `scrollToBottomTrigger` counter
