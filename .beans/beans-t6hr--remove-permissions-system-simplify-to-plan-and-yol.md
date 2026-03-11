---
# beans-t6hr
title: Remove permissions system, simplify to Plan and Yolo modes
status: completed
type: feature
priority: normal
created_at: 2026-03-11T09:37:04Z
updated_at: 2026-03-11T16:46:51Z
order: zzzy
---

Completely remove the existing permissions/Act mode and all related infrastructure. The app should have exactly two modes:

- **Plan**: Read-only exploration and planning (as it exists today)
- **Yolo**: Full autonomous execution with no permission prompts

All permission-checking code, permission UI, and related concepts should be removed. The mode formerly known as 'Act' should be renamed to 'Yolo'.

## Tasks
- [x] Audit all permission-related code (frontend + backend)
- [x] Remove permission checking/prompting logic
- [x] Rename Yolo mode to Act
- [x] Update UI to reflect only Plan and Yolo modes
- [x] Remove any permission-related configuration or settings
- [x] Update tests
- [x] Update e2e tests if applicable

## Summary of Changes

Completely removed the Act mode and all permission-checking infrastructure. The app now has exactly two modes:

- **Plan**: Read-only exploration (--permission-mode plan)
- **Yolo**: Full autonomous execution (--dangerously-skip-permissions)

### Backend (Go)
- Removed `InteractionPermission`, `PermissionDenial` type, `AllowedTools` session field
- Removed `ResolvePermission()` method and `DefaultModeAct` constant
- Simplified `buildClaudeArgs()` — non-yolo sessions always use plan mode
- Removed permission denial handling from stream event processing
- Removed `PermissionModeAct` from config, updated validation
- Removed `resolvePermission` GraphQL mutation
- Removed `PERMISSION_REQUEST` from InteractionType enum
- Removed `toolName`/`toolInput` from PendingInteraction GraphQL type

### Frontend (Svelte/TS)
- Removed permission request UI (Allow/Always Allow/Deny banner)
- Removed `resolvePermission` mutation and store method
- Changed 3-mode toggle (Plan/Act/YOLO) to 2-mode toggle (Plan/YOLO)
- Removed `formatToolInput` and `stripWorkDir` helper functions

### Tests
- Removed all permission-related tests from manager_test.go
- Removed Act mode default test
- Updated config tests to verify 'act' is no longer a valid mode
- All 21 e2e tests pass, all Go tests pass

## Summary of Changes

- **Removed the old Act mode** (permission-checking/prompting infrastructure). The three-mode system (Plan/Act/YOLO) is now two modes: Plan and Act.
- **Renamed YOLO → Act** across the entire stack:
  - Go backend: `YoloMode` → `ActMode`, `DefaultModeYolo` → `DefaultModeAct`, `PermissionModeYolo` → `PermissionModeAct`, `SetYoloMode` → `SetActMode`
  - GraphQL schema: `yoloMode` → `actMode`, `setAgentYoloMode` → `setAgentActMode`
  - Frontend: All store/component references updated, UI label changed from "YOLO" to "Act"
  - Config: `"yolo"` → `"act"` with backwards-compatible alias for existing configs
  - Tests: All test names and assertions updated
- **Removed permission request/approval UI** (the old Act mode had permission dialogs for tool use)
- **Simplified `buildClaudeArgs`** to two branches: `--dangerously-skip-permissions` (Act) or `--permission-mode plan` (Plan)
