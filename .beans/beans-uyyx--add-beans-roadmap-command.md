---
title: Add 'beans roadmap' command
status: done
type: feature
created_at: 2025-12-08T07:33:48Z
updated_at: 2025-12-08T07:36:55Z
---

Add a `beans roadmap` command that generates a Markdown roadmap document, structured by milestones and epics.

## Behavior

- Uses `parent:` links to determine hierarchy
- Milestones at top level, epics nested within, work items under epics
- 'Other' section for items directly under milestone (no epic)
- 'Unscheduled' section for epics without a milestone parent
- Done items excluded by default (`--include-done` to show)
- Items sorted by status (in-progress first)
- Supports `--json` for structured output

## Checklist

- [x] Create `cmd/roadmap.go` with command structure and flags
- [x] Implement `buildRoadmap()` grouping logic
- [x] Implement Markdown output rendering
- [x] Implement JSON output
- [x] Add milestone filtering (--status, --no-status)
- [x] Add tests in `cmd/roadmap_test.go`
- [ ] Create changie entry (skipped - changie not configured)