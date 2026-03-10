---
# beans-799s
title: 'Worktree: use configurable base ref instead of HEAD'
status: completed
type: feature
priority: normal
created_at: 2026-03-10T19:41:47Z
updated_at: 2026-03-10T19:47:35Z
---

Worktrees are currently created from HEAD. They should default to origin/main and be configurable via worktree.base_ref in .beans.yml.

## Summary of Changes

- Added `WorktreeConfig` with `base_ref` field to `pkg/config/config.go`
- Worktree manager now accepts and uses a `baseRef` parameter when creating new branches
- Default base ref is `origin/main` (previously used HEAD)
- Config is serialized/deserialized under the `worktree:` top-level key in `.beans.yml`
- Section is omitted from saved config when not explicitly set
- Added tests for config loading, saving, and worktree branch creation from base ref

- `beans init` now auto-detects the remote's default branch via `git symbolic-ref` and uses it as `worktree.base_ref`
- Falls back to `origin/main` when not in a git repo or no remote is configured
- Added `gitutil.DefaultRemoteBranch()` helper with tests
