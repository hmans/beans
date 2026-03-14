---
# beans-03uj
title: Add worktree.setup and worktree.run config options
status: completed
type: feature
priority: normal
created_at: 2026-03-13T18:59:12Z
updated_at: 2026-03-13T19:05:21Z
---

Add setup command (runs after worktree creation) and run command (shows Run button in workspace toolbar)

## Summary of Changes

### Backend (Go)
- **`pkg/config/config.go`**: Added `Setup` and `Run` fields to `WorktreeConfig`, with accessors `GetWorktreeSetup()` and `GetWorktreeRun()`, and YAML serialization support.
- **`internal/worktree/worktree.go`**: Extended `NewManager()` to accept a `setupCommand` parameter. After `git worktree add`, the setup command is executed via `sh -c` in the new worktree directory (failures are logged but don't block creation).
- **`internal/commands/serve.go`**: Passes `cfg.GetWorktreeSetup()` to the worktree manager.
- **`internal/graph/schema.graphqls`**: Added `worktreeRunCommand: String!` query and `writeTerminalInput(sessionId, data): Boolean!` mutation.
- **`internal/graph/schema.resolvers.go`**: Implemented resolvers for both new schema fields.

### Frontend (Svelte)
- **`frontend/src/lib/config.svelte.ts`**: Added `worktreeRunCommand` to the config query and store.
- **`frontend/src/lib/components/WorkspaceView.svelte`**: Added "Run" button (with play icon) to the workspace toolbar. Clicking it opens the terminal and sends the configured command via the `writeTerminalInput` mutation.

### Tests
- Config: tests for loading/saving `setup` and `run` fields.
- Worktree: tests verifying setup command runs after creation (and that creation works without one).
