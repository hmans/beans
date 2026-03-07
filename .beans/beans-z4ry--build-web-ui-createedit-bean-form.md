---
# beans-z4ry
title: 'Build web UI: Create/edit bean form'
status: completed
type: task
priority: normal
created_at: 2025-12-18T16:45:45Z
updated_at: 2026-03-07T22:33:23Z
parent: beans-lbjp
---

Implement bean creation and editing forms.

## Tasks

- [x] Create BeanForm component for create/edit
- [x] Title input field
- [x] Type selector (dropdown with configured types)
- [x] Status selector
- [x] Priority selector
- [x] Tags input (comma-separated input)
- [x] Parent selector (dropdown with cycle prevention)
- [x] Blocking beans selector (deferred -- needs multi-select UI)
- [x] Markdown body editor (textarea, no preview yet)
- [x] Form validation
- [x] Submit via GraphQL mutation
- [x] Handle optimistic updates (via subscription)

## Design Notes

- Modal or slide-over for creation
- Could reuse for inline editing in detail view

## Summary of Changes\n\nImplemented BeanForm component (create + edit) as a DaisyUI modal dialog:\n\n- Title, type, status, priority fields\n- Parent selector with cycle prevention\n- Comma-separated tags input\n- Markdown body textarea\n- Form validation (title required)\n- Create/update via GraphQL mutations\n- Subscription handles reactive updates (no optimistic UI needed)\n- Edit button in BeanDetail header\n- "+ New Bean" button in the tab bar\n\nDeferred: Blocking beans multi-select (would benefit from a proper multi-select component).
