---
# beans-tbtr
title: 'Two-column layout polish: footer, pane widths, preview height'
status: completed
type: task
created_at: 2025-12-28T19:44:32Z
updated_at: 2025-12-28T19:44:32Z
parent: t0tv
---

Several polish fixes for the two-column TUI layout:

- Footer is now app-global, spanning full terminal width (not constrained to left pane)
- Right pane capped at 80 chars max width (text files follow 80-char convention), left pane gets remaining space
- Preview height properly constrained to prevent overflow when bean body is long
- Detail view linked beans show full type/status names instead of single-char abbreviations