---
title: 'TUI: ''e'' shortcut to edit bean in $EDITOR'
status: completed
type: feature
priority: normal
created_at: 2025-12-13T00:25:45Z
updated_at: 2025-12-13T00:38:09Z
parent: beans-xnp8
---

Add an 'e' keyboard shortcut to the TUI that opens the selected bean's markdown file in an external editor.

This enables full editing of the bean (title, body, frontmatter) without leaving the TUI workflow. When the editor closes, the TUI should detect the file change and refresh.

## Editor Detection

Use this fallback chain:
1. `$VISUAL`
2. `$EDITOR`
3. `nano`
4. `vi`

## Implementation Notes

- Suspend the TUI while the editor is open (use tea.ExecProcess or similar)
- The file watcher should already handle refreshing when the file changes
- Add 'e' keybinding to both list and detail views