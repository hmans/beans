---
# beans-34gf
title: 'beans show: list blocked-by in text and JSON'
status: completed
type: feature
priority: normal
created_at: 2026-10-02T12:46:26Z
updated_at: 2026-10-02T13:19:00Z
---

`beans show` lists parent and blocking, but not the beans blocking this one. The JSON `blocked_by` holds only the bean's own front-matter field, so a blocker stored as `blocking` in the other bean's file is missing, although both are the same relation.

- [x] Text output: `blocked by:` line per direct blocker (front matter + incoming links), resolved ones marked with their status
- [x] JSON (`show --json`, `list --json`): `blocked_by` lists every direct blocker; the etag stays that of the file
- [x] Tests

## Summary of Changes

- Text: `formatRelationships` prints a `blocked by:` line per direct blocker from `CoreResolver.BeanBlockedBy`; a blocker with an archive status gets its status as a muted mark.
- JSON: new `output.Bean` view; `blocked_by` holds the IDs of all direct blockers (own field + incoming `blocking` links), resolved ones included. `show --json` and `list --json` use it via `jsonBeans`.
- The wrong etag of `list --json` without `--full` is fixed in beans-pl3m.
- Blocker IDs pointing to a missing bean no longer appear in JSON `blocked_by`, since `BeanBlockedBy` skips broken links.
- Tests: `TestFormatRelationships_BlockedBy`, `TestJSONBeansBlockedBy`, `TestBeanJSON`.
