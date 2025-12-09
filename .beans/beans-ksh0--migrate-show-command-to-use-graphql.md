---
title: Migrate 'show' command to use GraphQL
status: completed
type: task
priority: normal
created_at: 2025-12-09T12:04:04Z
updated_at: 2025-12-09T12:16:00Z
links:
    - parent: beans-7ao1
---

## Summary

Migrate the `beans show` command to use the GraphQL resolver instead of directly calling `core.Get()`.

## Current Implementation

In `cmd/show.go` line 28:
```go
b, err := core.Get(args[0])
```

## Target Implementation

Use the GraphQL resolver pattern (like `list` command):
```go
resolver := &graph.Resolver{Core: core}
b, err := resolver.Query().Bean(context.Background(), args[0])
```

## Checklist

- [ ] Import the graph package
- [ ] Create resolver instance
- [ ] Replace `core.Get()` with `resolver.Query().Bean()`
- [ ] Update any bean field access to match GraphQL model
- [ ] Run tests to verify functionality