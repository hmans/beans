---
# beans-7yh3
title: Add worktree.integrate config option (local/pr)
status: completed
type: feature
priority: normal
created_at: 2026-03-17T17:32:10Z
updated_at: 2026-03-17T17:35:42Z
---

Add a new config option worktree.integrate that controls whether the integration workflow uses local merging or PR-based workflow. When set to 'local', PR-related buttons are hidden. When set to 'pr', the Integrate button is hidden.

## Summary of Changes

- Added `IntegrateMode` type (`local` | `pr`) to `pkg/config/config.go`
- Added `Integrate` field to `WorktreeConfig` struct, defaulting to `local`
- Added `GetWorktreeIntegrate()` accessor method with fallback to `local` for invalid values
- Added `integrate` to YAML serialization in `toYAMLNode()` with descriptive comment
- Added `worktreeIntegrateMode` GraphQL query field and resolver
- Added field to frontend `Config` query operation and `ConfigStore`
- Updated `integrate` action visibility: hidden when mode is `pr`
- Updated `create-pr` action visibility: hidden when mode is `local`
- Updated `WorkspaceView.svelte`: PR link hidden when mode is `local`
- Added `IntegrateMode` to `actionContext` and populated it in `AgentActions` resolver
- Added unit tests for `GetWorktreeIntegrate` and load/save roundtrip
