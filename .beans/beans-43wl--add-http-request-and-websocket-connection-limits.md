---
# beans-43wl
title: Add HTTP request and WebSocket connection limits
status: todo
type: task
priority: low
created_at: 2026-03-09T17:01:59Z
updated_at: 2026-03-09T20:28:54Z
order: zzy
parent: beans-oe8n
---

The HTTP server has no explicit MaxHeaderBytes or body size limits, and WebSocket connections are unlimited. A single client could open thousands of WebSocket subscriptions and exhaust file descriptors, or send massive HTTP payloads. Fix: (1) Set http.Server.MaxHeaderBytes to 1MB. (2) Add request body size middleware (e.g., gin's MaxMultipartMemory or a custom LimitReader middleware) — 10MB should be generous. (3) Add a WebSocket connection counter and reject new connections above a threshold (e.g., 100 concurrent). (4) Consider adding per-IP connection tracking if needed later. This is lower priority since it's primarily a DoS concern for localhost usage.
