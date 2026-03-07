---
# beans-d91u
title: 'Add tabs to web app: Backlog and Board views'
status: completed
type: feature
priority: normal
created_at: 2026-03-07T22:21:38Z
updated_at: 2026-03-07T22:23:04Z
---

Add tab navigation to the main view with two tabs:\n1. Backlog - the current list view of all beans\n2. Board - a kanban board showing non-completed beans arranged by status

## Summary of Changes\n\nAdded tab navigation to the web app's main view with two tabs:\n\n- **Backlog** - The existing list view showing all beans in a tree structure with the detail pane on the right\n- **Board** - A kanban board view with columns for Draft, Todo, and In Progress statuses. Clicking a card opens a detail pane on the right side.\n\nThe active tab is persisted to localStorage. Both views share the same bean selection/detail mechanism.
