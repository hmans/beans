---
# beans-32ue
title: 'Fix: beancore picks up non-bean .md files from worktrees/ and conversations/'
status: completed
type: bug
priority: normal
created_at: 2026-03-10T17:22:54Z
updated_at: 2026-03-10T17:32:05Z
---

loadFromDisk() and Watch() recursively walk the entire .beans/ directory, picking up .md files from worktrees/ and conversations/ subdirectories as bogus beans.

## Summary of Changes

- Renamed `worktrees/` → `.worktrees/` and `conversations/` → `.conversations/`
- Skip all dot-prefixed subdirectories in `loadFromDisk()`, `Watch()`, and `watchLoop()`
- Added `migrateLegacyDirs()` to auto-rename old directories on load
- Updated `.gitignore` template, worktree path, agent store path, e2e tests, and CLAUDE.md rules
- Added `TestLoadSkipsDotPrefixedSubdirectories` test
