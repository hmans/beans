---
# beans-3kd3
title: Automatically fix broken bean links
status: todo
type: task
created_at: 2025-12-27T19:32:26Z
updated_at: 2025-12-27T19:32:26Z
---

When beans are deleted or their IDs change, other beans may be left with broken references (parent or blocking fields pointing to non-existent beans).

## Problem

Beans can have references to other beans that no longer exist:
- `parent: beans-xyz` where `beans-xyz` was deleted
- `blocking: [beans-abc]` where `beans-abc` no longer exists

This can happen when:
- Beans were manually deleted from the filesystem
- Beans were archived before the non-destructive archive feature was implemented
- Bean files were corrupted or lost

## Proposed Solution

Add a `beans doctor` or `beans fix` command that:
1. Scans all beans for broken references
2. Reports which beans have invalid links
3. Optionally removes the broken references (with `--fix` flag)

## Considerations

- Should run on startup as a warning? Or only on-demand?
- Should be included in `beans archive` as a cleanup step?
- Consider adding `--dry-run` to show what would be fixed