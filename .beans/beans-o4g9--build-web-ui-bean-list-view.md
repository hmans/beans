---
# beans-o4g9
title: 'Build web UI: Bean list view'
status: completed
type: task
priority: normal
created_at: 2025-12-18T16:45:29Z
updated_at: 2026-03-09T20:28:02Z
order: V1y
parent: beans-lbjp
---

Implement the main bean listing interface in SvelteKit.

## Tasks

- [ ] Set up GraphQL client (urql or Apollo)
- [ ] Create BeanList component with filtering/sorting
- [ ] Implement status filter (tabs or dropdown)
- [ ] Implement type filter
- [ ] Implement text search
- [ ] Display bean cards with title, status, type, priority badges
- [ ] Add keyboard navigation
- [ ] Subscribe to live updates and refresh list

## Design Notes

- Keep it simple and fast
- Consider a table view vs card view (or both)
- Show parent/child relationships visually

## Summary of Changes

Bean list view was implemented across multiple components: BeanItem, BeanCard, BeanPane, BoardView, with filtering, sorting, keyboard navigation, and live GraphQL subscriptions. Task list was not kept current but all functionality is shipped.
