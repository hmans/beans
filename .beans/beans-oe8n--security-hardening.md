---
# beans-oe8n
title: Security hardening
status: todo
type: epic
priority: normal
created_at: 2026-03-09T17:01:23Z
updated_at: 2026-03-09T20:28:54Z
order: zzk
---

Harden beans against common web security vulnerabilities identified in security review. Priority-ordered by impact and ease of fix. Covers XSS via unsanitized markdown, missing CORS/origin restrictions, no input validation on GraphQL mutations, path traversal risks, missing CSP headers, and no request/connection limits. Out of scope for now: authentication (separate effort for multi-user), rate limiting (low priority for local use), HTTPS/TLS (reverse proxy concern).
