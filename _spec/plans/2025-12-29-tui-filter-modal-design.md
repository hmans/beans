---
date: 2025-12-29
author: Stefan + Claude
status: approved
topic: TUI Filter Modal Design
tags: [design, tui, filtering]
---

# TUI Filter Modal Design

## Overview

Add a unified filter modal to the TUI that allows filtering beans by status, type, and tags. The modal provides quick keyboard-driven toggling with visual feedback.

## Trigger

Press `g` to open the filter modal from the list view.

## Modal Layout

```
┌─────────────────────────────────────────────────────────────────┐
│                        Filter Beans                             │
│                                                                 │
│  STATUS              TYPE              TAGS                     │
│                                                                 │
│  [1] draft           [m] milestone     [A] backend              │
│  [2] todo        ●   [e] epic          [B] blocked              │
│  [3] in-progress ●   [f] feature   ●   [C] frontend         ●   │
│  [4] completed       [b] bug       ●   [D] tech-debt            │
│  [5] scrapped        [t] task          [E] urgent           ●   │
│                                                                 │
│  [x] reset all                              [enter] apply       │
└─────────────────────────────────────────────────────────────────┘
```

### Visual Indicators

- Active filters shown with `●` dot AND bold text
- Status/type labels use their configured colors (same as in list view)
- Inactive items are dimmed
- Key bindings shown in brackets and highlighted

### Column Visibility

- Tags column is hidden if the project has no tags
- Two-column layout (Status | Type) when no tags exist

## Key Bindings

| Key | Action |
|-----|--------|
| `1` | Toggle draft |
| `2` | Toggle todo |
| `3` | Toggle in-progress |
| `4` | Toggle completed |
| `5` | Toggle scrapped |
| `m` | Toggle milestone |
| `e` | Toggle epic |
| `f` | Toggle feature |
| `b` | Toggle bug |
| `t` | Toggle task |
| `A-Z` | Toggle tag (alphabetically assigned) |
| `x` | Reset all filters |
| `Enter` | Apply filters and close modal |
| `Esc` | Close without applying |

## Filter Logic

### Within a dimension (OR)

Multiple selections within status, type, or tags use OR logic:

- `[todo] + [in-progress]` → shows beans that are todo OR in-progress

### Across dimensions (AND)

Different dimensions combine with AND logic:

- `[todo] + [feature]` → shows beans that are todo AND feature type

### Empty selection

Nothing selected in a dimension means no filtering on that dimension:

- No statuses selected → show all statuses
- No types selected → show all types
- No tags selected → show all tags (no tag filtering)

## Filter Bar

A persistent filter bar appears at the bottom of the list view, above the help shortcuts.

### Format

```
draft [todo] [in-progress] completed scrapped │ milestone epic [feature] [bug] task
```

Order is always fixed (matches modal: statuses 1-5, types m-e-f-b-t). Active items shown in bold with color.

### Styling

- Active filters: bold text with configured colors
- Inactive filters: dimmed text
- Filter bar always visible (all dimmed when no filters active)
- Separator `│` between status and type sections
- Tags shown after types if any are active: `│ [frontend] [urgent]`

### Location

```
┌─ Beans ──────────────────────────────────────────────────────────────────────┐
│   ID       S  T  Title                                                       │
│ ▌ bean-abc T  F  Implement user authentication                               │
│   bean-def I  M  v2.0 Release                                                │
│     └─ bean-ghi T  F  Add dark mode support                                  │
│                                                                              │
├──────────────────────────────────────────────────────────────────────────────┤
│ draft [todo] [in-progress] completed scrapped │ milestone epic [feature] [bug] task │
└──────────────────────────────────────────────────────────────── g filter  ? help  q quit ┘
```

Placed above the shortcuts bar. Should span the entire screen like the shortcut bar.

## Tag Handling

- Tags sorted alphabetically
- Assigned to keys A-Z (first 26 tags)
- If more than 26 tags exist, show the first 26 alphabetically
- Tags column hidden entirely if project has no tags

## State Management

### Filter State

Add to `listModel`:

```go
type listModel struct {
    // ... existing fields ...

    // Filter state (replaces tagFilter string)
    statusFilter []string  // active status filters
    typeFilter   []string  // active type filters
    tagFilters   []string  // active tag filters (multi-select)
}
```

### Building BeanFilter

When loading beans, construct filter from state:

```go
filter := &model.BeanFilter{
    Status: m.statusFilter,  // empty = no filtering
    Type:   m.typeFilter,    // empty = no filtering
    Tags:   m.tagFilters,    // empty = no filtering
}
```

## Implementation Notes

### Files to Modify

| File | Changes |
|------|---------|
| `internal/tui/list.go` | Add filter state fields, update `loadBeans()`, remove `tagFilter` |
| `internal/tui/tui.go` | Add `g` key handler, filter modal view state, remove `gt` chord handling |
| `internal/tui/filterpicker.go` | New file: filter modal model and view |
| `internal/tui/tagpicker.go` | Remove (no longer needed for filtering) |

### Breaking Changes

- Remove `gt` key chord for tag filtering (replaced by unified filter modal)
- Remove `tagFilter string` field from `listModel` (replaced by `tagFilters []string`)

### Reuse Existing Infrastructure

- `model.BeanFilter` already supports all filter types
- `ApplyFilter()` in `internal/graph/filters.go` handles the logic
- Modal patterns from `internal/tui/modal.go` for overlay rendering
- Color configuration from `config.GetBeanColors()`

## Open Items

- Filter persistence across TUI sessions (future enhancement)
- Keyboard shortcut shown in help overlay
