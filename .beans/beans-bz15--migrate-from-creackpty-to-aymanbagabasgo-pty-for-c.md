---
# beans-bz15
title: Migrate from creack/pty to aymanbagabas/go-pty for cross-platform PTY support
status: completed
type: task
priority: normal
created_at: 2026-03-20T17:50:59Z
updated_at: 2026-03-20T17:53:49Z
---

Replace creack/pty (Unix-only, unmaintained) with aymanbagabas/go-pty which provides a unified API for Unix PTY and Windows ConPTY. Only internal/terminal/terminal.go needs changes.

## Summary of Changes

Replaced `creack/pty` with `aymanbagabas/go-pty` in `internal/terminal/terminal.go`:
- Swapped `*os.File` PTY handle for `gopty.Pty` interface (`io.ReadWriteCloser` + `Resize`)
- Swapped `*exec.Cmd` for `*gopty.Cmd` (same API: `Process`, `Start`, `Wait`)
- Added `defaultShell()` helper with Windows support (`pwsh.exe`/`cmd.exe` fallback)
- All existing tests pass, no API changes to `Session` or `Manager`
