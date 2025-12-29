---
date: 2025-12-28
status: approved
bean: beans-t0tv
---

# TUI Two-Column Layout Design

## Overview

Add a two-column layout to the TUI: bean list on the left, read-only detail preview on the right. Cursor movement updates the preview. Enter opens full-screen detail view for interaction.

## Key Decisions

1. **No hierarchy drilling** - list stays flat with tree structure, filtering handles focus
2. **Cursor updates preview** - moving through list immediately shows bean details
3. **Read-only right pane** - no focus, no shortcuts, just visual preview
4. **Enter for full detail** - opens existing full-screen detail view with all features
5. **Responsive collapse** - below 120 columns, single-column list (current behavior)
6. **Compact list format** - single-character type/status codes everywhere

## Layout

**Two-column mode (≥120 columns):**
```
┌─────────────────────────────────┬──────────────────────────────────────────┐
│ Beans                           │ beans-t0tv                               │
│                                 │ Refactor TUI to two-column layout        │
│ ▌ beans-t0tv  F T Refactor TUI  │──────────────────────────────────────────│
│   beans-f11p  E T TUI Improve.. │ Status: todo    Type: feature            │
│   beans-govy  F T Add Y shortc. │ Parent: beans-f11p                       │
│                                 │──────────────────────────────────────────│
│                                 │ ## Summary                               │
│                                 │ Refactor the TUI to a two-column format  │
│                                 │ ...                                      │
├─────────────────────────────────┴──────────────────────────────────────────┤
│ enter view · e edit · space select · ? help                                │
└────────────────────────────────────────────────────────────────────────────┘
```

**Single-column mode (<120 columns):** Current list behavior, unchanged.

**Dimensions:**
- Left pane: fixed 55 characters
- Right pane: remaining width minus borders
- Threshold: 120 columns for two-column mode

## Compact List Format

Single-character codes for type and status columns:

**Types:**
- M = milestone
- E = epic
- B = bug
- F = feature
- T = task

**Statuses:**
- D = draft
- T = todo
- I = in-progress
- C = completed
- S = scrapped

Applied everywhere (not just two-column mode) for consistency.

## Navigation

**In two-column mode:**
- `j/k`, arrows - move cursor, preview updates automatically
- `enter` - open full-screen detail view
- `space` - toggle multi-select
- `p/s/t/P/b/e/y/c` - existing shortcuts work on highlighted bean
- `g t` - tag filter
- `/` - text filter
- `?` - help overlay
- `esc` - clear selection, then clear filter

**In full-screen detail (unchanged):**
- `tab` - switch focus between links and body
- `j/k` - scroll body
- `enter` - navigate to linked bean
- All existing shortcuts
- `esc` - back to two-column view

## Implementation

### State Changes

No new fields in `App` struct. Reuse existing:
- `list` (listModel) for left pane
- Create lightweight detail preview from highlighted bean
- `width/height` for responsive behavior

### Cursor Sync

Detect cursor change in list Update():
```go
previousIndex := m.list.Index()
m.list, cmd = m.list.Update(msg)
if m.list.Index() != previousIndex {
    return m, tea.Batch(cmd, cursorChangedMsg{beanID: item.bean.ID})
}
```

App handles `cursorChangedMsg` to update detail preview.

### View Rendering

```go
func (a *App) View() string {
    if a.state == viewList && a.width >= 120 {
        left := a.list.ViewCompact(55)
        right := a.renderDetailPreview(a.width - 55 - 3)
        return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
    }
    // Existing behavior for other cases
}
```

### Files to Modify

- `internal/tui/tui.go` - View() composition, cursor change handling
- `internal/tui/list.go` - ViewCompact(), compact type/status rendering, cursor change detection
- `internal/tui/detail.go` - extract preview rendering (or create new lightweight preview)
- `internal/ui/styles.go` - single-char type/status formatting helpers

## Edge Cases

- **Empty list:** right pane shows "No bean selected"
- **Terminal resize:** automatic switch between one/two column
- **Long body:** truncated in preview, scroll in full-screen detail
- **Bean deleted:** list reloads, cursor adjusts, preview updates
- **Multi-select:** preview shows cursor's bean (not summary)
- **Links in preview:** shown but non-interactive

## Out of Scope (YAGNI)

- Hierarchy drilling (Enter to show only children)
- Configurable pane widths
- Keyboard focus on right pane
- Breadcrumb navigation
