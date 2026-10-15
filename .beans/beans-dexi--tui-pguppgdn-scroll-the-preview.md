---
# beans-dexi
title: 'TUI: PgUp/PgDn scroll the preview'
status: completed
type: feature
priority: normal
created_at: 2026-10-01T14:32:49Z
updated_at: 2026-10-01T14:35:40Z
---

When a preview is visible (two-column or stacked), PgUp/PgDn scroll it by previewScrollStep (3) rows instead of paging the list. Without a preview they keep paging the list.

## Todo
- [x] Key handling in App.Update
- [x] Help overlay entry
- [x] Tests

## Summary of Changes

- `App.Update`: in the list view, outside filter input, with a preview visible, `pgup`/`pgdown` scroll the preview by `previewScrollStep` and are not forwarded to the list.
- Help overlay lists `pgup/pgdn  Scroll preview`.
- `TestPreviewPageKeys` covers two-column, stacked and list-only layouts.
