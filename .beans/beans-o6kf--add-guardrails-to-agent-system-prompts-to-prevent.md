---
# beans-o6kf
title: Add guardrails to agent system prompts to prevent auto-merging PRs
status: completed
type: task
priority: normal
created_at: 2026-03-18T14:40:49Z
updated_at: 2026-03-18T14:40:54Z
---

Agents spawned by beans-serve should never merge PRs or assume CI checks are absent. Add explicit instructions to both the worktree and central agent system prompts.

## Summary of Changes

Added two guardrails to both the central and worktree agent system prompts in `internal/commands/serve.go`:

1. **Never merge PRs** — agents must stop after creating a PR and report the URL
2. **Never assume CI is absent** — empty `gh pr checks` means checks haven't started yet, not that none are configured
