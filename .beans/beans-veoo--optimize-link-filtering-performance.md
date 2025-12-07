---
title: Optimize link filtering performance
status: done
created_at: 2025-12-07T15:22:06Z
updated_at: 2025-12-07T15:25:46Z
---


Refactor link filtering in cmd/list.go to eliminate redundant work:
- Build linkIndex (byID and targetedBy maps) once instead of twice
- Pre-parse filter strings into linkFilter structs
- Pass shared index to filterByLinkedAs and excludeByLinkedAs

## Checklist
- [ ] Add linkIndex struct and buildLinkIndex function
- [ ] Add linkFilter struct and parseLinkFilters function
- [ ] Refactor filterByLinks to use parsed filters
- [ ] Refactor filterByLinkedAs to use parsed filters and shared index
- [ ] Refactor excludeByLinks to use parsed filters
- [ ] Refactor excludeByLinkedAs to use parsed filters and shared index
- [ ] Update RunE call site to build index and parse filters once
- [ ] Update tests
- [ ] Run tests