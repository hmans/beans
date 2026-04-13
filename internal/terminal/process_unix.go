//go:build !windows

package terminal

import (
	"syscall"
	"time"

	gopty "github.com/aymanbagabas/go-pty"
)

const processGroupGrace = 3 * time.Second

func setProcessGroup(cmd *gopty.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// closeProcessGroup sends SIGTERM to the process group, waits for it to exit,
// and escalates to SIGKILL after the grace period.
func closeProcessGroup(pgid int, done <-chan struct{}) {
	_ = syscall.Kill(-pgid, syscall.SIGTERM)

	select {
	case <-done:
		return
	case <-time.After(processGroupGrace):
		_ = syscall.Kill(-pgid, syscall.SIGKILL)
	}
}
