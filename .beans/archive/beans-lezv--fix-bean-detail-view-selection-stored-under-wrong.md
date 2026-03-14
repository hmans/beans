---
# beans-lezv
title: Fix bean detail view selection stored under wrong workspace key
status: completed
type: bug
priority: normal
created_at: 2026-03-14T09:25:29Z
updated_at: 2026-03-14T09:26:14Z
---

When clicking a bean in the sidebar under a workspace you're not currently viewing, the selection gets stored under the current view's key instead of the target workspace's key. This causes the bean detail pane to be empty after navigation.

## Summary of Changes

Fixed a race condition in `Sidebar.svelte` where clicking a bean listed under a non-active workspace stored the selection under the wrong view key.

### Root Cause

`ui.navigateTo(item.id)` calls `goto()` (async), but `ui.selectBeanById(wtBean.id)` ran synchronously before `activeView` updated, storing the bean ID under the *current* view's key instead of the *target* workspace's key. When navigation completed, `syncFromUrl()` found no bean for the new view and cleared the URL `?bean=` param.

### Fix

- Added `selectBeanForView(beanId, view)` method to `UIState` that stores a bean selection directly under a specific view's key
- Updated `Sidebar.svelte` to call `selectBeanForView` before `navigateTo`, so the selection is in the correct slot when `syncFromUrl` fires after navigation
