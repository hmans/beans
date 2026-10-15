---
# beans-f8mp
title: TUI focus key and GraphQL exposure for beans next
status: todo
type: task
priority: normal
created_at: 2026-10-14T13:34:41Z
updated_at: 2026-10-14T14:57:04Z
parent: beans-mmyp
blocked_by:
    - beans-p17z
---

Follow-up to beans-p17z (CLI only).

- TUI: a key on a selected milestone or epic sets `next.focus` to it, or clears `next.focus` when it already is the focus (`.beans.local.yml`).
- GraphQL: expose the ranked `next` list and the focus setting, so the web UI can use them.
