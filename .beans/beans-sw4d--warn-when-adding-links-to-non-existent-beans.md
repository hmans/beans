---
title: Warn when adding links to non-existent beans
status: done
type: bug
created_at: 2025-12-07T11:35:24Z
updated_at: 2025-12-07T16:30:47Z
---


## Summary

When using `beans update <id> --link type:target`, there's no validation that the target bean exists. This can lead to broken links that are only discovered later.

## Suggested Behavior

- When adding a link via `--link`, check if the target bean ID exists
- If it doesn't exist, print a warning (but still create the link)
- Don't make it an error — the user might intend to create the target bean later

## Example

```
beans update beans-abc --link blocks:nonexistent
Warning: Target bean 'nonexistent' does not exist
Updated beans-abc
```

## Notes

- This is a softer version of the validation in beans-co1q (which focuses on `beans check`)
- The warning should respect `--json` output format
- Consider a `--strict` flag that makes this an error instead of a warning
