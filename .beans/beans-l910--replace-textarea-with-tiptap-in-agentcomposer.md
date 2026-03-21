---
# beans-l910
title: Replace textarea with tiptap in AgentComposer
status: completed
type: task
priority: normal
created_at: 2026-03-21T13:11:55Z
updated_at: 2026-03-21T13:14:22Z
---

Replace the plain textarea chat input in AgentComposer.svelte with a tiptap editor for richer input capabilities.

## Summary of Changes

- Installed `@tiptap/core`, `@tiptap/pm`, `@tiptap/starter-kit`, and `@tiptap/extension-placeholder`
- Replaced plain `<textarea>` in AgentComposer.svelte with a tiptap Editor instance
- Created custom `composerKeymap` extension for Enter (send), Shift-Tab (toggle mode), Escape (stop)
- Handled image paste via tiptap's `handlePaste` editor prop
- Maintained localStorage persistence and focus management
- Styled the editor to match the original textarea appearance
- Updated e2e test selector from `textarea` to `.composer-editor[contenteditable]`
