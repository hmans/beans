---
# beans-zow9
title: Move bean, beancore, and config packages to public API
status: completed
type: task
priority: normal
created_at: 2026-03-10T19:11:48Z
updated_at: 2026-03-10T19:13:18Z
---

Move internal/bean, internal/beancore, and internal/config out of internal/ to make them importable by external Go applications. internal/search stays internal as it's only used privately by beancore.

## Summary of Changes

Moved `bean`, `beancore`, and `config` packages from `internal/` to top-level, making them importable by external Go applications:

- `github.com/hmans/beans/bean` — domain types, parsing, sorting
- `github.com/hmans/beans/beancore` — thread-safe store, CRUD, watcher, events
- `github.com/hmans/beans/config` — project configuration

Updated all import paths across the codebase, gqlgen config (`gqlgen.yml`), and regenerated GraphQL code. `internal/search` stays internal as it's only consumed privately by `beancore`.
