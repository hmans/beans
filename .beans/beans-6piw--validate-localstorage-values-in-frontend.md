---
# beans-6piw
title: Validate localStorage values in frontend
status: todo
type: task
priority: low
created_at: 2026-03-09T17:02:04Z
updated_at: 2026-03-09T20:28:54Z
order: zzzV
parent: beans-oe8n
---

Several frontend components read localStorage without validating the stored values. SplitPane.svelte line 37 does parseInt() which can return NaN. Layout.ts reads view preferences without checking they match expected values. If localStorage is tampered with (or corrupted), the app could malfunction. Fix: (1) SplitPane: clamp parsed value between MIN_SIZE and MAX_SIZE, fallback to default on NaN. (2) Layout.ts: validate view value is one of the expected enum values ('backlog', 'board', etc.) before using it. (3) Any other localStorage reads should have similar guards. Low priority since this requires existing XSS or physical access to exploit.
