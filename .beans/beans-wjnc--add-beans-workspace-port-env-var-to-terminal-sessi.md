---
# beans-wjnc
title: Add BEANS_WORKSPACE_PORT env var to terminal sessions
status: completed
type: feature
priority: normal
created_at: 2026-03-17T13:51:52Z
updated_at: 2026-03-17T13:55:29Z
---

Each workspace gets a unique port (starting at 44000, incrementing by 10) injected as BEANS_WORKSPACE_PORT into terminal/run sessions. Ports are managed in RAM and freed when workspaces are destroyed.

## Tasks
- [x] Create port allocator package
- [x] Integrate with terminal session creation
- [x] Allocate ports on workspace creation, free on removal
- [x] Allocate a port for the central workspace
- [x] Write tests for port allocator
- [x] Write tests for terminal env var injection

## Summary of Changes

- Created `internal/portalloc/` package with in-memory port allocator (base 44000, step 10, port recycling)
- Added `EnvFunc` to `terminal.Manager` for injecting per-session env vars
- Wired port allocation into workspace lifecycle: allocate on create, free on remove
- Central workspace and existing worktrees get ports allocated at server startup
- `BEANS_WORKSPACE_PORT` env var injected into all terminal sessions
