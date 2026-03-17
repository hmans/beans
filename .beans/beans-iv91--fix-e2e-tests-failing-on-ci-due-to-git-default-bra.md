---
# beans-iv91
title: Fix e2e tests failing on CI due to git default branch name
status: completed
type: bug
priority: normal
created_at: 2026-03-17T16:16:44Z
updated_at: 2026-03-17T16:18:01Z
---

E2e tests fail on CI because git init creates 'master' branch on Ubuntu while the app expects 'main'. The fix is to use git init -b main in the e2e fixtures.

## Summary of Changes

The e2e test fixtures in `frontend/e2e/fixtures.ts` used `git init` without specifying a branch name. On macOS, git defaults to `main`, but on CI (Ubuntu), git defaults to `master`. Since the app's worktree base ref defaults to `main`, the `git worktree add ... main` command failed silently, causing the `createWorktree` mutation to error and navigation to never happen.

Fixed by adding `-b main` to both `git init` calls in the fixtures (template creation and per-test repo creation).
