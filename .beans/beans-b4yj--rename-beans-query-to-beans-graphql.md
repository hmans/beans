---
title: Rename 'beans query' to 'beans graphql'
status: backlog
type: feature
created_at: 2025-12-12T22:38:27Z
updated_at: 2025-12-12T22:38:27Z
parent: beans-xnp8
---

Rename the `beans query` command to `beans graphql` for clarity.

The command supports both queries AND mutations, so 'query' is a misnomer. 'graphql' better represents what it actually does.

## Checklist
- [ ] Rename cmd/query.go to cmd/graphql.go
- [ ] Update command name from 'query' to 'graphql'
- [ ] Keep 'query' as a hidden alias for backwards compatibility
- [ ] Update documentation and help text