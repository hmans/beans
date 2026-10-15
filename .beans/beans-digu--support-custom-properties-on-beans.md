---
# beans-digu
title: Support custom properties on beans
status: completed
type: feature
priority: normal
created_at: 2025-12-13T00:52:24Z
updated_at: 2026-10-06T11:44:46Z
order: s
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

## Decisions

- Values are strings only (`map[string]string`). YAML scalars like `3` or `true` are read as strings and written back quoted.
- GraphQL exposes properties as `[Property!]!` with `type Property { key: String!, value: String! }`, sorted by key.
- Filtering by properties and editing in the UI are out of scope (follow-up bean).
- `format_version` stays unchanged.

## Todo

- [x] Data model: `Properties` in `Bean`, `frontMatter`, `renderFrontMatter`; key validation
- [x] GraphQL schema: `Bean.properties`, `CreateBeanInput.properties`, `UpdateBeanInput.setProperties/removeProperties`; `mise codegen`
- [x] Resolvers: create + update merge logic
- [x] CLI: `--property key=value` on create/update, `--remove-property key` on update, display in `beans show`
- [x] TUI detail view display
- [x] UI: display in BeanDetail, operations.graphql, e2e test
- [x] Agent prompt docs (prompt.tmpl)
- [x] Tests (parse/render roundtrip, mutations, CLI flags)

## Summary of Changes

- `pkg/bean`: `Bean.Properties map[string]string`, read from and written to the `properties` front matter key (keys sorted, omitted when empty); `ValidatePropertyKey` (`^[a-z][a-z0-9_-]*$`)
- GraphQL: `Bean.properties: [Property!]!` sorted by key, `CreateBeanInput.properties`, `UpdateBeanInput.setProperties` / `removeProperties`; keys validated in the resolver
- CLI: `--property key=value` on `create` and `update`, `--remove-property key` on `update`, properties section in `beans show`; JSON output carries `properties`
- TUI: properties in preview and detail header
- Beans UI: Properties section in the bean detail view, Playwright spec `e2e/bean-properties.spec.ts`
- Agent prompt documents the new flags

Follow-up: beans-1ae4 (filtering, UI editing, format version).
