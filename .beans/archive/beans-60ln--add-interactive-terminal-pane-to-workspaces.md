---
# beans-60ln
title: Add interactive terminal pane to workspaces
status: completed
type: feature
priority: normal
created_at: 2026-03-12T07:23:31Z
updated_at: 2026-03-12T07:33:29Z
order: zz
---

Add a toggleable terminal pane (interactive PTY) to both worktree and main workspaces. Uses xterm.js on the frontend, a dedicated WebSocket endpoint on the backend with creack/pty for PTY management.

## Tasks

- [x] Backend: Create internal/terminal package (PTY session manager)
- [x] Backend: Add /api/terminal WebSocket endpoint to serve.go
- [x] Frontend: Install @xterm/xterm and @xterm/addon-fit
- [x] Frontend: Create TerminalPane component
- [x] Frontend: Add showTerminal toggle to UIState + layout
- [x] Frontend: Integrate into WorkspaceView with toggle button
- [x] Frontend: Integrate into PlanningView with toggle button
- [x] Write tests

## Summary of Changes

### Backend
- **New package `internal/terminal/`**: PTY session manager using `creack/pty`. Manages shell process lifecycle (create, resize, read/write, close) with session ID keying and mutex safety.
- **New file `internal/commands/terminal_handler.go`**: WebSocket endpoint at `/api/terminal` that bridges browser WebSocket to PTY. Protocol: JSON control messages (init, input, resize) from client, binary PTY output frames to client. Validates working directory against known worktrees.
- **Modified `internal/commands/serve.go`**: Wires up terminal manager and registers the WebSocket route.

### Frontend
- **New dependency**: `@xterm/xterm` + `@xterm/addon-fit` for terminal emulation.
- **New component `TerminalPane.svelte`**: Mounts xterm.js, connects via WebSocket, handles input/output/resize. Uses ResizeObserver + FitAddon to auto-fit when SplitPane is dragged.
- **`uiState.svelte.ts`**: Added `showTerminal` toggle with localStorage persistence.
- **`+layout.ts` / `+layout.svelte`**: Load and initialize terminal visibility state.
- **`WorkspaceView.svelte`**: Wrapped in vertical SplitPane for bottom terminal pane, added toggle icon button in agent toolbar.
- **`PlanningView.svelte`**: Same treatment — vertical SplitPane wrapper, toggle button in planning toolbar.

### Tests
- Unit tests for terminal manager: create, close, replace, resize, I/O, shutdown.
