---
date: 2025-12-29T12:00:00+01:00
researcher: Claude
git_commit: dc553baba6568b964f5b65e48def0afd1bbedb11
branch: main
repository: beans
topic: "TUI Filtering Feature - Relevant Codebase Components"
tags: [research, tui, filtering, beans, feature-request]
status: complete
last_updated: 2025-12-29
last_updated_by: Claude
---

# Research: TUI Filtering Feature - Relevant Codebase Components

**Date**: 2025-12-29
**Researcher**: Claude
**Git Commit**: dc553baba6568b964f5b65e48def0afd1bbedb11
**Branch**: main
**Repository**: beans

## Research Question

Identify relevant parts of the codebase for implementing TUI filtering. Feature request: when there are many beans (especially closed/scrapped), users can't easily see relevant/open ones. The TUI should offer a way to filter beans (show only relevant/open ones, or more general filtering).

## Summary

The TUI currently **only supports tag filtering**, while the CLI has comprehensive filtering capabilities (status, type, priority, tags, parent, blocking, search). To implement TUI filtering, the key approach is to extend the `listModel` to use the existing `BeanFilter` infrastructure that already powers the CLI.

**Key insight**: The filtering backend is already complete. The work is purely in the TUI layer.

## Detailed Findings

### Current TUI Filter State

The TUI's `listModel` currently has a single filter field:

**File**: `internal/tui/list.go:117-118`
```go
type listModel struct {
    // ...
    tagFilter string // if set, only show beans with this tag
    // ...
}
```

This is applied when loading beans:

**File**: `internal/tui/list.go:168-180`
```go
func (m listModel) loadBeans() tea.Msg {
    var filter *model.BeanFilter
    if m.tagFilter != "" {
        filter = &model.BeanFilter{Tags: []string{m.tagFilter}}
    }
    filteredBeans, err := m.resolver.Query().Beans(context.Background(), filter)
    // ...
}
```

### Existing Filter Infrastructure (CLI)

The CLI already has comprehensive filtering via the `beans list` command:

**File**: `cmd/list.go:19-40` - Filter variables
```go
var (
    listSearch     string
    listStatus     []string
    listNoStatus   []string
    listType       []string
    listNoType     []string
    listPriority   []string
    listNoPriority []string
    listTag        []string
    listNoTag      []string
    listHasParent  bool
    listNoParent   bool
    listParentID   string
    listHasBlocking bool
    listNoBlocking  bool
    listIsBlocked   bool
    listReady       bool
)
```

**File**: `cmd/list.go:62-109` - Filter construction from CLI flags

### GraphQL Filter Schema

The `BeanFilter` model supports all filter types:

**File**: `internal/graph/model/models_gen.go:6-51`
```go
type BeanFilter struct {
    Search          *string  // Full-text search
    Status          []string // Include by status (OR)
    ExcludeStatus   []string // Exclude by status
    Type            []string // Include by type (OR)
    ExcludeType     []string // Exclude by type
    Priority        []string // Include by priority (OR)
    ExcludePriority []string // Exclude by priority
    Tags            []string // Include by tags (OR)
    ExcludeTags     []string // Exclude by tags
    HasParent       *bool
    NoParent        *bool
    ParentID        *string
    HasBlocking     *bool
    NoBlocking      *bool
    BlockingID      *string
    IsBlocked       *bool
}
```

### Filter Application Logic

**File**: `internal/graph/filters.go:9-80`
- `ApplyFilter()` applies all filter types sequentially to a bean slice
- Already used by GraphQL resolver for `Beans()` query
- TUI can leverage this directly by passing a populated `BeanFilter`

### TUI Key Handling

Current tag filter workflow:

**File**: `internal/tui/tui.go:162-185` - Key chord handling (`gt` opens tag picker)
**File**: `internal/tui/tui.go:248-262` - Tag filter messages

### Existing Modal Pickers

The TUI already has modal pickers for status, type, priority that could be repurposed for filtering:

- `internal/tui/statuspicker.go` - Status selection modal
- `internal/tui/typepicker.go` - Type selection modal
- `internal/tui/prioritypicker.go` - Priority selection modal
- `internal/tui/tagpicker.go` - Tag selection modal

These currently mutate beans but could inspire filter picker UI.

## Code References

### TUI Core Files

| File | Purpose |
|------|---------|
| `internal/tui/tui.go` | Main app, message routing, key handling |
| `internal/tui/list.go` | List view model, `loadBeans()`, current filter state |
| `internal/tui/styles.go` | TUI-specific styles |

### Filter Infrastructure

| File | Purpose |
|------|---------|
| `internal/graph/filters.go` | `ApplyFilter()` - core filter logic |
| `internal/graph/model/models_gen.go` | `BeanFilter` struct |
| `internal/graph/schema.graphqls:136-183` | GraphQL filter schema |
| `internal/graph/schema.resolvers.go:262-277` | `Beans()` resolver applying filters |

### CLI Reference Implementation

| File | Purpose |
|------|---------|
| `cmd/list.go` | CLI filter flags and construction |

### Modal Pickers (UI patterns to follow)

| File | Purpose |
|------|---------|
| `internal/tui/tagpicker.go` | Tag picker modal (already used for filtering) |
| `internal/tui/statuspicker.go` | Status picker modal (pattern reference) |
| `internal/tui/modal.go` | Modal rendering utilities |

## Architecture Documentation

### Current Data Flow

```
TUI list view
    ↓
loadBeans() with BeanFilter{Tags: [tagFilter]} (only tags currently)
    ↓
GraphQL resolver.Query().Beans(ctx, filter)
    ↓
ApplyFilter(beans, filter, core)
    ↓
Filtered beans returned
    ↓
BuildTree() + FlattenTree()
    ↓
Display with dimmed ancestors for context
```

### Key Extension Points

1. **`listModel` struct** (`internal/tui/list.go:102-124`)
   - Replace `tagFilter string` with `filter *model.BeanFilter`
   - Or add individual filter fields for status, type, etc.

2. **`loadBeans()` method** (`internal/tui/list.go:168-211`)
   - Already accepts `BeanFilter`, just needs populated fields

3. **Key bindings** (`internal/tui/tui.go:162-185`, `internal/tui/list.go:228-472`)
   - Add new key chords for filter pickers (e.g., `gs` for status filter, `gp` for priority)

4. **Filter pickers** (`internal/tui/tagpicker.go` as template)
   - Create status/type/priority filter pickers (or repurpose existing)

5. **List title** (`internal/tui/list.go:530-535`)
   - Already shows tag filter, extend for other filters

## Historical Context

No existing specs specifically address TUI filtering. Related documents:

- `_spec/research/2025-12-28-beans-t0tv-tui-two-column-layout.md` - Recent TUI improvements
- `_spec/plans/2025-12-28-tui-two-column-layout.md` - TUI implementation patterns

## Related Research

None found specifically for filtering features.

## Open Questions

1. **UI approach**: Should there be a single "filter panel" or individual pickers per field?
2. **Preset filters**: Should there be quick filters like "Open beans" (exclude completed/scrapped)?
3. **Filter persistence**: Should filter state persist across TUI sessions?
4. **Search integration**: Should the built-in `/` search filter visually, or should there be a "search mode" using Bleve?
5. **Filter indicators**: How to show active filters (title bar? footer? sidebar?)?
