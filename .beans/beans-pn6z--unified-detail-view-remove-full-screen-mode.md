---
# beans-pn6z
title: Unified detail view (remove full-screen mode)
status: todo
type: feature
priority: normal
created_at: 2025-12-30T14:02:35Z
updated_at: 2025-12-30T14:17:30Z
parent: beans-t0tv
---

## Summary

Remove the separate full-screen detail view. Instead, the detail pane is always the right side of the two-column layout, with responsive behavior based on terminal width.

## Motivation

The two-column layout already shows bean details on the right. Having a separate full-screen detail view is redundant. Unifying these simplifies the mental model and reduces code complexity.

## Design

### Interaction Model

- **Enter** (from list): Focus right pane (links section if present, else body)
- **Backspace** (from right pane): Return focus to list
- **Tab** (when detail focused): Toggle between links↔body within detail pane
- **j/k**: Navigate within focused area (list items, links, or scroll body)

### Visual Indication

- Border color only: primary (cyan) when focused, muted (gray) when not
- Applied to: left pane border, right pane links section border, right pane body border

### Layout Behavior

**Wide terminal (≥120 columns):**
- Both panes visible simultaneously
- Focus determines which pane receives keyboard input
- Unfocused pane still visible but non-interactive

**Narrow terminal (<120 columns):**
- Only one pane visible at a time
- Default: list pane visible (detail width = 0)
- Enter: list hidden, detail visible (list width = 0)
- Backspace: detail hidden, list visible

### Implementation Approach

1. Delete `preview.go` - no longer needed
2. Always use `detailModel` in right pane
3. Add `detailFocused bool` to `App` struct to track focus state
4. In `Update()`:
   - Enter (when list focused): set `detailFocused = true`
   - Backspace (when detail focused): set `detailFocused = false`
   - Route keyboard events based on `detailFocused`
5. In `View()`:
   - Wide mode: render both panes, pass focus state for border colors
   - Narrow mode: render only the focused pane at full width
6. Add border to left pane (list) with focus-dependent color
7. Remove `viewDetail` state - detail is always in right pane, not a separate view

### Edge Cases

- Empty list: right pane shows "No bean selected", Enter does nothing
- Terminal resize while detail focused: if now wide, show both panes
- Link navigation (Enter on link): stay in detail focus, update both list cursor and detail content

### Files to Modify

- `internal/tui/tui.go` - focus state, routing, view composition
- `internal/tui/detail.go` - accept focus prop for border styling
- `internal/tui/list.go` - add border, accept focus prop
- `internal/tui/preview.go` - delete

## Out of Scope

- Shift+Tab for reverse cycling
- Drill-down navigation (filter to children)
- Top/bottom layout alternative
- Configurable pane widths