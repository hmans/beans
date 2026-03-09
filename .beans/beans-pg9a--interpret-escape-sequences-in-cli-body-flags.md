---
# beans-pg9a
title: Interpret escape sequences in CLI body flags
status: completed
type: bug
priority: normal
created_at: 2026-03-09T12:19:01Z
updated_at: 2026-03-09T12:21:38Z
---

CLI body flags (--body-append, --body, --body-replace-old/new) pass literal \\n instead of real newlines when called by agents. Add unescapeBody() to interpret \\n, \\t, \\\\ sequences.

## Summary of Changes

- Added `UnescapeBody()` to `internal/bean/content.go` — interprets \\n, \\t, \\\\ escape sequences
- Applied unescaping in CLI content resolution (`resolveContent`, `resolveAppendContent`, body-replace flags)
- Added table-driven tests for `UnescapeBody`
- Fixed 9 existing bean files that had literal \\n in their bodies
