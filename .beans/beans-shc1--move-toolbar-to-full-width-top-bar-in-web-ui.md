---
# beans-shc1
title: Move toolbar to full-width top bar in web UI
status: completed
type: task
priority: normal
created_at: 2026-03-10T21:27:50Z
updated_at: 2026-03-11T16:46:51Z
order: zzzzw
---

Move the PlanningView toolbar (New Bean, Backlog/Board toggle, filter, Changes/Agent buttons) from inside the backlog/board pane to span the full width of the main content area, with all panes nested below it.

## Summary of Changes

Extracted the PlanningView toolbar from inside the inner SplitPane (where it only spanned the backlog/board pane) to the top of the component, wrapping everything in a flex column container. The toolbar now spans the full width of the main content area, with all panes (backlog/board, detail, changes, agent chat) nested below it.
