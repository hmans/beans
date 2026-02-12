---
# beans-jvkq
title: beans start command
status: completed
type: task
priority: normal
created_at: 2025-12-27T21:44:04Z
updated_at: 2026-02-12T00:26:36Z
parent: beans-mmyp
---

Add `beans start <id>` command.

## Behavior

- Sets status to `in-progress`
- Displays the bean details (like `beans show`) so you can see what you're working on
- Could optionally check if another bean is already in-progress and warn

## Example

```bash
beans start beans-abc
# Sets to in-progress and shows the bean
```
