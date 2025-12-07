---
title: Add beans types command for managing types in config.yaml
status: open
type: feature
created_at: 2025-12-07T16:00:49Z
updated_at: 2025-12-07T16:00:49Z
---

## Summary

Introduce a `beans types` command with subcommands for managing the types configured in `.beans/config.yaml`.

## Subcommands

- `beans types list` - List all configured types with their colors
- `beans types create <name>` - Create a new type (with optional `--color` flag)
- `beans types update <name>` - Update an existing type (e.g., change color)
- `beans types delete <name>` - Delete a type

## Implementation Notes

- Follow the same pattern as a potential `beans statuses` command would
- Validate that type names are unique when creating
- Warn or prevent deletion if beans exist with that type
- Support `--json` output for all subcommands
- Colors can be named colors or hex codes

## Checklist

- [ ] Add `types` command group in `cmd/`
- [ ] Implement `types list` subcommand
- [ ] Implement `types create` subcommand with `--color` flag
- [ ] Implement `types update` subcommand
- [ ] Implement `types delete` subcommand (with warning if types in use)
- [ ] Add `--json` support to all subcommands
- [ ] Write tests for all subcommands
- [ ] Update help text and documentation
