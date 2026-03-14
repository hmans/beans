---
# beans-7hoc
title: Optimistic user message in agent chat
status: completed
type: feature
priority: normal
created_at: 2026-03-13T15:21:04Z
updated_at: 2026-03-13T16:31:32Z
---

Add optimistic UI update so user messages appear instantly in agent chat instead of waiting for the subscription round-trip

## Summary of Changes

Added optimistic UI update to AgentChatStore.sendMessage() so user messages appear instantly in the chat instead of waiting for the GraphQL subscription round-trip.
