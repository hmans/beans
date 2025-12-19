---
# beans-lbjp
title: beans web - Web UI server
status: draft
type: feature
priority: normal
tags:
    - idea
created_at: 2025-12-08T17:11:36Z
updated_at: 2025-12-13T14:44:14Z
parent: beans-f11p
---

Add a `beans web` command that starts a webserver providing a Beans web UI.

This would allow users to interact with their beans through a browser-based interface, making it easier to:
- View and browse all beans
- Create, edit, and update beans
- Visualize relationships between beans
- Filter and search beans

## Open Questions
- What web framework to use? (stdlib net/http, chi, echo, etc.)
- Should it support live reload/hot updates?
- Read-only mode vs full editing capabilities?
- What UI framework for the frontend? (htmx, templ, plain HTML, etc.)