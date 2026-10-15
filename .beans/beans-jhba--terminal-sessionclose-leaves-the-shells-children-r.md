---
# beans-jhba
title: Terminal Session.Close leaves the shell's children running
status: todo
type: bug
priority: low
created_at: 2026-10-12T13:59:03Z
updated_at: 2026-10-12T13:59:03Z
---

`Session.Close` (`internal/terminal/terminal.go:181`) sends SIGKILL to the session's shell only. Processes the shell started stay alive. A Run session's command can thus outlive its Stop button, for example a server started by a script that does not handle SIGHUP.

The shell is started with `Setsid` (go-pty), so it leads its own session and process group, and signalling `-pid` reaches everything it started.

## Todo

- [ ] `Close` signals the process group (`syscall.Kill(-pid, ...)`), SIGTERM first, SIGKILL after a short grace period; no-op on non-Unix platforms
- [ ] Test: a session that starts a background `sleep`; after `Close` the `sleep` is gone
- [ ] `go test ./internal/terminal/`
