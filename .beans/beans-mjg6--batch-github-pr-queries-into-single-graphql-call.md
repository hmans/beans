---
# beans-mjg6
title: Batch GitHub PR queries into single GraphQL call
status: completed
type: task
priority: normal
created_at: 2026-03-18T06:08:08Z
updated_at: 2026-03-18T06:12:21Z
---

Replace per-worktree gh CLI invocations in populatePRsAsync with a single gh api graphql call that fetches PR data for all branches at once, reducing GitHub API usage.

## Summary of Changes

- Added `FindPRs` batch method to the `forge.Provider` interface
- Implemented it on `GitHub` using a single `gh api graphql` call with aliased fields (one query fetches PR data for all branches)
- Added `ParseOwnerRepo` helper to extract owner/repo from git remote URLs
- Added `populatePRsBatch` in the resolver layer, replacing parallel per-worktree `populatePR` goroutines
- Updated both the `Worktrees` query resolver and the `WorktreesChanged` subscription to use batch fetching
- Added tests for `ParseOwnerRepo`, `graphQLCheckToStatusCheck`, and `graphQLPRToForge`
