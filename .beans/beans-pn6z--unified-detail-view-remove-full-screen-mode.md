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

### View States

Replace the current `viewDetail` state with granular focus states:

```go
const (
    viewListFocused viewState = iota
    viewDetailLinksFocused
    viewDetailBodyFocused
    viewTagPicker
    viewParentPicker
    // ... other pickers unchanged
)
```

Single source of truth - viewState tells you exactly what's focused.

### Interaction Model

**Key bindings:**

| Key | List Focused | Links Focused | Body Focused |
|-----|--------------|---------------|--------------|
| `enter` | Focus detail (links if present, else body) | Follow link | - |
| `backspace` | - | Navigate history, then focus list | Navigate history, then focus list |
| `tab` | - | Switch to body | Switch to links |
| `esc` | Clear selection/filter | - | - |
| `j/k` | Navigate list | Navigate links | Scroll body |
| `/` | Filter list | Filter links | - |
| `space` | Toggle select | - | - |
| `c` | Create bean | - | - |
| `p,s,t,P,b,e,y` | Edit shortcuts | Edit shortcuts | Edit shortcuts |
| `?` | Help | Help | Help |
| `q` | Quit | Quit | Quit |

**Notes:**
- Only `q` quits the app. `esc` is for cancel/clear only.
- `backspace` means "go back" - navigates history first, then returns to list when empty.
- `esc` clears selection first, then clears filter (list only).
- Edit shortcuts (p, s, t, P, b, e, y) work from all three focus states.

### History Navigation

When following a link (Enter on a linked bean):
1. Push current bean to history stack
2. Move list cursor to linked bean
3. Detail pane updates automatically (recreated on cursor change)
4. Stay in detail focus

When pressing Backspace in detail:
1. If history not empty → pop from history, move cursor to that bean, stay in detail
2. If history empty → focus list

### Visual Indication

- Border color shows focus: primary (cyan) when focused, muted (gray) when not
- Applied to: list pane border, detail links section border, detail body section border
- Both panes already have borders - just change colors based on focus

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

### Footer

Footer changes based on focused area:

**List focused:**
`space select · enter view · c create · / filter · esc clear · b blocking · e edit · p parent · P priority · s status · t type · y copy id · ? help · q quit`

**Links focused:**
`tab switch · / filter · enter go to · j/k navigate · backspace back · b blocking · e edit · p parent · P priority · s status · t type · y copy id · ? help · q quit`

**Body focused:**
`tab switch · j/k scroll · backspace back · b blocking · e edit · p parent · P priority · s status · t type · y copy id · ? help · q quit`

### Picker/Modal Return

When opening a picker (status, type, parent, etc.):
- Save current viewState to previousState
- On close, restore previousState

This works naturally with granular view states - if you opened from `viewDetailLinksFocused`, you return to `viewDetailLinksFocused`.

### Detail Model Updates

- Recreate `detailModel` on every cursor change (same as current preview behavior)
- No need to preserve scroll position since it's a different bean
- Links section focus resets, which makes sense for a new bean

### Implementation Approach

1. Delete `preview.go` - no longer needed
2. Replace `viewDetail` with `viewListFocused`, `viewDetailLinksFocused`, `viewDetailBodyFocused`
3. Move `linksActive` logic from detailModel to App level (viewState handles it)
4. Always use `detailModel` in right pane, recreate on cursor change
5. Update keyboard routing based on viewState
6. Update border colors based on viewState
7. Update footer based on viewState
8. Keep history stack, update Backspace to navigate it first

### Edge Cases

- Empty list: right pane shows "No bean selected", Enter does nothing
- Terminal resize while detail focused: if now wide, show both panes
- Link navigation: move list cursor, recreate detail, stay in detail focus
- No links on bean: Tab does nothing, Enter from list focuses body directly

### Files to Modify

- `internal/tui/tui.go` - view states, routing, view composition, history
- `internal/tui/detail.go` - remove linksActive (handled by viewState), accept focus prop for borders
- `internal/tui/list.go` - accept focus prop for border color
- `internal/tui/preview.go` - delete

## Design Rationale

**Why granular view states instead of a `detailFocused` bool?**
We considered using `viewList` + `detailFocused bool`, but this creates a problem with picker return. When opening a picker, we save `previousState`. With a bool, we'd need to save/restore both viewState AND the bool separately. Granular states (`viewDetailLinksFocused`) capture everything in one place - picker return just restores the single viewState.

**Why Backspace for navigation, Esc for cancel/clear?**
Gives each key a consistent meaning: Backspace = "go back" (navigation), Esc = "cancel/clear" (selection, filter, modal). Mixing them would be confusing - e.g., sometimes Esc navigates, sometimes it clears.

**Why keep the history stack?**
Following a blocking relationship can jump to a bean far away in the list. Without history, you'd lose your place and have to manually scroll back. History lets you retrace your steps through linked beans.

**Why keep all linked beans (parent, children, blocking, blocked-by)?**
Parent/children are visible in the list tree, so showing them in detail is redundant in wide mode. But in narrow mode, you can only see one pane - linked beans is the only way to see/navigate the hierarchy. Keeping it consistent across modes is simpler than conditional display.

**Why only `q` quits?**
Esc already does multiple things (clear selection, clear filter, close modals). Adding "quit" to that list makes it unclear when Esc will quit vs do something else. Single quit key (`q`) is predictable.

**Why keep edit shortcuts in detail view?**
You often want to change status/type/priority while looking at the full details. Forcing users to Backspace to list first adds friction. Same shortcuts in both views = less to remember.

**Why border color for focus indication?**
Simplest option that works. Both panes already have borders. Alternatives (title highlighting, background tint) add complexity for marginal benefit.

## Implementation Tasks

1. **beans-mvme**: Update view states to granular focus states
2. **beans-238n**: Delete preview.go and update App struct
3. **beans-2v6j**: Add focus parameter to list border rendering
4. **beans-ozhq**: Update detail.go to accept focus parameters
5. **beans-lmk4**: Update keyboard routing for granular focus states
6. **beans-unjz**: Update View() for two-column rendering with focus
7. **beans-7vrp**: Update history navigation for unified view
8. **beans-p63f**: Remove Esc as quit, clean up keyboard handling
9. **beans-oms6**: Integrate detail model with App keyboard routing
10. **beans-csnk**: Testing and polish

## Out of Scope

- Shift+Tab for reverse cycling
- Drill-down navigation (filter to children)
- Top/bottom layout alternative
- Configurable pane widths