---
# beans-9rix
title: Add Content Security Policy headers
status: completed
type: task
priority: normal
created_at: 2026-03-09T17:01:54Z
updated_at: 2026-03-12T08:21:33Z
order: zzs
parent: beans-oe8n
---

The server sends no Content Security Policy headers, so even if XSS is found, there are no restrictions on what injected scripts can do (exfiltrate data, load external resources, etc.). Fix: add CSP headers in the serve.go middleware. Recommended starting policy: default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline' (needed for Svelte/Tailwind); img-src 'self' data:; connect-src 'self' ws://localhost:* wss://localhost:*; font-src 'self'. The 'unsafe-inline' for styles is unfortunate but required by most CSS-in-JS/utility frameworks. Consider making CSP configurable or adding a --csp flag. Test that the SPA still works correctly with the policy applied.

## Summary of Changes

Added Content-Security-Policy middleware to the HTTP server in `serve.go`:

- `default-src 'self'` — restricts all resource loading to same-origin by default
- `script-src 'self'` — only same-origin scripts
- `style-src 'self' 'unsafe-inline'` — allows inline styles (required by Tailwind/Svelte)
- `img-src 'self' data:` — allows data URIs for images (markdown rendering)
- `font-src 'self'` — same-origin fonts only
- `connect-src 'self' ws: wss:` — allows same-origin HTTP and all WebSocket connections (needed for GraphQL subscriptions and terminal)

Added `csp_test.go` with tests verifying all directives are present and the header is set on every response.
