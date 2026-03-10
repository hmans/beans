---
# beans-htnb
title: Fix agent chat Act mode permission handling
status: completed
type: bug
priority: normal
created_at: 2026-03-10T11:57:44Z
updated_at: 2026-03-10T12:02:10Z
---

Use acceptEdits permission mode and add Bash(beans:*) allowed tool for Act mode. Add logging for permission denials.

## Summary of Changes

- **`internal/agent/claude.go` — `buildClaudeArgs()`**: Act mode now uses `--permission-mode acceptEdits` and `--allowedTools Bash(beans:*)` so file operations and beans CLI work without permission prompts.
- **`internal/agent/claude.go` — `readOutput()`**: Permission denials now log individual tool names (`permission denied: tool=X`) instead of just a count.
- **`internal/agent/manager_test.go`**: Updated `TestBuildClaudeArgs_ActMode`, `TestBuildClaudeArgs_NoPlanMode`, and `TestBuildClaudeArgs_AllowedTools` to match the new behavior.
