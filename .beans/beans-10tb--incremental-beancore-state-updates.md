---
# beans-10tb
title: Incremental beancore state updates
status: completed
type: feature
created_at: 2025-12-20T09:50:14Z
updated_at: 2025-12-20T09:50:14Z
---

Update beancore state incrementally instead of full reload on file changes.

## Changes
- Accumulate fsnotify events during debounce window
- Process only affected files instead of reloading all beans
- Update search index incrementally per-bean
- Handle edge cases: rapid updates, create+delete, invalid files

## Files
- internal/beancore/watcher.go
- internal/beancore/core_test.go