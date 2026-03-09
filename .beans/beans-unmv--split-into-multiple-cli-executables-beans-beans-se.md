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

## Summary of Changes

- Created `internal/version/` package for shared version variables
- Moved `cmd/` to `internal/commands/` with exported registration functions
- Created three `cmd/*/main.go` entrypoints: beans, beans-serve, beans-tui
- Updated `mise.toml` build tasks for three binaries
- Updated `.goreleaser.yaml` with three build configs
- All tests pass, all binaries build and run correctly
