package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewManagerHasNoAgentCommand(t *testing.T) {
	m := NewManager("", nil)
	if m.agentCommand != "" {
		t.Fatalf("NewManager must not configure an agent command, got %q", m.agentCommand)
	}
}

func TestSpawnWithoutAgentCommandFails(t *testing.T) {
	m := NewManager("", nil)

	if err := m.SendMessage("wt-none", t.TempDir(), "hello", nil); err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		s := m.GetSession("wt-none")
		if s.Status == StatusError {
			if !strings.Contains(s.Error, errNoAgentCommand.Error()) {
				t.Fatalf("session error = %q, want %q", s.Error, errNoAgentCommand)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("session status = %s, want %s", s.Status, StatusError)
		}
		time.Sleep(10 * time.Millisecond)
	}

	if got := GenerateQuickReplies("", "message", ""); got != nil {
		t.Errorf("GenerateQuickReplies without command = %v, want nil", got)
	}
	if got := GenerateDescription("", "message"); got != "" {
		t.Errorf("GenerateDescription without command = %q, want empty", got)
	}
}

// writeStubAgent writes an executable that records one line per invocation
// into the returned log file and prints output.
func writeStubAgent(t *testing.T, output string) (command, logFile string) {
	t.Helper()
	dir := t.TempDir()
	logFile = filepath.Join(dir, "calls.log")
	command = filepath.Join(dir, "stub-agent")
	script := "#!/bin/sh\necho called >> '" + logFile + "'\nprintf '%s' '" + output + "'\n"
	if err := os.WriteFile(command, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return command, logFile
}

func TestGeneratorsRunConfiguredCommand(t *testing.T) {
	command, logFile := writeStubAgent(t, "Yes\nNo\n")

	if got := GenerateQuickReplies(command, "message", ""); len(got) != 2 {
		t.Errorf("GenerateQuickReplies = %v, want the stub's two lines", got)
	}
	if got := GenerateDescription(command, "message"); got != "Yes\nNo" {
		t.Errorf("GenerateDescription = %q, want the stub's output", got)
	}

	calls, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("stub agent was not run: %v", err)
	}
	if n := strings.Count(string(calls), "called"); n != 2 {
		t.Errorf("stub agent ran %d times, want 2", n)
	}
}
