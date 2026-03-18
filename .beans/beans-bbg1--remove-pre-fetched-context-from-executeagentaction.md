---
# beans-bbg1
title: Remove pre-fetched context from ExecuteAgentAction, let agents gather their own context
status: completed
type: task
priority: normal
created_at: 2026-03-18T09:59:28Z
updated_at: 2026-03-18T10:01:19Z
---

## Summary of Changes

Removed all blocking I/O (git status checks, forge API calls, worktree listing) from the `ExecuteAgentAction` mutation resolver. The mutation now only passes static context (working directory path, main repo path, forge CLI name) to prompt generation. The agents themselves will inspect git state and PR status as needed.

Simplified prompts:
- **commit**: Single prompt that tells the agent to inspect git status itself
- **create-pr**: Single unified prompt that tells the agent to check PR state and take the appropriate action, instead of pre-branching into 4 different prompts based on pre-fetched state
- **integrate**: Unchanged (only used `MainRepoPath`, which is a path lookup, not I/O)
- **review/learn/tests**: Already didn't use context, unchanged
