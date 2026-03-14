---
# beans-d92r
title: Add workflow action buttons to bean detail view
status: completed
type: feature
priority: normal
created_at: 2026-03-12T12:32:44Z
updated_at: 2026-03-12T12:48:05Z
---

Add status transition buttons to BeanDetail: draft→Todo/Scrap, todo→Start Work/Scrap, in-progress→Complete/Scrap, completed/scrapped→nothing

## Summary of Changes

Added workflow action buttons to BeanDetail.svelte that show context-appropriate status transitions:
- draft: Todo (sky-600), Scrap (danger)
- todo: Start Work (success, existing behavior), Scrap (danger)
- in-progress: Complete (success), Scrap (danger)
- completed/scrapped: no workflow buttons (only Archive + Edit)

Uses optimistic updates with rollback on error, consistent with existing patterns in dragOrder.ts.

## Summary of Changes

Added workflow action buttons to BeanDetail.svelte with data-driven approach and full e2e test coverage.
