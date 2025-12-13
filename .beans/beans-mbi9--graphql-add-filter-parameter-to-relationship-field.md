---
title: 'GraphQL: add filter parameter to relationship fields'
status: todo
type: feature
priority: normal
created_at: 2025-12-13T00:49:39Z
updated_at: 2025-12-13T02:02:11Z
---

Add optional filter parameter to relationship fields (children, blocking, blockedBy) to allow filtering related beans directly in nested queries.

Example:
```graphql
{ bean(id: "beans-xnp8") { 
  children(filter: { status: ["todo"] }) { id title } 
} }
```

Currently you can work around this with a top-level query using parentId/blockingId filters, but supporting filters on relationship fields would be more ergonomic for nested queries.