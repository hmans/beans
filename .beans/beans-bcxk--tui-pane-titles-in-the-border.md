---
# beans-bcxk
title: 'TUI: pane titles in the border'
status: completed
type: feature
priority: normal
created_at: 2026-10-01T14:52:52Z
updated_at: 2026-10-01T14:55:11Z
---

The list's "Beans" title bar takes two rows (title plus padding) and the preview spends a row on the bean ID. Render both in the top border of their pane instead.

- [x] List title (and filter input while filtering) in the list's top border
- [x] Bean ID in the preview's top border; drop the ID line from the header
- [x] Mouse hit-testing follows the new row offsets
- [x] Tests updated

## Summary of Changes

- `withBorderTitle` (internal/tui/styles.go) writes a title into the top border of a rounded-border pane, truncating it to fit.
- The list hides the bubbles title bar and filter row; its border shows the title, or the filter input while the user types a filter. This gains two list rows.
- The preview border shows the bean ID; the header starts with the title. This gains one preview row.
- List click hit-testing no longer offsets by a title bar.
- Tests: border titles for list, filter and preview; truncation keeps the pane width.
