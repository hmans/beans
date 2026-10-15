---
# beans-pl3m
title: list --json without --full reports a wrong etag
status: completed
type: bug
priority: normal
created_at: 2026-10-02T13:16:13Z
updated_at: 2026-10-02T13:19:00Z
---

`beans list --json` without `--full` clears each bean's body before printing it, and the printed etag is computed at print time. It is therefore the hash of the body-less bean, not of the file, so `beans update --if-match <etag>` with that value fails with a conflict for every bean with a body. Example: `anvl-41r7` printed `c07623cba2d921a8` from `list --json`, `33cc928c901220c7` from `list --json --full` and `show --json`.

- [x] Take the etag from the full bean, and omit the body in the JSON view instead of clearing it in the core
- [x] Regression test on the `list --json` path, failing with the old order

## Summary of Changes

- Root cause: `list --json` set `b.Body = ""` on the beans that `resolver.Beans` returns. These are the core's own in-memory beans, so the printed etag was the hash of the body-less bean, and the body was also gone from the core for every later read in the same process.
- `output.NewBean(b, blockedBy, withBody)` takes the etag from the full bean and, without `withBody`, prints a body-less copy; the bean in the core stays unchanged. `list --json` passes `listFull`, `show --json` passes `true`.
- Regression test `TestListJSONETag` runs `listCmd` with and without `--full` and compares etag and body with the file. It fails when the body is cleared in the core as before. `TestBeanJSON` checks that `NewBean` leaves the bean unchanged.
