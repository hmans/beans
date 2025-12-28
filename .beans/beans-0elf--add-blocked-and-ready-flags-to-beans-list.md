---
# beans-0elf
title: Add --blocked and --ready flags to beans list
status: completed
type: feature
priority: normal
created_at: 2025-12-28T11:44:10Z
updated_at: 2025-12-28T11:46:51Z
---

Add convenience flags to beans list command:
- --blocked: alias for --is-blocked (beans that are blocked by others)
- --ready: actionable beans (not blocked, excludes completed/scrapped/draft)

This avoids adding new top-level commands while providing easy workflow shortcuts.

Refs: beans-7kb7, beans-8q44