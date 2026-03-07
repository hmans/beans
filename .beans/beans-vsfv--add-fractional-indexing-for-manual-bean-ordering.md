---
# beans-vsfv
title: Add fractional indexing for manual bean ordering
status: completed
type: feature
priority: normal
created_at: 2026-03-07T23:02:43Z
updated_at: 2026-03-07T23:07:09Z
---

Implement fractional indexing so beans can be manually reordered on the board via drag-and-drop. Each bean gets an 'order' field in frontmatter. Moving a bean only writes one file.\n\n## Tasks\n\n- [x] Create internal/bean/fractional.go with fractional indexing\n- [x] Add Order field to Bean struct and frontmatter\n- [x] Update sort to use Order as primary key within status groups\n- [x] Add order field to GraphQL schema\n- [x] Run codegen\n- [x] Update frontend Bean interface and subscription\n- [x] Update BoardView drag-and-drop to compute order from neighbors\n- [x] Tests

## Summary of Changes

Implemented fractional indexing for manual bean ordering:
- `internal/bean/fractional.go` — base-62 fractional index generation (OrderBetween)
- `internal/bean/fractional_test.go` — comprehensive tests including stress tests
- Added `Order` field to Bean struct, frontmatter parsing, and rendering
- Sort now uses Order as primary key within each status group
- GraphQL schema updated with `order` field on Bean type and UpdateBeanInput
- Frontend `fractional.ts` — TypeScript port of the algorithm
- BoardView supports drag-and-drop reordering within and across columns with drop position indicators
