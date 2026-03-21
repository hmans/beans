---
# beans-x0zh
title: Fix subscription resolver dropping notifications under backpressure
status: completed
type: bug
priority: normal
created_at: 2026-03-21T08:16:46Z
updated_at: 2026-03-21T08:21:04Z
---

The AgentSessionChanged and ActiveAgentStatuses subscription resolvers block on sending to the out channel. While blocked, new notifications on the buffer-1 ch channel are silently dropped. This can cause the frontend to miss entire RUNNING status transitions, resulting in the agent working indicator disappearing while the agent is still active. Fix by using a latest-value pattern that refreshes pending state when new notifications arrive during a blocked send.

## Summary of Changes

Changed the AgentSessionChanged and ActiveAgentStatuses subscription resolvers from a simple block-on-send pattern to a latest-value pattern. When the resolver is blocked sending a payload to the WebSocket transport and a new notification arrives, it now refreshes the pending state with the latest session data instead of dropping the notification. This ensures the frontend always receives the most recent session status, preventing the 'Agent is working' spinner from disappearing while the agent is actively working.

- Modified AgentSessionChanged resolver in schema.resolvers.go
- Modified ActiveAgentStatuses resolver in schema.resolvers.go
- Added test TestAgentSessionSubscription_DeliversLatestState
