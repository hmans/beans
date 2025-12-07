---
title: Add epics / directory-based parent beans
status: open
type: epic
created_at: 2025-12-06T22:04:39Z
updated_at: 2025-12-06T22:04:39Z
---

Allow beans to be directories, enabling parent-child hierarchies (epics).

## Requirements
- Beans can be either files OR directories
- Directory beans contain \`bean.md\` for their own content
- Child beans are \`.md\` files inside the parent directory
- Children have implicit \`parent\` relationship to directory bean
- Structural guarantee: no cyclical parent relationships possible

## Structure
\`\`\`
.beans/
├── beans-abc/           # Epic (directory)
│   ├── bean.md          # Epic's own content
│   ├── beans-def.md     # Child task
│   └── beans-ghi.md     # Another child
└── beans-xyz.md         # Standalone bean (file)
\`\`\`

## Checklist
- [ ] Update store to recognize both file and directory beans
- [ ] Directory beans load \`bean.md\` for their content
- [ ] Implement child bean discovery within directories
- [ ] Add implicit \`parent\` field to child beans
- [ ] Update ID/path parsing for nested structure
- [ ] Add \`--parent\` flag to \`beans create\` command
- [ ] Add \`beans list --tree\` for hierarchical view
- [ ] Ensure migration: existing flat beans continue to work
- [ ] Unit tests for directory bean handling

## Notes
- This is a significant structural change - save for last
- Consider depth limits (2-3 levels max?)
- ID uniqueness must be global, not just within a directory

## Context
Part of the issue metadata expansion. See original planning bean: beans-v8qj