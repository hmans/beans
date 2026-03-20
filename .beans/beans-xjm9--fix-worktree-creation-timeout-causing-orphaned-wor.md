---
# beans-xjm9
title: Fix worktree creation timeout causing orphaned worktrees
status: completed
type: bug
priority: normal
created_at: 2026-03-20T14:21:04Z
updated_at: 2026-03-20T14:28:40Z
---

When fetchBaseRef() hangs (SSH agent issues, remote down, slow network), the HTTP server's 15s WriteTimeout kills the connection. The frontend gets an error, but the backend goroutine continues, creating an orphaned worktree with no agent session. Fix by adding a context-aware timeout to fetchBaseRef, making it configurable, and skipping fetch for local-only base refs.

## Summary of Changes

- Added context-aware timeout to `fetchBaseRef()` using `exec.CommandContext` with configurable timeout (default 10s, below the HTTP server's 15s WriteTimeout)
- Added `fetchTimeout` field to worktree `Manager` with functional option `WithFetchTimeout()`
- Added `worktree.fetch_timeout` config option in `.beans.yml` (integer seconds; 0 disables fetch entirely for airgapped environments)
- Added `GetWorktreeFetchTimeout()` config getter with proper nil-pointer handling for the default
- Wired config through `serve.go` to pass fetch timeout to worktree manager
- Added tests for: default timeout, custom timeout, zero-disables-fetch, timeout behavior with hanging remote, config loading from YAML
