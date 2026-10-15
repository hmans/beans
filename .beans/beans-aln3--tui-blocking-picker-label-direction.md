---
# beans-aln3
title: 'TUI blocking picker: label direction'
status: completed
type: bug
priority: normal
created_at: 2026-10-02T09:21:06Z
updated_at: 2026-10-02T09:22:02Z
---

The TUI 'Manage Blocking' picker does not say who blocks whom. A filled dot means the current bean blocks the listed bean.

- [x] Title 'Blocks:' instead of 'Manage Blocking'
- [x] Legend '● = blocked by <id>' in description line

## Summary of Changes

- Picker list title and modal title read 'Blocks:'.
- Description shows the legend '● = blocked by <id>' on its own line above the key hints; list height reduced by one line to fit.
- No test: user decided string-only changes need none.
