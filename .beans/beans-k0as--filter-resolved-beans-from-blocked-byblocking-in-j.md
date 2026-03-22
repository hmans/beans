---
# beans-k0as
title: Filter resolved beans from blocked_by/blocking in JSON output
status: completed
type: bug
priority: high
created_at: 2026-03-22T04:24:00Z
updated_at: 2026-03-22T04:24:03Z
---

## Problem

The `blocked_by` and `blocking` arrays returned in CLI JSON output (`beans show --json`, `beans list --json`) and via the GraphQL `blockingIds`/`blockedByIds` fields reflect raw frontmatter — they include bean IDs regardless of whether those beans have been completed or scrapped.

This causes agents and scripts to misread a bean as still blocked when all its blockers are already done. The runtime `--ready` filter correctly ignores resolved blockers (via `findActiveBlockersLocked`), but the serialized data does not.

## Fix

Added `ActiveBlockedByIds` and `ActiveBlockingIds` methods on `Core` that filter out resolved (completed/scrapped) beans using the existing `isResolvedStatus` check. Applied in:

- `BeanBlockedByIds` / `BeanBlockingIds` GraphQL resolvers
- `beans show --json` and `beans list --json` CLI output

## Summary of Changes

- `pkg/beancore/links.go`: Added `ActiveBlockedByIds` and `ActiveBlockingIds`
- `pkg/beangraph/bean_fields.go`: Resolvers now return active-only IDs
- `internal/commands/show.go`: Filter before JSON serialization
- `internal/commands/list.go`: Filter before JSON serialization
- `pkg/beancore/links_test.go`: Tests for both new methods
