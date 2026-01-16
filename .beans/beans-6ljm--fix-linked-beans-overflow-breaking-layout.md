---
# beans-6ljm
title: Fix linked beans overflow breaking layout
status: completed
type: bug
priority: normal
created_at: 2025-12-30T18:40:03Z
updated_at: 2025-12-30T21:34:12Z
parent: beans-pn6z
---

When a bean has many linked beans, the view completely breaks - linked beans text overflows and wraps incorrectly, causing garbled display where list and detail panes overlap visually. Need to properly truncate or scroll the linked beans section.

## Related fixes completed

- **beans-65cw**: Changed `UseFullNames: false` in linked beans to use short type/status (single char) instead of full names. This fixed the overflow where lines were too long.
- **beans-e2gi**: Used `strings.TrimRight(m.linkList.View(), "\n ")` to remove empty trailing lines from the bubbles list. This worked but caused height calculation mismatches.

## Bubbles list component research

The `list.New(items, delegate, width, height)` height parameter is the **TOTAL height** for the entire component. The component internally divides this among:

1. **Title bar** (1 line) - if `showTitle` is true (default: true)
2. **Status bar** - if `showStatusBar` is true (default: true)
3. **Pagination** (1 line for dots) - if `showPagination` is true (default: true)
4. **Help** - if `showHelp` is true (default: true)
5. **Items** - remaining space, calculated as: `availHeight / (delegate.Height() + delegate.Spacing())`

### Key insight

The height you give is NOT "number of items + title". It's the total pixel/line budget. The component subtracts space for title, pagination, etc., and gives the rest to items.

### Correct height calculation

```go
// For showing up to N items:
height := 1                    // title
height += numItemsToShow       // items (delegate.Height()=1 each)
if totalItems > numItemsToShow {
    height++                   // pagination dots
}
// Don't add +1 for title again - it's already counted!
```

### Current issue (still not working)

The body viewport height calculation doesn't match what's actually rendered. The `calculateHeaderHeight()` tries to predict the height of header + links section, but there's still a mismatch causing the body to be too short.

## Potential simplification

Consider rendering linked beans manually without the bubbles list component. This gives full control over height and removes the complexity of predicting bubbles' internal layout calculations.