//go:build windows

package terminal

import "os"

func killProcessGroup(pid int, _ <-chan struct{}) {
	if p, err := os.FindProcess(pid); err == nil {
		_ = p.Kill()
	}
}
