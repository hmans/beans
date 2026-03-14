---
# beans-ukvl
title: 'Fix worktree integrate: use atomic update-ref to prevent race conditions and remove config hack'
status: completed
type: bug
priority: normal
created_at: 2026-03-14T09:45:42Z
updated_at: 2026-03-14T09:46:28Z
---

The current integrate action uses git diff main...HEAD | git apply --index which has two problems: (1) requires receive.denyCurrentBranch config hack, (2) silently overwrites concurrent integrations. Replace with rebase + squash + atomic git update-ref.

## Summary of Changes

Replaced the integrate action prompt with a safer approach:
- Removed the `receive.denyCurrentBranch updateInstead` config hack
- Replaced `git diff | git apply` with `git rebase main && git reset --soft main && git commit`
- Added atomic `git update-ref refs/heads/main HEAD $MAIN_SHA` (compare-and-swap) to prevent race conditions when two worktrees integrate concurrently
- On CAS failure, the agent retries by re-rebasing onto the new main
