---
# beans-xwvt
title: Add justfile mirroring mise.toml tasks
status: completed
type: task
created_at: 2026-09-24T15:16:53Z
updated_at: 2026-09-24T15:16:53Z
---

Build without mise: a justfile that mirrors the tasks in mise.toml, using go, node and corepack from PATH.

## Summary of Changes

- Added `justfile` with recipes setup, deps, codegen, build-frontend, build-embed, build, install, test, test-e2e, beans, beans-serve, beans-tui, dev-frontend.
- pnpm runs as `corepack pnpm@10`: pnpm 11+ ignores `onlyBuiltDependencies` in frontend/pnpm-workspace.yaml and aborts the install.
- Not mirrored: dev/dev:serve (need watchexec via `mise watch`), release:* (need svu), dev:kill.
- Verified: `just setup` and `just build` succeed from a clean frontend/node_modules.
