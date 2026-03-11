---
# beans-ct2v
title: Rename Changes pane to Status and add workflow action buttons
status: completed
type: feature
priority: normal
created_at: 2026-03-10T21:56:53Z
updated_at: 2026-03-11T16:46:51Z
order: zzzz
---

## Changes

1. Rename the 'Changes' pane header to 'Status'
2. Add a workflow action buttons area at the bottom of the Status pane
3. Hardcode two buttons initially:
   - **Commit** — sends 'Create a commit' to the current worktree/repo agent conversation
   - **Review** — sends 'Ask a subagent for a thorough code review' to the current worktree/repo agent conversation

## Tasks

- [x] Rename 'Changes' pane header to 'Status'
- [x] Add workflow action buttons area at bottom of Status pane
- [x] Implement 'Commit' button that auto-sends message to agent chat
- [x] Implement 'Review' button that auto-sends message to agent chat
- [x] Write/update e2e tests (none needed — no existing tests reference Changes)

## Summary of Changes

- Renamed "Changes" pane to "Status" across all views and tooltips
- Added Commit and Review workflow action buttons at the bottom of the Status pane
- Buttons send messages to the agent chat via GraphQL mutation
- Buttons are disabled while the agent is busy
- Lifted AgentChatStore into parent views to avoid duplicate subscriptions
- AgentChat accepts optional external store prop
- Added build warnings rule to frontend.md
