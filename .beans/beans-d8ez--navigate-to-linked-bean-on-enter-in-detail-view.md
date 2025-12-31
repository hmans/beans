---
# beans-d8ez
title: Navigate to linked bean on Enter in detail view
status: completed
type: task
priority: normal
created_at: 2025-12-30T21:58:18Z
updated_at: 2025-12-30T22:00:17Z
parent: beans-pn6z
---

When pressing Enter on a linked bean in the detail view's linked beans list, the TUI should:
1. Move the list cursor to that bean
2. Re-render the detail view showing the selected bean

Currently `moveCursorToBean()` calls `Select()` on the bubbles list but doesn't trigger `cursorChangedMsg`, so the detail view isn't updated.