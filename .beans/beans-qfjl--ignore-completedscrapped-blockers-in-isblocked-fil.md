---
# beans-qfjl
title: Ignore completed/scrapped blockers in isBlocked filter
status: completed
type: feature
priority: normal
created_at: 2026-01-29T09:13:26Z
updated_at: 2026-01-29T09:17:06Z
---

Currently beans are blocked by just existence of blocking beans, regardless of their status. This feature makes the isBlocked filter ignore completed/scrapped blocking beans.

## Tasks
- [x] Add IsBlocked and FindActiveBlockers methods to beancore/links.go
- [x] Add tests for new Core methods in beancore/links_test.go
- [x] Update graph/filters.go to use Core.IsBlocked()
- [x] Add GraphQL-level tests in schema.resolvers_test.go
- [x] Run all tests and verify

## Summary of Changes

- Added `IsBlocked(beanID string) bool` method to `beancore.Core` that checks if a bean is blocked by any active (non-completed, non-scrapped) blockers
- Added `FindActiveBlockers(beanID string) []*bean.Bean` method to find all active blockers for a bean
- Added `isResolvedStatus(status string) bool` helper function
- Updated `filterByIsBlocked()` and `filterByNotBlocked()` in `graph/filters.go` to use the new `Core.IsBlocked()` method
- Added comprehensive tests at both beancore and GraphQL resolver levels
- Verified with manual CLI testing that `beans list --ready` correctly shows beans after their blockers are completed or scrapped