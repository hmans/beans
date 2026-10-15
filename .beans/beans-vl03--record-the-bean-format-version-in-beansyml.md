---
# beans-vl03
title: Record the bean format version in .beans.yml
status: completed
type: feature
priority: normal
created_at: 2026-10-02T14:59:41Z
updated_at: 2026-10-02T15:00:58Z
---

Write a top-level `format_version` into `.beans.yml` so a future updater can tell which data format a project uses. No check reacts to it yet.

v0.3.0 is the last breaking format change (links → parent/blocking, backlog → draft); later releases only added fields (blocked_by, order).

- [x] FormatVersion constant and Config field in pkg/config
- [x] beans init writes it; Load keeps a missing field empty
- [x] Tests
- [x] Add format_version to this repo's .beans.yml

## Summary of Changes

- `config.FormatVersion = "v0.3.0"` and a top-level `Config.FormatVersion` field (`format_version` in `.beans.yml`)
- `Default()` sets it, so `beans init` writes it; `Load` leaves it empty when the file has none, so an empty value means the format is unknown
- `Save` writes the field first, with a comment, and omits it when empty
- Tests: load with a missing or older version, save writes and round-trips it, save omits an empty one
- Added `format_version: v0.3.0` to the project's own `.beans.yml`
