---
paths:
  - "pkg/bean/**"
  - "pkg/beancore/**"
  - "internal/graph/**"
  - "internal/commands/**"
---

# Bean Data Model

## Sorting Consistency

Bean sorting must be consistent across the entire system. All places that return lists of beans should use `bean.SortByStatusPriorityAndType()` from `internal/bean/sort.go`. The frontend mirrors this in `sortBeans()` in `frontend/src/lib/beans.svelte.ts`. If you change the sort order, update both.

The sort order is: status → manual order (fractional index) → priority → type → title (case-insensitive).

## Relationship Validation

When modifying bean relationships (parent, blocking, blocked-by):

- Always check for **cycles** using `Core.DetectCycle()` before adding a relationship
- Validate **type hierarchy** for parent relationships (e.g., a task can't be parent of a milestone)
- Normalize short IDs to full IDs via `Core.NormalizeID()` before storing
- Use the resolver helper methods (`validateAndSetParent`, `validateAndAddBlocking`, etc.) — don't inline this logic

## ETag Concurrency Control

Beans use content-hash ETags for optimistic concurrency:

- `bean.ETag()` computes a hash of the bean's content
- The `ifMatch` parameter on mutations validates the caller has seen the latest version
- `require_if_match` in config can make ETags mandatory for all mutations
- The frontend doesn't currently use ETags, but the CLI does (especially for body modifications)

## Dirty Beans

`beancore.Core` tracks beans modified at runtime but not yet written to disk:

- `dirty` map in Core tracks which bean IDs have unsaved changes
- Worktree bean changes use `WithPersist(false)` — they're runtime-only until the PR merges to main
- The `saveDirtyBeans` and `saveBean` mutations allow explicit persistence
- The `hasDirtyBeans` query lets the frontend show unsaved state indicators
