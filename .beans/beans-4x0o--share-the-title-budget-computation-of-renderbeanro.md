---
# beans-4x0o
title: Share the title-budget computation of RenderBeanRow
status: todo
type: task
created_at: 2026-10-07T07:19:04Z
updated_at: 2026-10-07T07:19:04Z
blocking:
    - beans-q7ye
---

Three callers compute the title budget for `RenderBeanRow` (`internal/ui/styles.go`) with their own arithmetic. The title budget is the title width that makes a row exactly a given width.

| Caller | Formula | Minimum |
|---|---|---|
| TUI list (`internal/tui/list.go:64-72`) | `width − (id + status + type + 4) [− tags]` | 0 |
| TUI detail links (`internal/tui/detail.go:118-122`) | `width − (id + status + type + 12 + 4) [− tags] − 8` | 10 |
| CLI tree (`internal/ui/tree.go:198-204`) | `termWidth − (treeCol + 3 + 3 + 3) [− tags]` | 20 |

All three subtract the parts `RenderBeanRow` assembles (`styles.go:626-628`): optional cursor, ID column, three 1-column separators, type and status columns, optional tags column. Only the caller's extras differ (TUI cursor, detail label and border).

Suspected bug, unverified: with tags shown, `RenderBeanRow` adds a separator before the tags column (`styles.go:626`), which the TUI list does not subtract. Rows with tags would then be 1 column too wide.

## Design

```go
// TitleBudget returns the title width that makes a RenderBeanRow row
// exactly rowWidth wide. The result can be 0 or negative.
func TitleBudget(rowWidth int, cfg BeanRowConfig) int
```

`TitleBudget` reads the column widths from the same `BeanRowConfig` that `RenderBeanRow` receives. The minimum stays with each caller: the CLI has no layout fallback and no pane clip, so its floor of 20 stays; the TUI minimum is beans-q7ye.

## Todo

- [ ] Add `ui.TitleBudget` with a table-driven test: for each config (cursor on/off, tags on/off, tree depth), `lipgloss.Width` of the rendered row with a long title equals `rowWidth`
- [ ] TUI list delegate uses `TitleBudget`
- [ ] TUI detail links use `TitleBudget` (extras: label and border), keep the minimum 10
- [ ] CLI tree uses `TitleBudget`, keep the minimum 20
- [ ] Verify or refute the tags-separator bug; fix it in `TitleBudget` if confirmed
- [ ] `mise test`
