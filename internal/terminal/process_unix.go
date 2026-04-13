//go:build !windows

package terminal

import (
	"syscall"
	"time"
)

const processGroupGrace = 3 * time.Second

// killProcessGroup sends SIGTERM to the process group, then escalates to
// SIGKILL after the grace period. go-pty sets Setsid on every spawned command,
// so the PID is always the process group leader.
func killProcessGroup(pid int, done <-chan struct{}) {
	_ = syscall.Kill(-pid, syscall.SIGTERM)

	select {
	case <-done:
		return
	case <-time.After(processGroupGrace):
		_ = syscall.Kill(-pid, syscall.SIGKILL)
	}
}
