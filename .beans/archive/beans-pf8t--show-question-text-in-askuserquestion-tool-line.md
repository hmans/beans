---
# beans-pf8t
title: Show question text in AskUserQuestion tool line
status: completed
type: task
priority: normal
created_at: 2026-03-13T20:13:37Z
updated_at: 2026-03-13T20:14:25Z
---

The AskUserQuestion tool line in the agent chat only shows 'AskUserQuestion' with no context. It should include the question text so users can see what was asked.

## Summary of Changes

Added special-case handling in `extractToolSummary` (internal/agent/parse.go) to extract the first question's text from AskUserQuestion tool input's `questions` array. Previously, the tool line just showed "AskUserQuestion" with no context; now it shows the actual question text (truncated to 80 chars if needed).

Added 3 test cases for this behavior.
