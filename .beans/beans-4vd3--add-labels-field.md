---
title: Add labels field
status: open
created_at: 2025-12-06T22:04:39Z
updated_at: 2025-12-06T22:04:39Z
---

Add a `labels` array field to bean frontmatter for tagging/categorizing beans.

## Requirements
- Free-form string array, no pre-declaration required
- Users can add any label without defining it first
- Optional `[[labels]]` config in beans.toml for color customization
- Repeatable `--label` flag for CLI

## Checklist
- [ ] Add `Labels []string` field to Bean struct in `internal/bean/bean.go`
- [ ] Update frontmatter parsing/rendering for YAML arrays
- [ ] Add `--label` flag (repeatable) to `beans create` command
- [ ] Add `--label` flag (repeatable) to `beans update` command
- [ ] Add `labels` to JSON output
- [ ] Add optional `[[labels]]` config section to beans.toml for colors
- [ ] Consider `--remove-label` flag for update command
- [ ] Unit tests for labels field handling

## Context
Part of the issue metadata expansion. See original planning bean: beans-v8qj