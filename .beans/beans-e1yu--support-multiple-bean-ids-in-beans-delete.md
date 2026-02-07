---
# beans-e1yu
title: Support multiple bean IDs in beans delete
status: completed
type: feature
priority: normal
created_at: 2026-01-20T08:06:07Z
updated_at: 2026-01-20T08:12:39Z
---

Allow beans delete to accept multiple IDs like beans show does. GitHub issue #64.

## Summary of Changes

- Changed `cobra.ExactArgs(1)` to `cobra.MinimumNArgs(1)` to accept multiple IDs
- Updated command usage from `delete <id>` to `delete <id> [id...]`
- Added `beanWithLinks` struct to hold bean and its incoming links for batch processing
- Implemented fail-fast validation: all beans are validated before any are deleted
- Created `confirmDeleteMultiple()` function with batch confirmation UX:
  - Single bean: same UX as before (backward compatible)
  - Multiple beans: lists all beans with incoming link indicators, single confirmation
- JSON output: single bean returns `{"bean": ...}`, multiple returns `{"beans": [...], "count": N}`
- Non-interactive output: shows total references removed and lists all deleted files
- Updated `cmd/prompt.tmpl` to document the delete command for agents