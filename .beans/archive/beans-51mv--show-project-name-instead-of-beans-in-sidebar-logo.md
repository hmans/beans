---
# beans-51mv
title: Show project name instead of 'beans' in sidebar logo
status: completed
type: feature
priority: normal
created_at: 2026-03-12T11:29:40Z
updated_at: 2026-03-12T12:00:31Z
---

The upper-left corner of the sidebar currently shows the hardcoded text 'beans'. It should instead show the name of the project being tracked. This likely involves:

1. Determining the project name (e.g. from the directory name of the beans root, or a config option)
2. Exposing it via the GraphQL API (e.g. a `projectName` query)
3. Updating the Sidebar.svelte component to fetch and display the project name

Location: `frontend/src/lib/components/Sidebar.svelte:45`

## Plan

- [x] Add `Project` section with `Name` field to config struct
- [x] Initialize `project.name` in `beans init` using directory name heuristic
- [x] Include `project.name` in config save/template
- [x] Expose `projectName` via GraphQL
- [x] Update frontend ConfigStore and Sidebar to display project name
- [x] Write tests

## Summary of Changes

- Added `ProjectConfig` struct with `Name` field to `pkg/config/config.go`
- Added `project` section to `Config` struct (appears first in YAML output)
- `beans init` now sets `project.name` to the directory name (same heuristic as prefix)
- `Save`/`toYAMLNode` serialize the project section with comments
- Added `GetProjectName()` accessor method
- Added `projectName` query to GraphQL schema
- Implemented `ProjectName` resolver
- Updated frontend `ConfigStore` to fetch `projectName` alongside `agentEnabled`
- Updated `Sidebar.svelte` to display project name (falls back to "beans")
- Added unit tests for load, save, and omit-when-empty behavior
