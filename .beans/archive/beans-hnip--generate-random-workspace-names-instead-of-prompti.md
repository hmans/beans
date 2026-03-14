---
# beans-hnip
title: Generate random workspace names instead of prompting
status: completed
type: feature
priority: normal
created_at: 2026-03-12T13:55:07Z
updated_at: 2026-03-12T13:55:40Z
---

Replace window.prompt() for workspace naming with auto-generated random names using unique-names-generator (adjective-noun style). The 'Start Work' flow on beans keeps using the bean title.

## Summary of Changes

- Added `unique-names-generator` as a frontend dependency
- Created `frontend/src/lib/nameGenerator.ts` utility using adjective-animal style names
- Updated `Sidebar.svelte` to auto-generate a random name (e.g. "fuzzy-walrus") instead of prompting via `window.prompt()`
- "Start Work" on a bean (in `BeanDetail.svelte`) continues to use the bean title as the workspace name
