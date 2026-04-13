//go:build !windows

package terminal

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestCloseKillsProcessGroup(t *testing.T) {
	mgr := NewManager(nil)
	defer mgr.Shutdown()

	pidFile := filepath.Join(t.TempDir(), "child.pid")

	command := fmt.Sprintf(`sh -c 'echo $$ > %s; sleep 300' & wait`, pidFile)
	_, err := mgr.CreateWithCommand("test-pgkill", os.TempDir(), 80, 24, command)
	if err != nil {
		t.Fatalf("CreateWithCommand failed: %v", err)
	}

	var childPID int
	deadline := time.After(5 * time.Second)
	for {
		data, readErr := os.ReadFile(pidFile)
		if readErr == nil {
			trimmed := strings.TrimSpace(string(data))
			if trimmed != "" {
				childPID, err = strconv.Atoi(trimmed)
				if err == nil {
					break
				}
			}
		}
		select {
		case <-deadline:
			t.Fatal("timed out waiting for child PID file")
		default:
			time.Sleep(50 * time.Millisecond)
		}
	}

	if err := syscall.Kill(childPID, 0); err != nil {
		t.Fatalf("child process %d not alive before close: %v", childPID, err)
	}

	mgr.Close("test-pgkill")

	time.Sleep(500 * time.Millisecond)

	if err := syscall.Kill(childPID, 0); err == nil {
		t.Fatalf("child process %d still alive after session close", childPID)
	}
}

func TestCloseProcessGroupGracefulShutdown(t *testing.T) {
	mgr := NewManager(nil)
	defer mgr.Shutdown()

	// trap SIGTERM so the process exits cleanly on SIGTERM (not SIGKILL)
	sess, err := mgr.CreateWithCommand("test-pg-graceful", os.TempDir(), 80, 24,
		`trap 'exit 0' TERM; sleep 300`)
	if err != nil {
		t.Fatalf("CreateWithCommand failed: %v", err)
	}

	// Give the shell time to set up the trap
	time.Sleep(200 * time.Millisecond)

	mgr.Close("test-pg-graceful")

	select {
	case <-sess.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("session did not exit after SIGTERM")
	}
}
