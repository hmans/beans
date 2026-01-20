---
# beans-hz87
title: Add blocked-by relationship (possibly replacing blocking)
status: completed
type: feature
priority: normal
created_at: 2025-12-14T14:37:11Z
updated_at: 2026-01-20T08:19:45Z
parent: beans-f11p
---

## Summary

Add a `blocked-by` link relationship to beans, which may replace or complement the existing `blocking` relationship.

## Motivation

The current `blocking` relationship requires the blocker to declare what it's blocking. However, in practice, it's typically the **blocked** bean that knows why it can't proceed yet - not the other way around.

For example:
- "Implement user dashboard" is blocked by "Set up authentication" 
- The dashboard feature knows it needs auth first; the auth feature doesn't necessarily know what depends on it

A `blocked-by` relationship is more natural because:
1. **Context stays with the blocked item** - The bean that can't proceed documents its own dependencies
2. **Easier to maintain** - When creating a new feature, you know what you're waiting on
3. **Better discoverability** - Reading a bean tells you everything about why it's blocked

## Design Considerations

- Should `blocked-by` replace `blocking`, or should both exist?
- If both exist, should they be bidirectional (adding one creates the inverse)?
- How does this affect the GraphQL schema and queries?
- Migration path for existing `blocking` relationships

## Checklist

- [x] Decide: add `blocked-by` alongside `blocking` (both coexist)
- [x] Update front matter parsing to support `blocked-by`
- [x] Update GraphQL schema with new field/relationship
- [x] Update CLI commands (`beans update --blocked-by`, `--remove-blocked-by`, and `beans create --blocked-by`)
- [x] Update `beans prime` documentation
- [x] Keep both `blocking` and `blocked-by` (no migration needed)
- [x] Update tests

## Summary of Changes

Added `blocked_by` as a new stored field that coexists with the existing `blocking` field:

### Core Changes
- Added `BlockedBy []string` field to Bean struct with YAML/JSON serialization
- Added `IsBlockedBy()`, `AddBlockedBy()`, `RemoveBlockedBy()` helper methods
- Added `blockedByIds` field to GraphQL Bean type
- Added `blockedBy` to CreateBeanInput
- Added `addBlockedBy` and `removeBlockedBy` GraphQL mutations
- Added new filter options: `hasBlockedBy`, `blockedById`, `noBlockedBy`
- Updated `isBlocked` filter to check both incoming `blocking` links AND direct `blocked_by` entries

### CLI Changes
- Added `--blocked-by` flag to `beans create` command
- Added `--blocked-by` and `--remove-blocked-by` flags to `beans update` command

### Link Management
- Updated `FindIncomingLinks()` to include `blocked_by` link type
- Updated `CheckAllLinks()` to validate `blocked_by` links
- Updated `RemoveLinksTo()` to clean up `blocked_by` references
- Updated `FixBrokenLinks()` to repair broken `blocked_by` links
- Added cycle detection for `blocked_by` relationships

### Documentation
- Updated `prompt.tmpl` with `--blocked-by` examples and relationship documentation

Refs: #62