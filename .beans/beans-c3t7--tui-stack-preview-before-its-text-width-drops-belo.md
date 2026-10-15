---
# beans-c3t7
title: 'TUI: stack preview before its text width drops below 80; name layout constants'
status: completed
type: task
priority: normal
created_at: 2026-10-01T13:28:04Z
updated_at: 2026-10-01T13:46:50Z
---

Two columns only when the preview keeps 80 columns of net text; otherwise prefer the stacked layout. Replace magic layout numbers in tui.go, preview.go and list.go by named constants.

- [x] Switch to stacked when the two-column preview would drop below 80 net
- [x] Named constants for layout numbers
- [x] Tests

## Summary of Changes

- Two columns from TwoColumnFullWidth (= LeftPaneMinWidth + RightPaneMaxWidth = 128); between TwoColumnMinWidth and that, tall terminals stack and short ones keep a shrunk side preview.
- Pane geometry constants (paneBorder(s), footerHeight, paneSeparator, listBottomPadding, previewPaddingX, glamourMarginX, previewChromeX) replace the literals in tui.go, preview.go and list.go; RightPaneMaxWidth and StackedPreviewMaxWidth derive from text widths.
- TestLayoutSelection covers the five width/height regions.
