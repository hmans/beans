---
# beans-b175
title: 'okf-format: make it a project-independent documentation concept'
status: completed
type: task
priority: normal
created_at: 2026-10-02T06:23:00Z
updated_at: 2026-10-02T06:23:36Z
---

Rewrite okf/okf-format.md without references to the bundle it was written for (feed plugins, src/feed paths), as the general concept of keeping a project's documentation as an OKF bundle.

## Todo
- [x] Rewrite okf/okf-format.md
- [x] Re-sync okf/index.md entry

## Summary of Changes

- okf/okf-format.md: no references to a specific bundle; new section "The bundle in a project" (one directory, concepts, subdirectories, current state vs. history); generic examples for concept ID, type, resource, sources, footnote and log; description, tags and generated updated.
- okf/index.md: entry re-synced to the new description.
