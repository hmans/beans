---
# beans-r8mi
title: Split PR button catch-all prompt into state-specific prompts
status: completed
type: bug
priority: normal
created_at: 2026-03-19T15:13:29Z
updated_at: 2026-03-19T15:14:43Z
---

The create-pr action has a single catch-all PromptFunc that instructs the agent to check state and take the appropriate action. This often leads to the agent merging a PR even though the user only clicked 'Update PR'. Each button state (Create PR, Update PR, Fix Tests, Merge PR) should have its own focused prompt.

## Summary of Changes

Replaced the single catch-all `PromptFunc` for the `create-pr` action with a state-specific `prPrompt` function that returns focused prompts based on the current PR state:

- **No PR**: Only instructs to create a PR (commit, push, create)
- **Has local changes/unpushed commits**: Only instructs to update the PR (commit, push). Includes explicit "Do NOT merge" guardrail.
- **Checks failing**: Only instructs to fix the failures (inspect, fix, push). Includes explicit "Do NOT merge" guardrail.
- **Checks pass, mergeable**: Only instructs to merge.

Added tests verifying each state returns the correct prompt and that non-merge states include the "Do NOT merge" guardrail.
