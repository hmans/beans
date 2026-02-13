---
# beans-j7hd
title: Research Claude Code plugin/marketplace
status: completed
type: task
priority: normal
created_at: 2025-12-13T01:53:55Z
updated_at: 2026-02-12T00:15:29Z
---

Wrap up our work in progress Claude Code plugin/marketplace to make installation a little easier.

## Tasks

- [x] Finalize plugin README
- [x] Update main README Claude Code section
- [x] Add extraKnownMarketplaces to .claude/settings.json
- [x] Validate plugin structure
- [x] Commit changes

## Summary of Changes

- Replaced WIP plugin README with actual usage documentation
- Updated main README's Claude Code section: plugin install instructions up front, manual hook setup in a collapsible details block
- Added `extraKnownMarketplaces` and `enabledPlugins` to `.claude/settings.json` so projects using beans auto-prompt plugin installation
- Validated plugin structure with `claude plugin validate` (passes)
