---
# beans-pn6z
title: Unified detail view (remove full-screen mode)
status: completed
type: feature
priority: normal
created_at: 2025-12-30T14:02:35Z
updated_at: 2025-12-30T20:47:27Z
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
| `backspace` | - | Return to list | Return to list |
| `tab` | - | Switch to body | Switch to links |
| `esc` | Clear selection/filter | Navigate history, then return to list | Navigate history, then return to list |
| `j/k` | Navigate list | Navigate links | Scroll body |
| `/` | Filter list | Filter links | - |
| `space` | Toggle select | - | - |
| `c` | Create bean | - | - |
| `p,s,t,P,b,e,y` | Edit shortcuts | Edit shortcuts | Edit shortcuts |
| `?` | Help | Help | Help |
| `q` | Quit | Quit | Quit |

**Notes:**
- Only `q` quits the app.
- `backspace` in detail always returns to list immediately (clears history).
- `esc` in detail navigates history stack; when empty, returns to list.
- `esc` in list clears selection first, then clears filter.
- Edit shortcuts (p, s, t, P, b, e, y) work from all three focus states.

### History Navigation

When following a link (Enter on a linked bean):
1. Push current bean to history stack
2. Move list cursor to linked bean
3. Detail pane updates automatically (recreated on cursor change)
4. Switch to list focus for continued navigation

When pressing Escape in detail:
1. If history not empty → pop from history, move cursor to that bean, stay in detail
2. If history empty → return to list

When pressing Backspace in detail:
- Always return to list immediately (clears history)

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
`tab switch · / filter · enter go to · j/k navigate · esc back · b blocking · e edit · p parent · P priority · s status · t type · y copy id · ? help · q quit`

**Body focused:**
`tab switch · j/k scroll · esc back · b blocking · e edit · p parent · P priority · s status · t type · y copy id · ? help · q quit`

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
8. Keep history stack, Escape navigates history, Backspace returns to list

### Edge Cases

- Empty list: right pane shows "No bean selected", Enter does nothing
- Terminal resize while detail focused: if now wide, show both panes
- Link navigation: move list cursor, recreate detail, switch to list focus
- No links on bean: Tab does nothing, Enter from list focuses body directly

### Files to Modify

- `internal/tui/tui.go` - view states, routing, view composition, history
- `internal/tui/detail.go` - remove linksActive (handled by viewState), accept focus prop for borders
- `internal/tui/list.go` - accept focus prop for border color
- `internal/tui/preview.go` - delete

## Design Rationale

**Why granular view states instead of a `detailFocused` bool?**
We considered using `viewList` + `detailFocused bool`, but this creates a problem with picker return. When opening a picker, we save `previousState`. With a bool, we'd need to save/restore both viewState AND the bool separately. Granular states (`viewDetailLinksFocused`) capture everything in one place - picker return just restores the single viewState.

**Why Escape for history navigation, Backspace for immediate return?**
Backspace provides a quick "hard reset" back to list focus, clearing history. Escape lets you step back through history one bean at a time. This gives users two options: retrace steps (Esc) or return directly to list (Backspace).

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
5. **beans-s65d**: Update message handlers for detailModel (beansLoadedMsg, beansChangedMsg, WindowSizeMsg, empty list)
6. **beans-lmk4**: Update keyboard routing for granular focus states (includes moveCursorToBean helper)
7. **beans-unjz**: Update View() for two-column rendering with focus (includes footer rendering with statusMessage)
8. **beans-7vrp**: Update history navigation for unified view
9. **beans-p63f**: Simplify detail.go and clean up keyboard handling (consolidates former Task 9)
10. **beans-csnk**: Testing and polish

~~**beans-oms6**: Consolidated into beans-p63f~~

## Out of Scope

- Shift+Tab for reverse cycling
- Drill-down navigation (filter to children)
- Top/bottom layout alternative
- Configurable pane widths

## Implementation Notes - Height Alignment Issues

### Problem
The detail pane (right side) was consistently 1-2 lines shorter than the list pane (left side), causing misaligned bottom borders.

### Root Causes Discovered

1. **Bubbles list.View() ignores height with few items**
   - When we set `list.New(items, delegate, width, height=2)` with only 1 item
   - The list renders 3 lines (title + 1 item + spacing) instead of respecting height=2
   - The list only enforces height when there are enough items to paginate
   - Research: bubbles uses `lipgloss.NewStyle().Height()` internally but it doesn't pad for few items

2. **lipgloss.Place() doesn't work with styled content**
   - We tried using `lipgloss.Place(width, height, ...)` to enforce exact dimensions
   - But Place() is designed for "unstyled whitespace boxes"
   - It doesn't properly handle ANSI codes, borders, or styled content
   - Results in content being truncated or incorrectly positioned

3. **lipgloss.Height() behavior**
   - `Height(n)` sets TOTAL rendered height (including borders, excluding margins)
   - ALWAYS pads with blank lines when content is shorter
   - Borders add 2 lines (1 top + 1 bottom) to the total
   - Use `GetVerticalFrameSize()` to calculate content area

### Current Approach (IN PROGRESS)

Using `lipgloss.NewStyle().Height(m.height)` instead of `Place()`:
```go
container := lipgloss.NewStyle().
    Width(m.width).
    Height(m.height)
result = container.Render(result)
```

This should properly enforce exact height while preserving styling.

### Status
- **Tests are failing** after this change
- Need to investigate test failure before proceeding
- The styled container approach is theoretically correct but may need adjustment

### Files Modified
- `internal/tui/detail.go` - height calculation and rendering
- `internal/tui/tui.go` - debug logging

### Debug Findings
From `/tmp/tui-debug.txt`:
- 0 links: Renders 44 lines (need 45) = 1 short
- 1 link: Renders 43 lines (need 45) = 2 short  
- 6 links: Renders 43 lines (need 45) = 2 short

Component breakdown shows internal math is correct (4+5+34=43, 4+9+30=43), but we're missing newlines in the joining.

### Next Steps
1. Fix test failure caused by styled container change
2. Verify height alignment with all link scenarios (0, 1, 5+)
3. Consider alternative: manually calculate and add padding newlines instead of relying on lipgloss
4. Remove debug logging once fixed