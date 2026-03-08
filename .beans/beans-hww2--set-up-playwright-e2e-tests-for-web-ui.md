---
# beans-hww2
title: Set up Playwright e2e tests for web UI
status: completed
type: task
priority: normal
created_at: 2026-03-08T11:37:25Z
updated_at: 2026-03-08T11:46:22Z
---

Add Playwright e2e tests with page objects to verify the web UI works correctly, including real-time sorting updates from filesystem changes.

## Summary of Changes\n\n- Added @playwright/test and created playwright.config.ts\n- Created page objects: BacklogPage (backlog list view) and BoardPage (kanban board view)\n- Created test fixtures with BeansCLI helper for CLI-driven test data setup\n- Added e2e/run.sh wrapper script for temp dir lifecycle management\n- 5 backlog tests: sorting, re-sort on priority/status change, new bean insertion, deletion\n- 4 board tests: column placement, priority sorting, status change moves column, priority re-sort\n- Added data-status attribute to BoardView columns for reliable test selectors\n- Added mise test:e2e task and pnpm test:e2e script\n- Also fixed frontend sorting: added sortBeans() to beans.svelte.ts matching backend sort logic
