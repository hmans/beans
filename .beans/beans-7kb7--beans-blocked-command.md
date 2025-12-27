---
# beans-7kb7
title: beans blocked command
status: todo
type: task
created_at: 2025-12-27T21:44:05Z
updated_at: 2025-12-27T21:44:05Z
parent: beans-mmyp
---

Add `beans blocked` command to show beans that are currently blocked.

## Behavior

- Lists beans that have blockers (other beans blocking them)
- Shows what is blocking each bean
- Helps identify bottlenecks in the project

## Example

```bash
beans blocked
# Shows:
# beans-abc: "Add auth" (blocked by beans-xyz: "Set up database")
```