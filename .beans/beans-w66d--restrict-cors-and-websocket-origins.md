---
# beans-w66d
title: Restrict CORS and WebSocket origins
status: todo
type: task
priority: high
created_at: 2026-03-09T17:01:38Z
updated_at: 2026-03-09T20:28:54Z
order: zz
parent: beans-oe8n
---

The server currently sets Access-Control-Allow-Origin: * (serve.go line 65) and the WebSocket upgrader's CheckOrigin always returns true (line 97). This allows any website to make authenticated requests to the beans API and establish WebSocket subscriptions. Fix: replace the wildcard CORS origin with a configurable allowlist defaulting to localhost origins (http://localhost:*, http://127.0.0.1:*). The WebSocket CheckOrigin should validate the Origin header against the same allowlist. Consider adding a --cors-origin flag to beans serve for custom origins. Also review Access-Control-Allow-Headers and Access-Control-Allow-Methods to be specific rather than wildcard.
