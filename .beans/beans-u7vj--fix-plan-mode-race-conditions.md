---
# beans-u7vj
title: Fix plan mode race conditions
status: completed
type: bug
priority: normal
created_at: 2026-03-15T18:07:01Z
updated_at: 2026-03-15T19:21:24Z
---

Plan mode is wonky due to race conditions: (1) approveInteraction fires three concurrent mutations without awaiting, so sendMessage can arrive before mode changes causing plan-mode loops; (2) stale process events from dying processes can corrupt session status during auto-approved mode switches.

## Summary of Changes

Fixed multiple plan mode issues:
- Race condition in approveInteraction where three concurrent mutations could cause plan-mode loops
- Stale process events corrupting session status during mode switches
- Plan content not showing when Claude describes plan inline instead of writing a file
- File path truncation breaking plan file detection for long paths
