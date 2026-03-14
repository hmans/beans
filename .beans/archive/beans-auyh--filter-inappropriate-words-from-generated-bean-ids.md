---
# beans-auyh
title: Filter inappropriate words from generated bean IDs
status: completed
type: bug
priority: normal
created_at: 2026-03-13T15:56:30Z
updated_at: 2026-03-13T17:14:33Z
order: V
---

Bean IDs are 4-character nanoids from [a-z0-9]. This means offensive words like 'cock', 'nazi', 'dick', 'fuck' can appear as bean IDs. Add a blocklist check to NewID that regenerates if the ID contains a blocked substring.

## Tasks

- [x] Add blocklist of offensive words to id.go
- [x] Add retry loop in NewID when blocked word detected
- [x] Add tests for blocklist filtering
- [x] Run tests

## Summary of Changes

- Added a `blockedIDWords` list of offensive 3-4 character words to `pkg/bean/id.go`
- `NewID` now regenerates if the ID contains any blocked substring
- Added `TestContainsBlockedWord` table-driven tests and a fuzz test generating 10,000 IDs
