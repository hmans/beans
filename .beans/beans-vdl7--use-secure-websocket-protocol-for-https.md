---
# beans-vdl7
title: Use secure WebSocket protocol for HTTPS
status: todo
type: task
priority: low
created_at: 2026-03-09T17:02:08Z
updated_at: 2026-03-09T20:28:54Z
order: zzz
parent: beans-oe8n
---

graphqlClient.ts line 9 hardcodes ws:// for WebSocket connections. If beans is ever served behind HTTPS (via reverse proxy), this will fail or expose data in plaintext. Fix: detect the page protocol and use wss:// when on HTTPS. Something like: const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'. Low priority since beans currently runs localhost HTTP only, but trivial to fix and future-proofs the code.
