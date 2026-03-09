---
# beans-nix5
title: Split into multiple CLI executables (beans, beans-serve, beans-tui)
status: scrapped
type: task
priority: normal
created_at: 2026-03-08T10:55:32Z
updated_at: 2026-03-08T11:01:01Z
---

Restructure the repository to build three separate binaries sharing internal packages. beans for core CLI, beans-serve for GraphQL+web, beans-tui for terminal UI.

## Reasons for Scrapping

Duplicate of beans-unmv.
