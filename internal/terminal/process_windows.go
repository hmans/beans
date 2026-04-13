//go:build windows

package terminal

import gopty "github.com/aymanbagabas/go-pty"

func setProcessGroup(_ *gopty.Cmd) {}

func closeProcessGroup(_ int, _ <-chan struct{}) {}
