---
title: Enhance beans check to identify and fix broken links
status: open
type: feature
created_at: 2025-12-07T11:27:02Z
updated_at: 2025-12-07T17:08:36Z
links:
    - parent: beans-7lmv
---


## Summary

Add broken link detection to `beans check` command, with optional auto-fix capability.

## Requirements

- `beans check` should scan all beans and identify links that reference non-existent bean IDs
- Report broken links with the source bean ID, link type, and missing target ID
- Add `--fix` flag that automatically removes broken links from beans
- JSON output should include broken links in the response

## Example output

```
beans check
Checking config... OK
Checking beans...
  beans-abc: broken link blocks:nonexistent-id
  beans-def: broken link parent:deleted-bean
Found 2 broken links

beans check --fix
Checking config... OK
Checking beans...
  beans-abc: removed broken link blocks:nonexistent-id
  beans-def: removed broken link parent:deleted-bean
Fixed 2 broken links
```

## Checklist

- [ ] Add broken link detection logic to check command
- [ ] Display broken links in human-friendly output
- [ ] Add `--fix` flag to remove broken links
- [ ] Update JSON output to include broken links
- [ ] Add tests for broken link detection