---
# beans-1ae4
title: Filter and edit custom properties
status: draft
type: feature
tags:
    - idea
created_at: 2026-10-06T11:44:38Z
updated_at: 2026-10-06T11:44:38Z
---

Follow-up to beans-digu, which added string key-value properties (model, GraphQL, CLI, read-only display in TUI and UI).

Open scope:

- Filter by property in `BeanFilter` (e.g. `hasProperty`, `property: {key, value}`) and `beans list` flags
- Decide whether property values go into the Bleve full-text index
- Edit properties in the Beans UI detail view
- Decide whether adding `properties` warrants bumping `config.FormatVersion`: older binaries drop the key on write
