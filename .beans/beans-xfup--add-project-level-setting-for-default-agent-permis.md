---
# beans-xfup
title: Add project-level setting for default agent permission mode
status: in-progress
type: feature
priority: normal
created_at: 2026-03-10T12:21:17Z
updated_at: 2026-03-10T16:42:30Z
---

Allow configuring the default permission mode (yolo/act/plan) for new agent sessions via a project-level setting (e.g. in .beans.yml). Currently hardcoded to yolo mode (see beans-ftqb). This would let teams choose their preferred default.

## Implementation Plan

- [ ] Add `AgentConfig` with `default_permission_mode` to config package
- [ ] Add validation and accessor for the new config field
- [ ] Update YAML serialization (Save/toYAMLNode)
- [ ] Pass default permission mode to agent.Manager via NewManager
- [ ] Use configured default instead of hardcoded `YoloMode: true`
- [ ] Update tests (config + agent manager)
- [ ] Run `mise codegen` if GraphQL changes needed (none expected)
- [ ] Run tests
