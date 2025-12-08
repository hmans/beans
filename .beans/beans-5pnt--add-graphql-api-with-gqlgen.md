---
title: Add GraphQL API with gqlgen
status: completed
type: feature
created_at: 2025-12-08T19:18:10Z
updated_at: 2025-12-08T19:28:40Z
---

Add a GraphQL API layer using gqlgen with read-only queries via a `beans query` CLI command.

## Scope
- internal/graph/ package with schema and resolvers
- `beans query` command for programmatic execution (no HTTP)
- Read-only: bean(id) and beans(filter) queries
- Computed relationship fields (blockedBy, blocks, parent, children)
- Schema docs in cmd/prompt.md

## Checklist
- [x] Install gqlgen and create gqlgen.yml config
- [x] Create GraphQL schema (schema.graphqls)
- [x] Generate code and implement resolvers
- [x] Create `beans query` command
- [x] Update cmd/prompt.md with schema docs
- [x] Add tests