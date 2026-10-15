---
# beans-l4t4
title: Render bean bodies without glamour's document margin
status: completed
type: task
priority: normal
created_at: 2026-10-01T14:46:12Z
updated_at: 2026-10-01T14:47:46Z
---

Glamour's built-in styles indent every rendered markdown document by 2 columns per side (Document.Margin). The space carries nothing. Remove it in `beans show` and in the TUI (detail view and preview pane).

- [x] Shared helper in internal/ui returning glamour styles with Document.Margin = 0
- [x] beans show: auto style (dark/light/notty) without margin
- [x] TUI detail + preview: dark style without margin; drop glamourMarginX from preview geometry
- [x] Tests

## Summary of Changes

- New internal/ui/markdown.go: DarkMarkdownStyle() and AutoMarkdownStyle() return copies of glamour styles with Document.Margin = 0; AutoMarkdownStyle mirrors glamour.WithAutoStyle (notty / dark / light).
- beans show uses AutoMarkdownStyle; text now wraps at the full 80 columns.
- TUI detail view and preview pane use DarkMarkdownStyle; glamourMarginX removed from previewChromeX.
- Tests in internal/ui/markdown_test.go: rendered text starts at column 0, glamour defaults stay unmodified.
