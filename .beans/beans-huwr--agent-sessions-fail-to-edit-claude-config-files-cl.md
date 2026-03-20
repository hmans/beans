---
# beans-huwr
title: Agent sessions fail to edit Claude config files (.claude/rules, CLAUDE.md)
status: in-progress
type: bug
created_at: 2026-03-20T14:46:47Z
updated_at: 2026-03-20T14:46:47Z
---

When a Beans UI agent (running with --dangerously-skip-permissions) tries to write to .claude/rules/*.md or CLAUDE.md, Claude Code still blocks the write because it considers these 'sensitive files (project rules)'. The agent asks the user to approve the permission, but there's no way to grant it in non-interactive mode. Fix: add --allowedTools entries for Edit and Write on these sensitive file paths.
