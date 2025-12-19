---
# beans-hz87
title: Add blocked-by relationship (possibly replacing blocking)
status: todo
type: feature
priority: normal
created_at: 2025-12-14T14:37:11Z
updated_at: 2025-12-14T15:05:05Z
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

- [ ] Decide: replace `blocking` or add `blocked-by` alongside it
- [ ] Update front matter parsing to support `blocked-by`
- [ ] Update GraphQL schema with new field/relationship
- [ ] Update CLI commands (`beans update --blocked-by`, `--remove-blocked-by`)
- [ ] Update `beans prime` documentation
- [ ] Migrate or deprecate existing `blocking` if replacing
- [ ] Update tests