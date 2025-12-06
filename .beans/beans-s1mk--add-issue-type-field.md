---
title: Add issue type field
status: done
created_at: 2025-12-06T22:04:38Z
updated_at: 2025-12-06T22:11:32Z
---



Add a `type` field to bean frontmatter for categorizing beans (task, bug, epic, etc.).

## Requirements
- Free-form string field, not validated against a whitelist
- Unknown types are allowed (don't crash on unrecognized values)
- Optional `[[types]]` config in beans.toml for color/display hints

## Checklist
- [ ] Add `Type string` field to Bean struct in `internal/bean/bean.go`
- [ ] Update frontmatter parsing/rendering
- [ ] Add `--type` flag to `beans create` command
- [ ] Add `--type` flag to `beans update` command
- [ ] Add `type` to JSON output
- [ ] Add optional `[[types]]` config section to beans.toml
- [ ] Unit tests for type field handling

## Context
Part of the issue metadata expansion. See original planning bean: beans-v8qj