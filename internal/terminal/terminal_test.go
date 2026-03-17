package terminal

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestRingBufferBasic(t *testing.T) {
	rb := NewRingBuffer(8)

	rb.Write([]byte("hello"))
	got := string(rb.Bytes())
	if got != "hello" {
		t.Fatalf("expected %q, got %q", "hello", got)
	}
}

func TestRingBufferWrap(t *testing.T) {
	rb := NewRingBuffer(8)

	rb.Write([]byte("abcdefgh"))  // fills exactly
	rb.Write([]byte("ij"))        // wraps: should be "cdefghij"

	got := string(rb.Bytes())
	if got != "cdefghij" {
		t.Fatalf("expected %q, got %q", "cdefghij", got)
	}
}

func TestRingBufferOverflow(t *testing.T) {
	rb := NewRingBuffer(4)

	// Write more than capacity in a single call
	rb.Write([]byte("abcdefgh"))
	got := string(rb.Bytes())
	if got != "efgh" {
		t.Fatalf("expected %q, got %q", "efgh", got)
	}
}

func TestRingBufferEmpty(t *testing.T) {
	rb := NewRingBuffer(8)
	if rb.Bytes() != nil {
		t.Fatal("expected nil for empty buffer")
	}
}

func TestManagerCreateAndClose(t *testing.T) {
	mgr := NewManager(nil)
	defer mgr.Shutdown()

	sess, err := mgr.Create("test-1", os.TempDir(), 80, 24)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if sess == nil {
		t.Fatal("session is nil")
	}

	// Verify we can get the session
	got := mgr.Get("test-1")
	if got != sess {
		t.Fatal("Get returned different session")
	}

	// Close should remove it
	mgr.Close("test-1")
	if mgr.Get("test-1") != nil {
		t.Fatal("session still exists after Close")
	}
}

func TestManagerCreateReplacesExisting(t *testing.T) {
	mgr := NewManager(nil)
	defer mgr.Shutdown()

	sess1, err := mgr.Create("test-replace", os.TempDir(), 80, 24)
	if err != nil {
		t.Fatalf("first Create failed: %v", err)
	}

	sess2, err := mgr.Create("test-replace", os.TempDir(), 80, 24)
	if err != nil {
		t.Fatalf("second Create failed: %v", err)
	}

	if sess1 == sess2 {
		t.Fatal("expected different session objects")
	}

	got := mgr.Get("test-replace")
	if got != sess2 {
		t.Fatal("Get should return the new session")
	}
}

func TestSessionResize(t *testing.T) {
	mgr := NewManager(nil)
	defer mgr.Shutdown()

	sess, err := mgr.Create("test-resize", os.TempDir(), 80, 24)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := sess.Resize(120, 40); err != nil {
		t.Fatalf("Resize failed: %v", err)
	}
}

func TestSessionWriteAndAttach(t *testing.T) {
	mgr := NewManager(nil)
	defer mgr.Shutdown()

	sess, err := mgr.Create("test-io", os.TempDir(), 80, 24)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Attach to receive output
	_, output := sess.Attach()

	// Write a command to the PTY
	_, err = sess.Write([]byte("echo hello\n"))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Read output via the attached channel
	select {
	case data := <-output:
		if len(data) == 0 {
			t.Error("received empty data")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for output")
	}
}

func TestSessionScrollback(t *testing.T) {
	mgr := NewManager(nil)
	defer mgr.Shutdown()

	sess, err := mgr.Create("test-scrollback", os.TempDir(), 80, 24)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Write a command and wait for output
	_, output := sess.Attach()
	_, _ = sess.Write([]byte("echo scrollback-test\n"))

	// Wait for some output
	select {
	case <-output:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for output")
	}

	// Detach and reattach — scrollback should contain the output
	sess.Detach(output)
	scrollback, _ := sess.Attach()

	if len(scrollback) == 0 {
		t.Fatal("expected non-empty scrollback after reattach")
	}
}

func TestGetOrCreateReusesAliveSession(t *testing.T) {
	mgr := NewManager(nil)
	defer mgr.Shutdown()

	sess1, reconnected, err := mgr.GetOrCreate("test-reuse", os.TempDir(), 80, 24)
	if err != nil {
		t.Fatalf("first GetOrCreate failed: %v", err)
	}
	if reconnected {
		t.Fatal("first call should not be a reconnection")
	}

	sess2, reconnected, err := mgr.GetOrCreate("test-reuse", os.TempDir(), 80, 24)
	if err != nil {
		t.Fatalf("second GetOrCreate failed: %v", err)
	}
	if !reconnected {
		t.Fatal("second call should be a reconnection")
	}
	if sess1 != sess2 {
		t.Fatal("expected same session object on reconnection")
	}
}

func TestGetOrCreateReplacesDeadSession(t *testing.T) {
	mgr := NewManager(nil)
	defer mgr.Shutdown()

	sess1, _, err := mgr.GetOrCreate("test-dead", os.TempDir(), 80, 24)
	if err != nil {
		t.Fatalf("first GetOrCreate failed: %v", err)
	}

	// Kill the shell process to simulate death
	sess1.Close()

	// Wait for done channel to close
	select {
	case <-sess1.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for session to die")
	}

	sess2, reconnected, err := mgr.GetOrCreate("test-dead", os.TempDir(), 80, 24)
	if err != nil {
		t.Fatalf("second GetOrCreate failed: %v", err)
	}
	if reconnected {
		t.Fatal("should not reconnect to dead session")
	}
	if sess1 == sess2 {
		t.Fatal("expected different session after dead replacement")
	}
}

func TestSessionAlive(t *testing.T) {
	mgr := NewManager(nil)
	defer mgr.Shutdown()

	sess, err := mgr.Create("test-alive", os.TempDir(), 80, 24)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if !sess.Alive() {
		t.Fatal("new session should be alive")
	}

	sess.Close()

	// Wait for done
	select {
	case <-sess.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for done")
	}

	if sess.Alive() {
		t.Fatal("closed session should not be alive")
	}
}

func TestManagerShutdown(t *testing.T) {
	mgr := NewManager(nil)

	_, err := mgr.Create("test-shutdown-1", os.TempDir(), 80, 24)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	_, err = mgr.Create("test-shutdown-2", os.TempDir(), 80, 24)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	mgr.Shutdown()

	if mgr.Get("test-shutdown-1") != nil {
		t.Fatal("session 1 still exists after Shutdown")
	}
	if mgr.Get("test-shutdown-2") != nil {
		t.Fatal("session 2 still exists after Shutdown")
	}
}

func TestEnvFuncInjection(t *testing.T) {
	mgr := NewManager(func(sessionID string) []string {
		return []string{"BEANS_WORKSPACE_PORT=44000"}
	})
	defer mgr.Shutdown()

	sess, err := mgr.Create("test-env", os.TempDir(), 80, 24)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Attach and ask the shell to echo the env var
	_, output := sess.Attach()
	_, err = sess.Write([]byte("echo PORT=$BEANS_WORKSPACE_PORT\n"))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Collect output for a short time looking for the PORT= line
	deadline := time.After(5 * time.Second)
	var collected []byte
	for {
		select {
		case data := <-output:
			collected = append(collected, data...)
			if containsSubstring(collected, "PORT=44000") {
				return // success
			}
		case <-deadline:
			t.Fatalf("timed out; collected output: %q", string(collected))
		}
	}
}

func containsSubstring(data []byte, sub string) bool {
	return len(data) >= len(sub) && strings.Contains(string(data), sub)
}

