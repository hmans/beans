---
# beans-swx6
title: 'Fix workspace description generation: invalid -m flag'
status: completed
type: bug
priority: normal
created_at: 2026-03-13T18:17:11Z
updated_at: 2026-03-13T18:17:47Z
---

The GenerateDescription function in internal/agent/describe.go uses `-m haiku` flag which is not supported by the claude CLI. The correct flag is `--model haiku`. This causes all description generation to silently fail.

## Summary of Changes

Fixed `GenerateDescription` in `internal/agent/describe.go`: changed invalid `-m` flag to `--model` for the `claude` CLI invocation. The `-m` shorthand doesn't exist in Claude Code CLI, causing all workspace description generation to silently fail with `error: unknown option '-m'`.
