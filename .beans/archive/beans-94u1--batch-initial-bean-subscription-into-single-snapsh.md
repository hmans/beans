---
# beans-94u1
title: Batch initial bean subscription into single snapshot event
status: completed
type: task
priority: normal
created_at: 2026-03-12T11:32:18Z
updated_at: 2026-03-12T11:38:52Z
---

The BeanChanged subscription sends each bean as an individual INITIAL event during initial sync, causing performance issues with large bean counts. Replace with a single INITIAL_SNAPSHOT event carrying all beans at once.

## Summary of Changes

Replaced per-bean INITIAL + INITIAL_SYNC_COMPLETE subscription events with a single INITIAL_SNAPSHOT event that carries all beans in one message.

### Backend (GraphQL)
- Added `beans` list field to `BeanChangeEvent` type
- Replaced `INITIAL` and `INITIAL_SYNC_COMPLETE` change types with single `INITIAL_SNAPSHOT`
- Resolver now sends one event with all beans instead of looping

### Frontend
- Handle `INITIAL_SNAPSHOT` by creating a fresh `SvelteMap` in one pass (single reactivity trigger)
- Removed handling for removed event types

### Tests
- Updated `TestSubscriptionBeanChanged` to verify single snapshot event
