---
# beans-digu
title: Support custom properties on beans
status: completed
type: feature
priority: normal
created_at: 2025-12-13T00:52:24Z
updated_at: 2026-02-14T20:30:39Z
---

Allow users to attach custom key-value properties to beans. Custom properties should live under a dedicated `properties` key in the frontmatter to keep them separate from built-in fields.

## Example

```yaml
---
title: Fix authentication bug
status: in-progress
type: bug
properties:
  github_issue: "#135"
  author: alice@bob.com
  estimate: 3
  reviewed: true
---
```

## Considerations

- Properties can be any YAML-supported type (string, number, boolean, etc.)
- Should be exposed via GraphQL (probably as JSON scalar or key-value pairs)
- Could support filtering/searching by property values in the future
- CLI: `beans update <id> --set key=value` or similar

## Summary of Changes

- Added `Properties map[string]any` field to `Bean`, `frontMatter`, and `renderFrontMatter` structs
- Added helper methods: `SetProperty`, `UnsetProperty`, `GetProperty` (with nil-map safety and empty→nil normalization)
- Added `scalar JSON` to GraphQL schema mapped to gqlgen's `graphql.Map`
- Added `properties` field to `Bean` type, `CreateBeanInput`, and `UpdateBeanInput` (with `setProperties`/`unsetProperties` for granular updates)
- Resolver enforces mutual exclusivity between `properties` and `setProperties`/`unsetProperties`
- CLI: `--set key=value` (repeatable) on both `create` and `update`, `--unset key` on `update`
- Value types auto-detected via YAML unmarshaling (3→int, true→bool, 4.5→float, text→string)
- Properties displayed in `beans show` output between relationships and body
- Full test coverage across all layers (bean model, GraphQL resolvers, CLI flag parser)
