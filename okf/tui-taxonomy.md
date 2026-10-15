---
type: Reference
title: TUI taxonomy
description: The names of the terminal UI's views, layouts, panes, bean row columns and their parts, each with the code location that defines it.
resource: ../internal/tui/
tags: [tui, taxonomy, layout]
generated: { by: claude/opus-5-5, at: 2026-10-02T06:20:37Z }
---

# TUI taxonomy

This file lists the names of the terminal UI's elements, as the code uses them. It covers the
Bubbletea TUI (`beans-tui`, `internal/tui`, `internal/ui`), not the Beans UI in the browser. Each
name gives the code location that defines it.

## Views

A view fills the terminal. `App.state` (type `viewState`, `internal/tui/tui.go`) holds the active one.

| Name | Code | Opened by |
|---|---|---|
| list view | `viewList`, `listModel` (`list.go`) | start; `esc` from the detail view |
| detail view | `viewDetail`, `detailModel` (`detail.go`) | `enter` on a bean |
| tag picker | `viewTagPicker`, `tagPickerModel` | `g t` |
| parent picker | `viewParentPicker`, `parentPickerModel` | `p` |
| status picker | `viewStatusPicker`, `statusPickerModel` | `s` |
| type picker | `viewTypePicker`, `typePickerModel` | `t` |
| priority picker | `viewPriorityPicker`, `priorityPickerModel` | `P` |
| blocking picker | `viewBlockingPicker`, `blockingPickerModel` | `b` |
| create modal | `viewCreateModal`, `createModalModel` | `c` |
| help overlay | `viewHelpOverlay`, `helpOverlayModel` (`help.go`) | `?` |
| settings overlay | `viewSettingsOverlay`, `settingsOverlayModel` (`settingsoverlay.go`) | `,` |

The tag picker fills the terminal. The other pickers, the create modal, the help overlay and the
settings overlay are **modals**: `ModalView` draws them, dimmed, over the view they were opened from
(`App.previousState`). Over the list view, the background is the list view in its current layout
(`renderListLayout`).

## Layouts of the list view

| Name | Code | Arrangement |
|---|---|---|
| two-column layout | `isTwoColumnMode`, `renderTwoColumnView` | list pane left, preview pane right |
| stacked layout | `isStackedMode`, `renderStackedView` | preview pane below the list pane |
| single-column layout | neither of the above | list pane alone |

**Preview position** (`tuiSettings.previewPosition`, config `tui.preview_position`) chooses between
the two-column and the stacked layout:

- `auto` selects by terminal size.
- `right` forces the two-column layout; if it does not fit the terminal, the stacked layout applies,
  and if that does not fit either, the single-column layout.
- `below` forces the stacked layout while the list pane and the preview pane each get at least
  `config.MinStackedListHeight` rows; otherwise the single-column layout applies.
- The key `v` toggles between `right` and `below` until the TUI exits.

The **preview ratio** (`tuiSettings.previewPercent`, default 50) is the preview pane's share of the
height above the footer in the stacked layout. It applies only to position `below`, and it is kept
on resize, so both pane heights follow the terminal height (`stackedListPaneHeight`). Positions
`auto` and `right` give the stacked list pane the fixed height `tui.stacked_list_height`.

The constants `TwoColumnFullWidth`, `TwoColumnMinWidth`, `StackedBelowListMinHeight` and the config
`tui.stacked_list_height` set the size limits.

The **settings overlay** (`,`) sets the preview position, the preview ratio and the view mode. Its
settings last until the TUI exits; `.beans.yml` stays unchanged.

## Panes of the list view

A pane is a bordered rectangle; its **border title** (`withBorderTitle`, `styles.go`) sits in the
top border.

| Name | Code | Content |
|---|---|---|
| list pane (left pane in the two-column layout) | `listModel.viewContent` | the bean rows; border title "Beans", followed by the **view mode** unless it is "all", and by "[tag: …]"; replaced by the filter input while typing a filter |
| preview pane (right pane in the two-column layout) | `previewModel` (`preview.go`) | the highlighted bean; border title is its ID |
| footer | `listModel.Footer` | one unbordered line: key help, the selection count "(N selected)", or a status message |

The **view mode** (`viewMode`, `list.go`) selects which beans the list pane shows; `h` cycles
through the modes, and each one narrows the previous:

| Mode | Shows |
|---|---|
| all | every bean |
| active | beans without an archive status and not below a closed ancestor |
| unblocked | active beans that are not blocked, directly or via an ancestor |
| ready | unblocked beans that are not in-progress or draft; the same beans as `beans list --ready` |

## Columns of a bean row

`ui.RenderBeanRow` (`internal/ui/styles.go`) renders one row per bean, used by both the TUI list pane and
the CLI tree. `ui.CalculateResponsiveColumns` computes the column widths (`ResponsiveColumns`) from
the width it is given: from 120 columns the type and status columns show names, 5 columns wide,
and grow to full width at 160; from 140 the tags column appears, 24 columns wide, and grows to 70
between 160 and 220. The title column takes the rest.

| Order | Name | Content |
|---|---|---|
| 1 | cursor | `▌` on the highlighted row |
| 2 | ID column | bean ID, preceded by the **tree prefix** (`├─`, `└─`) |
| 3 | type column | type, as one letter below `ColWidthNameMin`, otherwise the name cut to `TypeColWidth` |
| 4 | status column | status, like the type column with `StatusColWidth`; a red `↑` marks an **implicit status** inherited from a closed ancestor, otherwise an amber `⊘` marks a **blocked** open bean |
| 5 | priority symbol | before the title, only for a priority other than normal |
| 6 | title column | title, cut with `...` at `MaxTitleWidth` |
| 7 | tags column | up to `MaxTags` tags, only when `ShowTags` is set |

States of a row:

- **highlighted** (`IsSelected`): the cursor is on it; the preview pane shows this bean.
- **marked** (`IsMarked`): chosen for a batch edit with `space`.
- **dimmed** (`Dimmed`): an ancestor shown for context that does not match the filter itself.

## Parts of the preview pane

| Name | Code | Content |
|---|---|---|
| header | `previewModel.renderHeader` | title, the metadata line "Status: … Type: … Priority: …", the tag line; an amber `⊘ blocked` follows the status when the list row carries the `⊘` mark |
| body | `previewModel.renderBody` | the bean's markdown; scrolls (`scrollBy`) with the mouse wheel and `pgup`/`pgdn`, while the header stays |

## Parts of the detail view

| Name | Code | Content |
|---|---|---|
| header box | `detailModel.renderHeader` | title, ID, status badge, tags |
| links section | `detailModel.linkList` | related beans: parent, children, blocking, blocked by; takes focus with `tab` |
| body | `detailModel.viewport` | the bean's markdown |
| footer | `detailModel.View` | scroll percentage and key help |

The focused one of links section and body has a border in the primary color (`linksActive`).
