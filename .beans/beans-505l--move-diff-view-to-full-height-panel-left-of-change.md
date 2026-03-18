---
# beans-505l
title: Move diff view to full-height panel left of Changes pane
status: completed
type: task
priority: normal
created_at: 2026-03-18T14:08:32Z
updated_at: 2026-03-18T14:11:27Z
---

Extract the diff view from inside ChangesPane (where it's in a vertical split below the file list) into its own full-height panel positioned to the left of ChangesPane in WorkspaceView's horizontal SplitPane.

## Summary of Changes

Extracted the diff view from inside ChangesPane into a full-height panel (DiffPane) positioned to the left of the Changes pane in WorkspaceView's horizontal SplitPane.

### Files changed:
- **frontend/src/lib/diffSelection.svelte.ts** (new) — Shared singleton store managing diff selection state, fetching, and parsing. Extracted from ChangesPane so both DiffPane and ChangesPane can share state.
- **frontend/src/lib/components/DiffPane.svelte** (new) — Full-height diff viewer component that reads from the shared store.
- **frontend/src/lib/components/ChangesPane.svelte** — Removed inline diff view, SplitPane, and diff state. Now delegates to diffSelectionStore for file selection.
- **frontend/src/lib/components/WorkspaceView.svelte** — Added DiffPane as a new collapsible panel (600px default) between AgentChat and ChangesPane. Clears diff selection when Changes pane is closed.
