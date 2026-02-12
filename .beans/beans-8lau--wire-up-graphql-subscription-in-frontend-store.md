---
# beans-8lau
title: Wire up GraphQL subscription in frontend store
status: completed
type: feature
priority: normal
created_at: 2025-12-20T10:04:15Z
updated_at: 2025-12-20T10:07:28Z
---

Add real-time updates to BeansStore via GraphQL subscription.

## Changes
- Subscribe to beanChanged events on load
- Handle CREATED/UPDATED/DELETED events
- Update store state reactively

## Files
- frontend/src/lib/beans.svelte.ts