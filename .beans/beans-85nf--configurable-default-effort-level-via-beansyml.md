---
# beans-85nf
title: Configurable default effort level via .beans.yml
status: completed
type: feature
priority: normal
created_at: 2026-03-17T19:14:11Z
updated_at: 2026-03-18T06:53:58Z
---

Add agent.default_effort config option to .beans.yml to set the project-level default thinking effort for new agent sessions

## Context
The agent composer UI hardcodes "high" as the visual default effort level
(`effort === 'high' || !effort` in `AgentComposer.svelte:290`). There is no
`.beans.yml` config option to set a project-level default. Users who prefer a
lower effort level must change it manually each session.

The backend already supports per-session effort via `session.Effort` and the
`setAgentEffort` mutation — it just has no way to seed that value from config.

## Higher Goal
Effort level has cost and latency implications. Projects (or users) should be
able to express a preferred default rather than being silently locked into
"high" for every new session.

## Acceptance Criteria
- [x] `agent.default_effort` field added to `.beans.yml` config (values: `low`, `medium`, `high`, `max`; omitting the field preserves current behavior)
- [x] New agent sessions are initialized with `session.Effort` set to the configured default (so `--effort` is passed to the Claude CLI from the start)
- [x] The UI effort selector reflects the actual session effort rather than hardcoding `high` as the fallback

## Out of Scope
- Per-user (rather than per-project) default effort
- Changing the effort level mid-session behavior (already works)

## Summary of Changes

- Added `agent.default_effort` config field to `AgentConfig` with `GetDefaultEffort()` validation method
- Extracted `newBaseSession()` helper in agent manager that applies both default mode and default effort to all newly created sessions (covering `AddInfoMessage`, `SetPlanMode`, `SetActMode`, `loadOrCreateSession`, and the disk-restore path in `GetSession`)
- Wired `cfg.GetDefaultEffort()` into `serve.go` via `agentMgr.SetDefaultEffort()`
- Fixed `AgentComposer.svelte` to remove the `|| !effort` fallback so the UI accurately reflects actual session effort
