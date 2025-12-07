---
title: Refactor links to array-based structure
status: done
created_at: 2025-12-07T15:06:13Z
updated_at: 2025-12-07T15:11:03Z
---


Change the links YAML format from nested map to array of single-key objects.

## Current format
```yaml
links:
  parent: abc
  blocks:
    - foo
    - bar
```

## New format
```yaml
links:
  - parent: abc
  - parent: xyz
  - blocks: foo
  - blocks: bar
```

## Checklist
- [ ] Add Link struct and Links type with custom YAML marshaling to bean.go
- [ ] Change Bean.Links type and update Parse/Render
- [ ] Update cmd/update.go link handling
- [ ] Update cmd/show.go formatLinks
- [ ] Update cmd/list.go filter functions
- [ ] Update all tests
- [ ] Run tests and verify