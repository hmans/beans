---
title: Migrate 'roadmap' command to use GraphQL
status: completed
type: task
priority: normal
created_at: 2025-12-09T12:04:04Z
updated_at: 2025-12-09T12:16:25Z
links:
    - parent: beans-7ao1
---

## Summary

Migrate the `beans roadmap` command to use the GraphQL resolver instead of directly calling `core.All()`.

## Current Implementation

In `cmd/roadmap.go` line 75:
```go
allBeans, err := core.All()
```

Also accesses bean links directly at lines 113 and 196.

## Target Implementation

Use the GraphQL resolver:
```go
resolver := &graph.Resolver{Core: core}
allBeans, err := resolver.Query().Beans(context.Background(), nil)
```

## Checklist

- [ ] Import the graph package
- [ ] Create resolver instance
- [ ] Replace `core.All()` with `resolver.Query().Beans()`
- [ ] Update bean field access to use GraphQL model types
- [ ] Verify link access works with new model
- [ ] Run tests to verify functionality