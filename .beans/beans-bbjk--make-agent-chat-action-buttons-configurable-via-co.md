---
# beans-bbjk
title: Make agent chat action buttons configurable via config file
status: scrapped
type: feature
priority: normal
created_at: 2026-03-11T18:10:06Z
updated_at: 2026-03-11T18:28:58Z
---

Replace hardcoded Commit and Review buttons in the Status pane with dynamic actions defined in the agent section of the config file. Add an actions subkey as an array with label and prompt fields. Make Commit and Review the defaults in beans init.

## Summary of Changes

### Backend (Go)
- Added `ActionConfig` struct and `DefaultActions` var to `pkg/config/config.go`
- Added `Actions []ActionConfig` field to `AgentConfig`  
- Added `GetActions()` accessor (returns defaults if none configured)
- Updated `Default()` to include default actions (Commit + Review)
- Updated `toYAMLNode()` to serialize actions in YAML output
- Added `AgentAction` type and `agentActions` query to GraphQL schema
- Implemented `AgentActions` resolver to read from config
- Added 5 new tests for actions functionality
- Fixed `TestSaveOmitsEmptyAgentSection` to clear actions too

### Frontend (Svelte)
- Created `agentActions.svelte.ts` store that fetches actions via GraphQL
- Replaced hardcoded Commit/Review buttons in `ChangesPane.svelte` with dynamic `{#each}` loop
- Store is fetched on app mount in `+layout.svelte`

### Config
- Updated `.beans.yml` with explicit actions section
- `beans init` now generates default actions in new projects


## Reasons for Scrapping

Reverted this feature in favor of hardcoding the Commit and Review buttons directly in the frontend. The configurability added unnecessary complexity (GraphQL type, query, resolver, store, config types) for minimal benefit. The button prompts were updated to be more detailed before re-hardcoding.
