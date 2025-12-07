---
title: Show relationship counts in list output
status: open
created_at: 2025-12-07T11:29:37Z
updated_at: 2025-12-07T11:29:37Z
---

## Summary

Add optional indicators in `beans list` showing how many incoming and outgoing links each bean has.

## Requirements

- Add `--show-links` flag (or similar) to show link counts
- Display format like "→2 ←1" meaning 2 outgoing links, 1 incoming
- Consider adding to default output if counts are non-zero
- JSON output should already include links, but could add computed counts

## Example output

```
beans list --show-links
ID          STATUS      TYPE    LINKS   TITLE
beans-abc   open        task    →2 ←1   Implement feature X
beans-def   open        bug     ←1      Fix login issue
beans-ghi   done        task            Clean up old code
```

## Notes

- Keep it compact to not clutter the output
- Could use emoji or symbols: 🔗2 or [2→1←]
- Useful for quickly spotting highly connected beans