---
title: Display bean relationships in TUI detail view
status: done
type: feature
created_at: 2025-12-07T17:18:09Z
updated_at: 2025-12-07T17:19:34Z
---

Show linked beans in the TUI detail view with navigation support.

## Requirements
- Show outgoing links (beans this one links to)
- Show incoming links (beans that link to this one)
- Make links navigable (Enter to jump to linked bean)
- Hide links to deleted/missing beans

## Checklist
- [ ] Fix Description→Body bug in detail.go
- [ ] Add store to detailModel
- [ ] Create link resolution helpers
- [ ] Add link selection state
- [ ] Implement link navigation (j/k, Enter, Tab)
- [ ] Render relationships in header
- [ ] Adjust layout calculations