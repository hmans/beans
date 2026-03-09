---
# beans-jy9h
title: Add input validation to GraphQL resolvers
status: todo
type: task
priority: high
created_at: 2026-03-09T17:01:44Z
updated_at: 2026-03-09T20:28:54Z
order: zy
parent: beans-oe8n
---

GraphQL mutations accept arbitrary strings without validation. Issues: (1) status, type, priority fields are not checked against configured enum values — invalid values get persisted. (2) title has no max length — could be megabytes. (3) body has no size limit. (4) createBean accepts a user-supplied prefix that goes directly to bean.NewID() without validation — a prefix like '../../etc' could cause path traversal in filename generation. (5) tags are validated in bean.go (ValidateTag regex) but this validation is never called in GraphQL resolvers. Fix: add a validation layer in the resolvers (or a shared validation function) that checks: prefix matches ^[a-z][a-z0-9]*$, title max 1000 chars, body max 1MB, status/type/priority are valid enum values from config, all tags pass ValidateTag(). Return clear GraphQL errors for invalid input.
