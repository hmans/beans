---
# beans-yggw
title: Terminal pane hidden by default with lazy initialization
status: completed
type: task
priority: normal
created_at: 2026-03-12T11:00:38Z
updated_at: 2026-03-12T11:02:13Z
---

Changes to terminal pane behavior:
1. Terminal pane should be hidden by default (not restored from localStorage on first visit)
2. User clicks 'Terminal' button to show it
3. Terminal should only initialize (create xterm instance, connect WebSocket) when shown for the first time
4. When hidden again, terminal keeps its state (no re-initialization when toggled back on)

Currently the terminal initializes immediately when the component mounts, and visibility is restored from localStorage.

## Summary of Changes

- **uiState.svelte.ts**: Added `terminalInitialized` flag. Terminal no longer persists to localStorage. `toggleTerminal()` sets `terminalInitialized = true` on first show.
- **+layout.ts**: Removed `showTerminal` from localStorage restore — terminal always starts hidden.
- **+layout.svelte**: Removed `showTerminal` initialization from load data.
- **PlanningView.svelte**: Changed `{#if ui.showTerminal}` to `{#if ui.terminalInitialized}` so the component stays mounted after first show.
- **WorkspaceView.svelte**: Same change as PlanningView.
