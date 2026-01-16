---
# beans-csnk
title: 'Task 10: Testing and polish'
status: completed
type: task
priority: normal
created_at: 2025-12-30T16:38:53Z
updated_at: 2025-12-30T18:18:57Z
parent: beans-pn6z
---

## Overview

Final testing pass and polish to ensure everything works correctly.

## Files

- Modify: Various TUI files as needed

## Steps

### Step 1: Test all keyboard interactions

Run: `mise build && mise beans`

Test in wide mode (≥120 cols):
- [ ] j/k navigates list, detail updates
- [ ] Enter focuses detail (links if present, else body)
- [ ] Tab toggles between links and body
- [ ] Backspace returns to list (or navigates history)
- [ ] Edit shortcuts work from all focus states (p, s, t, P, b, e, y)
- [ ] / filters in list and links
- [ ] q quits from any state
- [ ] ? opens help
- [ ] Esc clears selection/filter (list only)
- [ ] Border colors change based on focus

### Step 2: Test narrow mode

Resize terminal below 120 cols:
- [ ] Only list visible by default
- [ ] Enter shows only detail
- [ ] Backspace shows only list
- [ ] All shortcuts work correctly

### Step 3: Test history navigation

- [ ] Follow a link (Enter on linked bean)
- [ ] Backspace goes back to previous bean
- [ ] Follow multiple links, Backspace navigates stack
- [ ] Empty history, Backspace returns to list

### Step 4: Test edge cases

- [ ] Empty list shows "No bean selected" in detail
- [ ] Bean with no links: Tab does nothing, Enter focuses body
- [ ] Terminal resize while in detail focus
- [ ] Open picker from detail, returns to detail on close

### Step 5: Fix any issues found

Document and fix any issues discovered during testing.

### Step 6: Run tests

```bash
mise test
```

Fix any failing tests.

### Step 7: Final commit

```bash
git add -A
git commit -m "test(tui): verify unified detail view works correctly

Refs: beans-pn6z"
```

### Step 8: Update beans-pn6z status

```bash
beans update beans-pn6z -s completed
```