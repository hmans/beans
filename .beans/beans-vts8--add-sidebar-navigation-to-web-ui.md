---
# beans-vts8
title: Add sidebar navigation to web UI
status: in-progress
type: feature
priority: normal
created_at: 2026-03-10T18:56:08Z
updated_at: 2026-03-10T18:58:10Z
---

Add a sidebar to the beans-serve web UI with:
- Planning item at top (current Backlog/Board view + central agent)
- List of beans with active worktrees
- Each workspace shows agent chat + bean detail in split pane

## Tasks
- [x] Add activeView state to uiState.svelte.ts
- [x] Update +layout.ts and +layout.svelte for persistence/fallback
- [x] Extract PlanningView.svelte from +page.svelte
- [x] Create Sidebar.svelte component
- [x] Create WorkspaceView.svelte component
- [x] Rewrite +page.svelte as thin shell
