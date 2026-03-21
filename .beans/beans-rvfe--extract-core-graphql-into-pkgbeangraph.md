---
# beans-rvfe
title: Extract core GraphQL into pkg/beangraph
status: completed
type: task
priority: normal
created_at: 2026-03-21T08:11:51Z
updated_at: 2026-03-21T08:24:36Z
---

Split the GraphQL layer: core bean CRUD into pkg/beangraph/ (public), UI-specific resolvers stay in internal/graph/. Move model types to pkg/beangraph/model/. CLI commands switch to using CoreResolver directly, dropping internal/graph dependency.

## Tasks

- [x] Create `pkg/beangraph/` package with `CoreResolver` struct
- [x] Move model types from `internal/graph/model/` to `pkg/beangraph/model/`
- [x] Update gqlgen.yml to generate models in new location
- [x] Move core bean query resolvers to `pkg/beangraph/queries.go`
- [x] Move core bean mutation resolvers to `pkg/beangraph/mutations.go`
- [x] Move bean field resolvers to `pkg/beangraph/bean_fields.go`
- [x] Move filter logic to `pkg/beangraph/filters.go`
- [x] Move ETag/validation helpers to `pkg/beangraph/resolver.go`
- [x] Update `internal/graph/` to embed `CoreResolver` and delegate
- [x] Update CLI commands to use `beangraph.CoreResolver` directly
- [x] Run codegen, verify build compiles
- [x] Run tests

## Summary of Changes

- Created `pkg/beangraph/` package with `CoreResolver` struct containing all core bean CRUD operations
- Moved model types from `internal/graph/model/` to `pkg/beangraph/model/` (updated gqlgen.yml)
- Extracted core resolvers: queries (Bean, Beans, ProjectName, MainBranch), mutations (CreateBean, UpdateBean, DeleteBean, SetParent, Add/RemoveBlocking, Add/RemoveBlockedBy, ArchiveBean), bean field resolvers, filter logic, and ETag/validation helpers
- `internal/graph/Resolver` now embeds `*beangraph.CoreResolver` and delegates core operations
- CLI commands (`create`, `list`, `show`, `update`, `delete`, `roadmap`) now use `beangraph.CoreResolver` directly, removing their dependency on `internal/graph`
- TUI also migrated to use `beangraph.CoreResolver` directly
- Only `graphql.go` and `serve.go` still import `internal/graph` (expected — they need the full gqlgen schema)
- All 16 test packages pass, codegen works
