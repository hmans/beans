---
# beans-i597
title: 'TUI preview: show blocked status'
status: completed
type: feature
priority: normal
created_at: 2026-10-02T09:25:53Z
updated_at: 2026-10-02T09:26:59Z
---

The TUI list marks a blocked open bean with an amber ⊘, the preview pane does not. Show the blocked state in the preview's metadata line as well, with the same precedence as the list: a closed ancestor (↑) suppresses the blocked mark.

- [x] Carry the blocked flag from the list item to the preview
- [x] Render "⊘ blocked" after the status in the preview
- [x] Tests
- [x] Update okf/tui-taxonomy.md

## Summary of Changes

- `previewModel.blocked`: the metadata line renders an amber `⊘ blocked` after the status.
- `beanItem.showsBlocked` holds the list precedence (closed ancestor beats blocker); `cursorChangedMsg` carries it, and the `beansLoadedMsg` handler sets it from the selected item.
- Tests: `TestPreviewViewBlocked`, `TestPreviewBlockedFromList`.
- `okf/tui-taxonomy.md` mentions the mark in the preview header.
