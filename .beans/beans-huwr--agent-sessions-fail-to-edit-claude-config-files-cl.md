---
# beans-huwr
title: Agent sessions fail to edit Claude config files (.claude/rules, CLAUDE.md)
status: in-progress
type: bug
priority: normal
created_at: 2026-03-20T14:46:47Z
updated_at: 2026-03-20T15:17:40Z
---

When a Beans UI agent (running with --dangerously-skip-permissions) tries to write to .claude/rules/*.md or CLAUDE.md, Claude Code still blocks the write because it considers these 'sensitive files (project rules)'. The agent asks the user to approve the permission, but there's no way to grant it in non-interactive mode. Fix: add --allowedTools entries for Edit and Write on these sensitive file paths.

## Root Cause

The initial fix (commit 7a43a18) used relative path patterns like `Edit(.claude/**)` in `--allowedTools`, but Claude Code's Edit/Write tools operate on **absolute file paths**. The glob `.claude/**` never matched because the tool calls use paths like `/full/path/to/workdir/.claude/rules/frontend.md`.

Additionally, the patterns were passed as multiple values after a single `--allowedTools` flag, which could cause variadic argument parsing issues.

## Fix

- Use absolute paths based on `session.WorkDir` (e.g., `Edit(/path/to/workdir/.claude/**)`)
- Pass each pattern with its own `--allowedTools` flag to avoid variadic parsing ambiguity
