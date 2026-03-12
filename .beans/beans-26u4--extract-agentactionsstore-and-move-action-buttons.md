---
# beans-26u4
title: Extract AgentActionsStore and move action buttons to toolbar
status: completed
type: task
priority: normal
created_at: 2026-03-12T18:58:24Z
updated_at: 2026-03-12T19:02:18Z
---

Extract agent action fetch/execute logic from ChangesPane into a shared AgentActionsStore class (agentActions.svelte.ts). Move action buttons into ViewToolbar via a new 'right' snippet slot. Reorder toolbar buttons (Agent, Changes, Terminal) and move planning agent chat panel to left side of layout.

## Summary of Changes

- Created `agentActions.svelte.ts` with `AgentActionsStore` class encapsulating GraphQL queries, state, fetch, and execute logic
- Removed agent action code from `ChangesPane.svelte` (no longer needs beanId or agentBusy props)
- Added `right` snippet slot to `ViewToolbar.svelte` for action buttons
- Reordered toolbar: Agent toggle, Changes toggle, Terminal toggle (left-to-right)
- Both `PlanningView` and `WorkspaceView` now use `AgentActionsStore` and render action buttons in the toolbar
- Moved agent chat panel to left side in `PlanningView` SplitPane layout

## Code Review Fixes

- Moved idle-transition tracking (wasAgentBusy pattern) into AgentActionsStore.notifyAgentStatus()
- Added agentBusy guard to AgentActionsStore.execute() for defense-in-depth
- Cleaned up extra blank lines in ChangesPane.svelte
