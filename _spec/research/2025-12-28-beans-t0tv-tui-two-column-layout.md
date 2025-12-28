---
date: 2025-12-28T12:00:00-08:00
researcher: Claude
git_commit: e628ab1d4bd5f6066d76b871f4b41bda118c9c7e
branch: main
repository: beans
topic: "TUI Two-Column Layout Implementation Research"
tags: [research, tui, bubbletea, beans-t0tv]
status: complete
last_updated: 2025-12-28
last_updated_by: Claude
---

# Research: TUI Two-Column Layout Implementation

**Date**: 2025-12-28
**Researcher**: Claude
**Git Commit**: e628ab1d4bd5f6066d76b871f4b41bda118c9c7e
**Branch**: main
**Repository**: beans
**Related Bean**: beans-t0tv

## Research Question

What are the relevant parts of the codebase for implementing the two-column TUI layout with hierarchical navigation (bean t0tv)?

## Summary

The TUI is a Bubbletea-based application in `internal/tui/` with a state machine architecture. The current implementation already has separate list and detail models, which can be adapted for a two-column layout. Key components to modify include the main `App` model (view composition), `listModel` (left pane), and `detailModel` (right pane). The existing navigation, filtering, batch selection, and editor integration can be preserved with minimal changes.

## Detailed Findings

### TUI Architecture Overview

The TUI uses Bubbletea's Model-Update-View pattern with a main `App` struct that coordinates between multiple view states and sub-models.

**Main Entry Point**: `internal/tui/tui.go`

#### App Model Structure (`tui.go:75-104`)

```go
type App struct {
    state         viewState        // Current view (list, detail, picker modals)
    list          listModel        // List view model
    detail        detailModel      // Detail view model
    // ... picker models ...
    history       []detailModel    // Stack for back navigation
    core          *beancore.Core   // Bean data management
    resolver      *graph.Resolver  // GraphQL queries/mutations
    width, height int              // Terminal dimensions
    previousState viewState        // For modal backgrounds
}
```

#### View States (`tui.go:21-34`)

Currently 10 view states:
- `viewList` - Main bean list
- `viewDetail` - Single bean detail (full screen)
- `viewTagPicker`, `viewParentPicker`, `viewStatusPicker`, `viewTypePicker`, `viewPriorityPicker`, `viewBlockingPicker` - Modal pickers
- `viewCreateModal`, `viewHelpOverlay` - Other modals

**Key Insight**: The existing `viewList` and `viewDetail` states are separate full-screen views. For two-column layout, these would become panes rendered side-by-side in a single view.

### List Model (`internal/tui/list.go`)

#### Structure (`list.go:103-124`)

```go
type listModel struct {
    list          list.Model       // Bubbletea list component
    resolver      *graph.Resolver
    config        *config.Config
    width, height int
    hasTags       bool
    cols          ui.ResponsiveColumns  // Calculated column widths
    idColWidth    int                   // ID column width for tree depth
    tagFilter     string                // Active tag filter
    selectedBeans map[string]bool       // Multi-select state
    statusMessage string
}
```

#### Key Methods

- `newListModel()` (`list.go:126-146`) - Creates list with custom delegate
- `Init()` (`list.go:164-165`) - Returns `loadBeans` command
- `loadBeans()` (`list.go:168-211`) - GraphQL query, tree building, flattening
- `Update()` (`list.go:228-451`) - Key handling, window sizing
- `View()` (`list.go:465-550`) - Renders list with border and footer

#### Current Rendering

The list renders beans in a tree structure with:
- Tree prefixes (box-drawing characters)
- Responsive columns: ID, Type, Status, Title, optional Tags
- Cursor highlighting (purple "▌")
- Multi-select highlighting (amber ID)
- Dimmed ancestors for context

**Relevant for Two-Column**: List rendering already handles variable widths via `ui.ResponsiveColumns`. The width can be constrained to left pane width.

### Detail Model (`internal/tui/detail.go`)

#### Structure (`detail.go:128-141`)

```go
type detailModel struct {
    viewport      viewport.Model   // Scrollable body
    bean          *bean.Bean
    resolver      *graph.Resolver
    config        *config.Config
    width, height int
    ready         bool
    links         []resolvedLink   // Parent, children, blocking relationships
    linkList      list.Model       // Filterable link list
    linksActive   bool             // Focus: links vs body
    cols          ui.ResponsiveColumns
    statusMessage string
}
```

#### Sections Rendered

1. **Header** (`detail.go:491-527`) - Title, ID, status badge, tags
2. **Links Section** (`detail.go:419-431`) - Linked beans with focus border
3. **Body** (`detail.go:667-686`) - Markdown-rendered description in viewport

**Relevant for Two-Column**: Detail view already handles variable sizing. Can be adapted to right pane width. The links section and viewport scrolling work independently.

### Navigation and Hierarchy

#### Current Navigation Flow

1. **List → Detail**: `enter` key sends `selectBeanMsg` (`list.go:281-286`)
2. **Detail → List**: `esc`/`backspace` sends `backToListMsg` (`detail.go:298-301`)
3. **Detail → Detail**: Navigating to linked bean pushes to history (`tui.go:502-509`)
4. **Back Navigation**: Pops from history stack (`tui.go:511-523`)

#### History Stack (`tui.go:87`)

```go
history []detailModel  // Stack of previous detail views
```

**Key Insight for Hierarchical Navigation**: The existing history stack pattern can be adapted. Instead of storing detail views, store the "root" bean ID for hierarchy drilling.

### Filtering System

#### Tag Filtering (`list.go:117`, `tui.go:200-214`)

- Tag filter stored in `listModel.tagFilter`
- Applied at GraphQL query level via `BeanFilter{Tags: []string{tag}}`
- "g t" chord opens tag picker
- Tree building includes ancestors for context (dimmed)

#### Text Filtering

- Built into Bubbletea's list component
- Activated with "/" key
- Filters on `bean.Title + " " + bean.ID`

**Relevant for Two-Column**: Filtering remains on the left pane (list). The right pane shows detail of highlighted bean.

### Keybindings

#### List View Keys (`list.go:269-444`)

| Key | Action |
|-----|--------|
| `space` | Toggle multi-select, move down |
| `enter` | View bean detail |
| `p` | Open parent picker |
| `s` | Open status picker |
| `t` | Open type picker |
| `P` | Open priority picker |
| `b` | Open blocking picker |
| `c` | Create new bean |
| `e` | Edit in external editor |
| `y` | Copy bean ID(s) |
| `esc`/`backspace` | Clear selection, then filter |

#### Detail View Keys (`detail.go:297-386`)

| Key | Action |
|-----|--------|
| `tab` | Toggle focus: links ↔ body |
| `enter` | Navigate to linked bean |
| `p/s/t/P/b/e/y` | Same as list view |
| `esc`/`backspace` | Back to list/previous |

**Relevant for Two-Column**: Most keys work on highlighted bean. In two-column, highlight on left pane determines what's shown on right. Same keys should work.

### Batch Selection (`list.go:120-121`, `tui.go:246-334`)

- `selectedBeans map[string]bool` tracks selected IDs
- Visual: Amber highlight on ID column
- Operations: status, type, priority, parent changes
- Footer shows selection count

**Relevant for Two-Column**: Selection remains on left pane. Batch operations unaffected.

### Editor Integration (`tui.go:414-450`)

1. `e` key triggers `openEditorMsg`
2. Records bean ID and file mod time
3. `tea.ExecProcess()` suspends TUI, launches editor
4. On return, checks if file modified, updates `updated_at`
5. File watcher triggers `beansChangedMsg` for refresh

**Relevant for Two-Column**: Works the same. Editor opens for highlighted bean.

### Window Sizing (`tui.go:128-130`, `list.go:232-239`)

- `tea.WindowSizeMsg` received on resize
- `App` stores `width`, `height`
- List/detail models update their dimensions
- Responsive columns recalculated

**Critical for Two-Column**: Need to split width between panes. Consider:
- Fixed ratio (e.g., 40/60)
- Minimum widths for each pane
- Collapse to single pane on narrow terminals

### Shared UI Components (`internal/ui/`)

#### Tree Rendering (`ui/tree.go`)

- `BuildTree()` - Creates hierarchy from beans
- `FlattenTree()` - Converts to flat list with prefixes
- `MaxTreeDepth()` - For ID column width

#### Responsive Columns (`ui/styles.go:350-407`)

- `CalculateResponsiveColumns()` - Computes column widths
- Tags shown only when width >= 140
- Tag column scales 24-70 chars based on space

#### Bean Row Rendering (`ui/styles.go:409-543`)

- `RenderBeanRow()` - Shared between list and detail links
- Handles cursor, selection, dimming, tree prefix

## Code References

### Core Files to Modify

- `internal/tui/tui.go:75-104` - App model needs new layout state
- `internal/tui/tui.go:572-608` - View() needs two-column composition
- `internal/tui/list.go:465-550` - List View() needs width constraint
- `internal/tui/detail.go:411-471` - Detail View() needs width constraint

### Supporting Files

- `internal/tui/keys.go` - May need new keybindings for hierarchy navigation
- `internal/ui/styles.go:350-407` - Responsive column calculation
- `internal/tui/help.go` - Update help text with new keybindings

## Architecture Considerations

### Proposed Two-Column Structure

```go
type App struct {
    // ... existing fields ...

    // New fields for two-column
    rootBeanID    string         // Current hierarchy root (empty = top level)
    rootHistory   []string       // Stack of previous roots for back navigation
    leftPaneWidth int            // Calculated left pane width
}
```

### View Composition Pattern

The current `View()` switches between full-screen views. For two-column:

```go
func (a *App) View() string {
    if a.state == viewList {
        // Two-column layout
        left := a.renderLeftPane()   // Constrained list
        right := a.renderRightPane() // Constrained detail
        return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
    }
    // Modal overlays work as before
    return a.renderModalOverlay()
}
```

### Hierarchy Navigation

- **Enter on bean**: Set as new root, list shows only children
- **Escape/Backspace**: Pop from root history, show parent's children
- **Breadcrumb**: Show path like "Root > Epic > Feature"

### Width Handling

Suggested approach:
- Minimum left pane: 50 chars (enough for compact tree)
- Minimum right pane: 60 chars (readable detail)
- If terminal < 110 chars: Fall back to single pane (existing behavior)
- Otherwise: 40% left, 60% right (or configurable)

## Open Questions

1. **Hierarchy root indicator**: Where to show breadcrumb? Above list? In list title?
2. **Empty children**: What to show when a bean has no children?
3. **Back navigation key**: Use `esc` (conflicts with filter clear) or `backspace`?
4. **Detail pane focus**: Should right pane be scrollable with j/k when focused?
5. **Responsive breakpoint**: At what terminal width to collapse to single column?

## Related Documentation

- `.claude/skills/bubbletea/SKILL.md` - Bubbletea framework reference
- Bean t0tv checklist in `.beans/beans-t0tv--*.md`
