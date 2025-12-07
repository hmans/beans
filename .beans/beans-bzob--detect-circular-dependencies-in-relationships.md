---
title: Detect circular dependencies in relationships
status: open
created_at: 2025-12-07T11:29:37Z
updated_at: 2025-12-07T11:29:37Z
---

## Summary

Add cycle detection to identify circular dependencies in bean relationships, particularly for `blocks` links where cycles indicate impossible dependency chains.

## Requirements

- Detect cycles in `blocks` relationships (A blocks B, B blocks C, C blocks A)
- Report cycles in `beans check` output
- Consider whether other relationship types should also be checked for cycles

## Example output

```
beans check
Checking config... OK
Checking beans...
Circular dependency detected:
  beans-abc blocks beans-def
  beans-def blocks beans-ghi
  beans-ghi blocks beans-abc
```

## Notes

- Cycles in `parent` would also be problematic
- `relates-to` and `duplicates` are less concerning for cycles
- Could use depth-first search to detect back edges