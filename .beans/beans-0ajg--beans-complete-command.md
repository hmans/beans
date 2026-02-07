---
# beans-0ajg
title: beans complete command
status: todo
type: task
created_at: 2025-12-27T21:44:04Z
updated_at: 2025-12-27T21:44:04Z
parent: beans-mmyp
---

Add `beans complete <id> [--summary <text>]` command.

## Behavior

- Sets status to `completed`
- Optional `--summary` flag to add a completion note to the bean body
- Shows confirmation message with bean title

## Example

```bash
beans complete beans-abc --summary "Implemented via PR #42"
```