---
title: Add issue relationships
status: open
type: feature
created_at: 2025-12-06T22:04:39Z
updated_at: 2025-12-06T22:04:39Z
---

Add relationship fields to beans for expressing dependencies and connections.

## Requirements
- Hard-coded relationship types (can make configurable later):
  - \`blocks\` ↔ \`blocked-by\`
  - \`duplicates\` ↔ \`duplicated-by\`
  - \`relates-to\` ↔ \`relates-to\` (symmetric)
- Automatic reverse relationship inference when loading beans
- Frontmatter format: \`blocks: [beans-abc, beans-def]\`

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