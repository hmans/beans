---
# beans-8k9b
title: Runtime state layer for beans-serve
status: completed
type: feature
priority: normal
created_at: 2026-03-11T14:58:36Z
updated_at: 2026-03-11T16:46:51Z
order: F
---

Decouple beans-serve runtime state from disk persistence. Add dirty tracking, optional disk writes on mutations, worktree watching, and save mechanism.

## Plan

- [x] Step 1: Add dirty tracking to Core
- [x] Step 2: Make disk persistence optional on mutations
- [x] Step 3: Add save/persist mechanism + GraphQL
- [x] Step 4: Watch worktree bean directories
- [x] Step 5: Expose isDirty in GraphQL Bean type
- [x] Step 6: Clean up on worktree removal
