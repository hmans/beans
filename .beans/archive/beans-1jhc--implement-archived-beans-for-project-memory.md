---
# beans-1jhc
title: Implement archived beans for project memory
status: completed
type: feature
priority: normal
created_at: 2025-12-27T17:53:18Z
updated_at: 2025-12-27T18:08:15Z
---

Use beans as project memory by archiving completed/scrapped beans instead of deleting them.

## Overview
- Introduce `.beans/archive/` subdirectory (flat structure)
- `beans archive` moves beans there instead of deleting
- CLI only loads non-archived beans by default
- Add `--with-archived` global flag to include archived beans
- `beans update` on an archived bean unarchives it (moves back to `.beans/`)
- Update agent prompt to teach about using archived beans for context

## Checklist
- [x] Modify bean loading to exclude `.beans/archive/` by default
- [x] Add `--with-archived` global flag to root command
- [x] Update `beans archive` to move files instead of delete
- [x] Handle unarchiving when `beans update` targets an archived bean
- [x] Update file watcher to conditionally watch archive directory
- [x] Update agent prompt primer with guidance on using archived beans
- [x] Add tests for archive functionality