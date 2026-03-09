---
# beans-bu4x
title: Introduce SplitPane component for composable pane architecture
status: completed
type: feature
priority: normal
created_at: 2026-03-09T08:47:17Z
updated_at: 2026-03-09T10:13:29Z
---

Extract duplicated split layout logic from backlog, board, and worktree views into a reusable SplitPane component. Supports horizontal/vertical splits, nesting for 3+ panes, and per-instance resize persistence.

## Summary of Changes

- Created `SplitPane.svelte` component with support for:
  - `direction` prop (`horizontal`/`vertical`) for both side-by-side and stacked layouts
  - `side` prop (`start`/`end`) for controlling which pane has fixed size
  - `initialSize`, `min`, `max` for size constraints
  - `persistKey` for per-instance localStorage persistence
  - Self-contained resize logic using `getBoundingClientRect()` (works correctly when nested)
  - Composable via nesting for 3+ pane layouts
- Migrated all three views (backlog, board, worktree) to use SplitPane
- Removed global pane/drag state from `uiState.svelte.ts` (`paneWidth`, `isDragging`, `startDrag`, `onDrag`, `stopDrag`, `loadPaneWidth`)
- Removed global `svelte:window` mouse handlers from `+layout.svelte`
- Fixed resize bug: old code used raw `e.clientX` as width, which only worked for left-side panes

## TODO\n\n- [x] Add planningView state to uiState\n- [x] Create BeanPane.svelte (tabbed detail)\n- [x] Rewrite +page.svelte (single-page layout)\n- [x] Simplify +layout.svelte (remove nav bar)\n- [x] Add worktree indicators to BeanItem and BoardView\n- [x] Delete old routes and constants\n- [x] Update E2E tests

## Summary of Changes

- Collapsed multi-route UI into single-page two-pane layout
- Left pane: planning view with Backlog/Board toggle (button group)
- Right pane: BeanPane with tabbed detail (Bean tab + Chat tab when worktree active)
- Removed tab header navigation bar from layout
- Added green border-l-success indicator on beans with active worktrees (both list and board)
- Deleted /board and /worktree routes, consolidated to /
- Updated E2E board-page.ts to use toggle button instead of /board navigation
