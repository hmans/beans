---
# beans-l4ag
title: Auto-reload config file on changes and push updates to frontend
status: todo
type: feature
priority: normal
created_at: 2026-03-11T18:12:28Z
updated_at: 2026-03-11T19:54:28Z
order: zzzV
blocked_by:
    - beans-bbjk
---

Make beans-serve (or beanscore) watch the .beans.yml config file for changes and automatically push updated config (like agent actions) to the frontend via GraphQL subscriptions. Currently config is only read at startup.
