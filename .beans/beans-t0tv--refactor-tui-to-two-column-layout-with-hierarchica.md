---
# beans-t0tv
title: Refactor TUI to two-column layout with hierarchical navigation
status: in-progress
type: feature
priority: normal
created_at: 2025-12-14T15:37:22Z
updated_at: 2025-12-28T19:20:20Z
parent: beans-f11p
---

## Summary

Refactor the TUI to a two-column format:

- **Left pane**: List of beans (filterable, navigable)
- **Right pane**: Detail view of the currently highlighted bean

Navigation allows drilling down into bean hierarchies:
- Press Enter on a bean to make it the new "root" - the left pane then shows only that bean's children (and their descendants)
- Some key (Escape? Backspace?) to navigate back up the hierarchy

## Motivation

The current single-list view doesn't provide enough context about individual beans without opening an editor. A two-column layout allows:
- Quick scanning of bean details without leaving the list
- Better understanding of bean hierarchies
- More efficient triage and review workflows

## Requirements

### Layout
- Left pane: Bean list (similar to current view)
- Right pane: Full bean details (title, status, type, priority, tags, body, relationships)
- Responsive sizing (handle terminal width gracefully)

### Navigation
- Up/Down: Move cursor in the list
- Enter: Drill into selected bean (make it the root, show children)
- Escape/Backspace: Navigate back up to parent scope
- Show breadcrumb or indicator of current hierarchy position

### Preserved Functionality
- All filtering (by status, type, priority, tags)
- Status changes (keyboard shortcuts)
- Batch selection and editing
- Opening bean in editor

## Subtasks

- beans-3f64: Phase 1 - Compact list format (single-char type/status)
- beans-433o: Phase 2 - Detail preview component
- beans-41ly: Phase 3 - Two-column layout composition
- beans-pri5: Phase 4 - Cursor sync
- beans-6x50: Phase 5 - Integration and polish

## Implementation Plan

See `_spec/plans/2025-12-28-tui-two-column-layout.md` for detailed implementation steps.

## Research

- [[2025-12-28-beans-t0tv-tui-two-column-layout]] - Codebase research documenting the current TUI implementation and architecture considerations for two-column layout

## Design

### Key Decisions

1. **No hierarchy drilling** - list stays flat with tree structure, filtering handles focus
2. **Cursor updates preview** - moving through list immediately shows bean details
3. **Read-only right pane** - no focus, no shortcuts, just visual preview
4. **Enter for full detail** - opens existing full-screen detail view with all features
5. **Responsive collapse** - below 120 columns, single-column list (current behavior)
6. **Compact list format** - single-character type/status codes everywhere

### Layout

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

### Compact List Format

Single-character codes for type and status columns:

**Types:** M(ilestone), E(pic), B(ug), F(eature), T(ask)

**Statuses:** D(raft), T(odo), I(n-progress), C(ompleted), S(crapped)

Applied everywhere (not just two-column mode) for consistency.

### Navigation

**In two-column mode:**
- `j/k`, arrows - move cursor, preview updates automatically
- `enter` - open full-screen detail view
- `space` - toggle multi-select
- `p/s/t/P/b/e/y/c` - existing shortcuts work on highlighted bean
- `g t` - tag filter, `/` - text filter, `?` - help overlay
- `esc` - clear selection, then clear filter

**In full-screen detail (unchanged):**
- `tab` - switch focus between links and body
- `j/k` - scroll body
- `enter` - navigate to linked bean
- `esc` - back to two-column view

### Implementation

**Cursor sync:** Detect cursor change in list Update(), emit `cursorChangedMsg`. App handles it to update detail preview.

**View rendering:** In `View()`, if width ≥120, compose left (list) + right (preview) with `lipgloss.JoinHorizontal`.

**Files to modify:**
- `internal/tui/tui.go` - View() composition, cursor change handling
- `internal/tui/list.go` - ViewCompact(), compact type/status, cursor change detection
- `internal/tui/detail.go` - extract preview rendering
- `internal/ui/styles.go` - single-char type/status formatting helpers

### Edge Cases

- **Empty list:** right pane shows "No bean selected"
- **Terminal resize:** automatic switch between one/two column
- **Long body:** truncated in preview, scroll in full-screen detail
- **Bean deleted:** list reloads, cursor adjusts, preview updates
- **Multi-select:** preview shows cursor's bean (not summary)
- **Links in preview:** shown but non-interactive

### Out of Scope (YAGNI)

- Hierarchy drilling (Enter to show only children)
- Configurable pane widths
- Keyboard focus on right pane
- Breadcrumb navigation