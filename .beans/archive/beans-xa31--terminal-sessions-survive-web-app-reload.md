---
# beans-xa31
title: Terminal sessions survive web app reload
status: completed
type: feature
priority: normal
created_at: 2026-03-12T12:32:19Z
updated_at: 2026-03-12T12:35:55Z
---

Decouple PTY lifecycle from WebSocket lifecycle so terminal sessions persist across page reloads. Add scrollback ring buffer for state replay on reconnect.

## Summary of Changes

### Backend (internal/terminal/terminal.go)
- Added `RingBuffer` — a 64KB circular buffer that captures all PTY output for scrollback replay
- Sessions now run a background `readLoop` goroutine that reads from PTY, writes to scrollback buffer, and forwards to any attached client channel
- Added `Attach()` / `Detach()` methods for clients to connect/disconnect from a session's output stream
- Added `GetOrCreate()` method on Manager that reuses an existing alive session (with resize) instead of always spawning a new shell
- Added `Alive()` and `Done()` methods for lifecycle introspection

### Backend (internal/commands/terminal_handler.go)
- Handler now calls `GetOrCreate()` instead of `Create()` — existing sessions are reused on reconnect
- Removed `defer termMgr.Close()` — PTY sessions outlive WebSocket connections
- On reconnect, scrollback buffer is replayed to the client before live output starts
- Shell exit sends close code 1000 with reason "shell exited" so the frontend can distinguish it from connection loss

### Frontend (TerminalPane.svelte)
- Added auto-reconnect on unexpected WebSocket close (500ms delay)
- Terminal is reset before reconnect so scrollback replay starts clean
- "shell exited" close events still show `[session ended]` without reconnecting
- Reconnect timeout is cleaned up on component destroy

### Tests
- Added ring buffer tests (basic, wrap, overflow, empty)
- Added `TestSessionWriteAndAttach` (replaces old `TestSessionWriteAndRead`)
- Added `TestSessionScrollback` — verifies scrollback survives detach/reattach
- Added `TestGetOrCreateReusesAliveSession` and `TestGetOrCreateReplacesDeadSession`
- Added `TestSessionAlive` lifecycle test
