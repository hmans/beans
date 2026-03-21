---
# beans-ocss
title: Persist workspace port assignments across restarts
status: completed
type: task
priority: normal
created_at: 2026-03-21T09:29:08Z
updated_at: 2026-03-21T09:31:11Z
---

Add Port field to worktreeMeta so port assignments survive beans serve restarts. Also document worktree metadata pattern in CLAUDE.md.


## Summary of Changes

- Added `AllocateSpecific(workspaceID, port)` to `portalloc.Allocator` — restores a specific port for a workspace, falling back to normal allocation on conflict
- Added `Port` field to `worktreeMeta` struct, with `SavePort`/`GetPort` methods on `worktree.Manager`
- Updated `serve.go` startup to restore persisted ports from worktree metadata before falling back to fresh allocation
- Updated `CreateWorktree` resolver to persist allocated port to metadata
- Added tests for `AllocateSpecific` (happy path, idempotency, conflict, nextIndex advancement)
- Documented the worktree metadata file pattern in CLAUDE.md
