---
# beans-9rix
title: Add Content Security Policy headers
status: todo
type: task
priority: normal
created_at: 2026-03-09T17:01:54Z
updated_at: 2026-03-09T20:28:54Z
order: zzs
parent: beans-oe8n
---

The server sends no Content Security Policy headers, so even if XSS is found, there are no restrictions on what injected scripts can do (exfiltrate data, load external resources, etc.). Fix: add CSP headers in the serve.go middleware. Recommended starting policy: default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline' (needed for Svelte/Tailwind); img-src 'self' data:; connect-src 'self' ws://localhost:* wss://localhost:*; font-src 'self'. The 'unsafe-inline' for styles is unfortunate but required by most CSS-in-JS/utility frameworks. Consider making CSP configurable or adding a --csp flag. Test that the SPA still works correctly with the policy applied.
