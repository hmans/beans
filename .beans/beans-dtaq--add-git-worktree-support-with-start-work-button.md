---
# beans-dtaq
title: Add git worktree support with Start Work button
status: completed
type: feature
priority: normal
created_at: 2026-03-07T22:46:39Z
updated_at: 2026-03-07T22:54:37Z
---

Create git worktrees from the web UI via a Start Work button on beans.\n\n## Tasks\n\n- [x] Create internal/worktree Go package (list, create, remove)\n- [x] Add Worktree type and operations to GraphQL schema\n- [x] Add worktree subscription to GraphQL\n- [x] Implement resolvers\n- [x] Create frontend worktree store with subscription\n- [x] Add Start Work button to BeanDetail\n- [x] Add worktree tabs to layout nav bar\n- [x] Create worktree route page\n- [x] Tests

## Summary of Changes

Implemented full git worktree support:
- Go package `internal/worktree` with Manager (List/Create/Remove + pub/sub)
- GraphQL schema with Worktree type, query, mutations (createWorktree, removeWorktree), and worktreesChanged subscription
- Frontend WorktreeStore with real-time subscription via WebSocket
- "Start Work" button on BeanDetail (creates worktree + sets status to in-progress)
- Worktree tabs in nav bar linking to /worktree/[id] route
- Blank worktree page with bean detail pane
- SPA fallback in adapter-static config
