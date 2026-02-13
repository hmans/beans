---
# beans-18db
title: beans milestones command
status: completed
type: task
priority: normal
created_at: 2025-12-27T21:44:05Z
updated_at: 2026-02-12T00:26:36Z
parent: beans-mmyp
---

Add `beans milestones` command to list planned milestones.

## Behavior

- Lists all beans with type `milestone`
- Shows progress (e.g., "3/10 children completed")
- Sorted by priority or a target date if we add that field
- Excludes completed/scrapped milestones by default (`--all` to include)

## Example

```bash
beans milestones
# Lists active milestones with their progress
```
