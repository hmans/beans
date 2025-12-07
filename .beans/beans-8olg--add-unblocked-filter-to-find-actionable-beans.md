---
title: Add --unblocked filter to find actionable beans
status: open
type: feature
created_at: 2025-12-07T11:29:37Z
updated_at: 2025-12-07T11:36:39Z
---


## Summary

Add an `--unblocked` filter to `beans list` that shows only beans with no unresolved blockers.

## Requirements

- `beans list --unblocked` returns beans that are NOT blocked by any open bean
- A bean is considered blocked if another bean has it in its `blocks` links AND that blocking bean is not done/archived
- Useful for finding work that can be started immediately

## Example

Given:
- beans-abc (open) blocks beans-def
- beans-ghi (done) blocks beans-jkl

```
beans list --unblocked
```

Returns beans-jkl (blocker is done) but NOT beans-def (blocker is open).

## Notes

- Should combine with other filters: `beans list --status open --unblocked`
- Consider inverse: `--blocked` to show only blocked beans
- Could extend to show WHY a bean is blocked