---
# beans-zrn7
title: Forge abstraction + GitHub PR integration
status: completed
type: feature
priority: normal
created_at: 2026-03-17T13:41:58Z
updated_at: 2026-03-17T13:49:07Z
---

Add a forge provider abstraction (pkg/forge/) with GitHub implementation via gh CLI. Add Create PR / Update PR agent action button next to Integrate. Auto-detect existing PRs for worktree branches.

## Todo

- [x] Create `pkg/forge/` with Provider interface and types
- [x] Implement GitHub provider using `gh` CLI
- [x] Add `PullRequest` type to GraphQL schema
- [x] Add `pullRequest` field to `Worktree` type
- [x] Wire forge detection into server startup
- [x] Add `create-pr` agent action with dynamic label (Create PR / Update PR)
- [x] Populate forge context in action resolution and execution
- [x] Style the Create PR button in AgentActions.svelte
- [x] Write unit tests for forge package
- [x] Verify all tests pass and frontend builds cleanly

## Summary of Changes

Added a forge provider abstraction (`pkg/forge/`) with a GitHub implementation that uses the `gh` CLI. The system auto-detects the forge from the git remote URL on server startup.

Key components:
- **`pkg/forge/forge.go`** — Provider interface, PullRequest type, auto-detection from git remote
- **`pkg/forge/github.go`** — GitHub implementation: FindPR (via `gh pr list`) and CreatePR (via `gh pr create`)
- **GraphQL schema** — Added `PullRequest` type and `pullRequest` field on `Worktree`
- **Agent action** — New `create-pr` action with dynamic label ("Create PR" when no PR exists, "Update PR" when one does)
- **Frontend** — Styled the PR button with accent color and a code-branch icon in AgentActions.svelte

The GitLab provider can be added later by implementing the same `forge.Provider` interface with a `glab` CLI backend.
