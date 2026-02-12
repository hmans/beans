---
# beans-4ect
title: Channel-based file watcher with typed events
status: completed
type: feature
created_at: 2025-12-20T09:50:06Z
updated_at: 2025-12-20T09:50:06Z
---

Upgrade the file watcher from callback-based to channel-based pub/sub system.

## Changes
- Add EventType (Created, Updated, Deleted) and BeanEvent types
- Implement Subscribe() returning event channel + unsubscribe func
- Fan-out pattern: multiple subscribers each get their own buffered channel
- Non-blocking sends prevent slow subscribers from blocking others
- Add StartWatching() as preferred API (Watch() kept for compat)
- Migrate TUI to use channel-based subscription

## Files
- internal/beancore/watcher.go
- internal/beancore/core.go
- internal/tui/tui.go
- internal/beancore/core_test.go