---
# beans-ftqb
title: Default to yolo mode
status: in-progress
type: feature
priority: normal
created_at: 2026-03-10T12:03:01Z
updated_at: 2026-03-10T12:20:01Z
---

Currently, agent chat sessions start in **act mode** (`--permission-mode acceptEdits`) by default, requiring explicit user opt-in to switch to yolo mode (`--dangerously-skip-permissions`). This feature would make yolo mode the default, so new agent sessions run fully autonomously without requiring the user to toggle it on each time.


## Context

- Agent sessions are spawned in `internal/agent/claude.go` with permission flags based on `Session.YoloMode` / `Session.PlanMode`
- The current default is act mode (`acceptEdits`), with yolo mode toggled via `SetYoloMode` in the manager
- The UI toggle lives in `AgentChat.svelte`

## Tasks

- [ ] Change the default `YoloMode` on new sessions to `true`
- [ ] Update the UI toggle so it reflects the new default
- [ ] Consider making this configurable (e.g. a project-level setting)
- [ ] Update tests in `manager_test.go` to reflect the new default
