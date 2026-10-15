---
# beans-cwsf
title: TUI preview pane too narrow for 80-column markdown
status: completed
type: bug
priority: normal
created_at: 2026-10-01T12:57:36Z
updated_at: 2026-10-01T12:58:41Z
---

The two-column preview pane is 80 wide, but border (2), padding (2) and glamour's document margin (2+2) leave only 72 columns of paragraph text. Glamour also wraps at its default 80 while the pane wraps again at 76, so rendered lines break twice (e.g. a lone "(anvl-" line).

- [x] Glamour in the preview wraps at the pane's content width
- [x] Preview pane is wide enough for 80 columns of paragraph text
- [x] Tests

## Summary of Changes

- The preview renders markdown with a glamour renderer wrapping at the pane's content width (cached per width), so lines wrap once.
- RightPaneMaxWidth 80 → 88: 80 columns paragraph text + glamour margins + padding + border.
- TestPreviewWrapsBodyOnce covers pane overflow, double wrapping and an intact 80-column line.
