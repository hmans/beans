---
title: Add issue relationships
status: done
type: feature
created_at: 2025-12-06T22:04:39Z
updated_at: 2025-12-07T11:01:22Z
---










Add relationship fields to beans for expressing dependencies and connections.

- Relationships are directional (e.g., issue A blocks issue B).
- Only one of them needs to be set on one issue to establish the relationship between two issues. The code that loads the store will infer the reverse relationship automatically.
- Supported relationship types are "active": `blocks`, `duplicate`, `related` and `parent`.
- When loading the data, the system will automatically infer and populate the reverse relationships: issues have inferred slices for `blocked-by`, `duplicated-by`, `relates-to` (symmetric), and `children`.
- Relationships are expressed as data in the frontmatter Yaml. The root key is `links`, with subkeys for each relationship type. The value of each subkey is either a single bean ID, or an array of bean IDs.

## Checklist

- [ ] Design relationship data structure for Bean struct
- [ ] Add relationship fields to frontmatter parsing/rendering
- [ ] Implement reverse relationship inference in store.FindAll()
- [ ] Add relationship flags to \`beans update\` command
- [ ] Add relationship display to \`beans show\` command
- [ ] Add \`--filter\` support for relationship queries (e.g., \`!blocked-by\`)
- [ ] Consider \`beans graph\` command for visualization (optional)
- [ ] Unit tests for relationship handling and reverse inference

## Notes

- Relationships reference beans by ID
- Invalid/missing bean references should warn, not error
- Consider what happens when a referenced bean is deleted

## Context

Part of the issue metadata expansion. See original planning bean: beans-v8qj
