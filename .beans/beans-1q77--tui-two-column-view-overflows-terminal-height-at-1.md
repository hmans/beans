---
# beans-1q77
title: TUI two-column view overflows terminal height at 120x28
status: completed
type: bug
priority: normal
created_at: 2026-10-01T09:03:53Z
updated_at: 2026-10-06T12:47:24Z
---

At terminal size 120x28 the two-column TUI renders more lines than the terminal height; the top scrolls out of view. 120x27 and 119x28 render correctly.

- [x] Reproduce with a test rendering App.View() at 120x28
- [x] Identify the overflowing pane/row
- [x] Fix root cause
- [x] Guard pane heights

## Root cause

`RenderBeanRow` appended the ` ↑<status>` implicit-status annotation after a title already truncated to the full title budget, so annotated rows were 11 columns too wide. In the 37-column left pane at width 120 they wrapped inside the border and grew the pane. Whether a page contains such rows depends on rows-per-page, hence on terminal height.

## Fix

The width fix in `RenderBeanRow` is superseded by beans-5fp6: the marker moved from a title suffix into the status column (`↑T`), so annotated rows have the same width as plain ones. `TestListViewConstrained_ExactSize` guards the pane size.

## Summary of Changes

- Root cause fixed via beans-5fp6 (status marker moved into the status column).
- `viewContent` clips the list content to the pane's inner width and height, so an over-wide row can no longer wrap and grow the pane.
- `TestListViewContentClipsWideRows` renders rows twice the list width through a stub delegate; without the clip the pane grows (10 instead of 7 rows at 40x5).

## Review (opencode, e660795)

- Clip ineffective for `innerHeight <= 0` (terminal of 4 rows or less): rejected; lipgloss cannot render a bordered box with 0 inner rows, and such terminals are not a supported layout.
- Test exercised the clip only via the zero title budget in `RenderBeanRow`: applied; `TestListViewContentClipsWideRows` uses a stub delegate with rows twice the list width. The zero budget itself is beans-q7ye.
- Test tree prefixes differ from real ones (`├─` vs `├─ `): rejected; no effect on the result.
- Modal background loses the pagination line: rejected; beans-376r renders the real layout behind modals.
- Figure "69 instead of 26 rows" unconfirmed: rejected; measured by running the test without the clip.
