---
# beans-8x3r
title: 'TUI: y should copy all selected beans, not just highlighted one'
status: completed
type: bug
priority: normal
created_at: 2025-12-28T00:49:03Z
updated_at: 2025-12-28T00:55:06Z
---

Currently y only yanks the highlighted bean. The rest of the UI supports multi-select. Yank should support multi-select as well.

## Implementation Plan

Make `y` (yank) in the TUI respect multi-select state, copying all selected bean IDs comma-separated instead of just the highlighted one.

### Changes Required

1. **`internal/tui/tui.go`**: Change `copyBeanIDMsg` from single `id string` to `ids []string`
2. **`internal/tui/tui.go`**: Update message handler to join IDs with comma, adjust status message
3. **`internal/tui/list.go`**: Update `y` key handler to check `selectedBeans` first (like other multi-select ops)
4. **`internal/tui/detail.go`**: Update `y` handler to use `[]string{m.bean.ID}`

### Behavior

- When beans selected (space): copy all selected IDs comma-separated
- When nothing selected: copy highlighted bean ID (unchanged)
- Status: "Copied 3 bean IDs to clipboard" vs "Copied beans-xyz to clipboard"