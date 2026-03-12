---
# beans-5gw7
title: Remove agent chat, status pane, and terminal from main planning view
status: completed
type: task
priority: normal
created_at: 2026-03-12T08:26:03Z
updated_at: 2026-03-12T08:59:39Z
---

The main planning view should be for planning only, not for doing work. Remove the agent chat, status pane, and terminal panels since all work should happen in worktrees.

## Summary of Changes

Removed the agent chat, status pane (ChangesPane), and terminal from the main PlanningView component. The planning view is now focused purely on planning — the backlog/board with a detail pane. All agent/terminal functionality remains available in worktree WorkspaceViews where actual work happens.

Changes in `frontend/src/lib/components/PlanningView.svelte`:
- Removed imports: AgentChatStore, AgentChat, ChangesPane, TerminalPane, configStore, onDestroy
- Removed central agent session subscription and lifecycle management
- Removed Status, Agent, and Terminal toggle buttons from the toolbar
- Simplified layout from 4 nested SplitPanes to a single SplitPane (content + detail aside)
