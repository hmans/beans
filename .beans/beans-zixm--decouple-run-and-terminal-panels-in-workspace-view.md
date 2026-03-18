---
# beans-zixm
title: Decouple Run and Terminal panels in workspace view
status: completed
type: bug
priority: normal
created_at: 2026-03-18T13:15:57Z
updated_at: 2026-03-18T13:18:45Z
---

Run and Terminal panels should be independent. Run alone should take the full bottom panel. Terminal alone should take the full bottom panel. When both are open, show side-by-side with a draggable separator.

## Summary of Changes

- Decoupled Run and Terminal (Shell) panels so they are independent
- Run panel takes full bottom panel width when Terminal is not open
- Terminal takes full width when Run is not active
- When both are open, they display side-by-side with a draggable separator (using nested SplitPane)
- Removed the forced `ui.showTerminal = true` from `handleRun()`
- Updated e2e tests to reflect the new independent behavior
