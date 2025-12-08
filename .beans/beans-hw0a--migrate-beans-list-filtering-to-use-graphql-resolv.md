---
title: Migrate beans list filtering to use GraphQL resolver
status: completed
type: task
created_at: 2025-12-08T21:00:32Z
updated_at: 2025-12-08T21:00:32Z
---

## Summary

Make `beans list` execute GraphQL queries internally. The GraphQL schema becomes the source of
truth for filtering, and CLI commands become thin wrappers that build and execute GraphQL queries.

## Current State

- `cmd/list.go` has ~350 lines of filtering code (filterBeans, excludeByStatus, filterByType, etc.)
- `internal/graph/schema.resolvers.go` has filtering logic via `Beans` resolver
- GraphQL filter currently only supports type-only link filtering, not `type:id` format

## Goal

1. Extend the GraphQL schema to support `type:id` link filtering via a new `LinkFilter` input type
2. Have `beans list` construct and execute a GraphQL query internally
3. Remove duplicate filtering code from list.go

## Checklist

- [x] Add `LinkFilter` input type to GraphQL schema for `type:id` support
- [x] Update BeanFilter to use `[LinkFilter!]` for link fields
- [x] Run `mise codegen` to regenerate Go types
- [x] Update filters.go to handle LinkFilter
- [x] Update schema.resolvers.go to use new filter types
- [x] Update `beans list` to execute a GraphQL query internally
- [x] Remove duplicate filter functions from list.go
- [x] Ensure all existing tests still pass
- [x] Verify CLI behavior matches previous implementation