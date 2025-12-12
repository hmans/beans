---
title: 'Simplify bean linking: parent + blocks'
status: completed
type: feature
priority: normal
created_at: 2025-12-12T15:49:19Z
updated_at: 2025-12-12T16:16:19Z
---

Replace generic links array with explicit parent (scalar) and blocks (array) fields. Remove duplicates and related link types.

## Breaking Changes
- links array removed from frontmatter
- duplicates and related link types removed
- CLI: --link type:id becomes --parent id and --block id
- GraphQL: addLink/removeLink becomes setParent/addBlock/removeBlock

## Checklist
- [ ] Update Bean struct (internal/bean/bean.go)
- [ ] Update Beancore (internal/beancore/)
- [ ] Update GraphQL schema
- [ ] Run code generation
- [ ] Update GraphQL resolvers
- [ ] Update filters
- [ ] Update CLI commands
- [ ] Update tests
- [ ] Final validation