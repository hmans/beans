---
# beans-unmv
title: Split into multiple CLI executables (beans, beans-serve, beans-tui)
status: completed
type: task
priority: normal
created_at: 2026-03-08T10:55:35Z
updated_at: 2026-03-08T11:01:07Z
---

Restructure the repository to build three separate binaries sharing internal packages.

## Summary of Changes\n\n- Created `internal/version/` package for shared version variables\n- Moved `cmd/` to `internal/commands/` with exported registration functions\n- Created three `cmd/*/main.go` entrypoints: beans, beans-serve, beans-tui\n- Updated `mise.toml` build tasks for three binaries\n- Updated `.goreleaser.yaml` with three build configs\n- All tests pass, all binaries build and run correctly
