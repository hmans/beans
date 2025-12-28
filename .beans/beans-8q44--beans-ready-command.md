---
# beans-8q44
title: beans ready command
status: scrapped
type: task
priority: normal
created_at: 2025-12-27T21:44:04Z
updated_at: 2025-12-28T11:47:36Z
parent: beans-mmyp
---

Add `beans ready` command to find beans that are ready to work on.

## Behavior

- Shows beans with actionable status (`todo`) that are not blocked
- Sorted by priority (critical → high → normal → low → deferred)
- Excludes `completed`, `scrapped`, `draft`, and `in-progress`
- Supports `--json` output

## Example

```bash
beans ready
# Lists all beans ready to pick up
```