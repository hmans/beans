package agent

import (
	"strings"
	"testing"
	"time"
)

func TestBuildDescribePrompt(t *testing.T) {
	prompt := buildDescribePrompt("Fix the auth bug")

	// Should contain the system prompt
	if !strings.Contains(prompt, "Summarize what this workspace will be doing") {
		t.Error("prompt should contain system instructions")
	}

	// Should include the user message
	if !strings.Contains(prompt, "Fix the auth bug") {
		t.Error("prompt should contain user message")
	}
}

func TestBuildDescribePromptTruncation(t *testing.T) {
	longContent := strings.Repeat("x", 600)

	prompt := buildDescribePrompt(longContent)

	// Should be truncated to 500 chars + "..."
	if strings.Contains(prompt, strings.Repeat("x", 501)) {
		t.Error("long messages should be truncated to 500 characters")
	}
	if !strings.Contains(prompt, strings.Repeat("x", 500)+"...") {
		t.Error("truncated messages should end with '...'")
	}
}

func TestCleanDescription(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{`"Fix auth token refresh bug"`, "Fix auth token refresh bug"},
		{`'Add dark mode to settings'`, "Add dark mode to settings"},
		{"  Refactor resolvers  \n", "Refactor resolvers"},
		{"No quotes here", "No quotes here"},
		{`""`, ""},
	}

	for _, tt := range tests {
		got := cleanDescription(tt.input)
		if got != tt.want {
			t.Errorf("cleanDescription(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSendMessageFirstUserMessageCallback(t *testing.T) {
	// Verify the onFirstUserMessage callback fires when SendMessage creates a new session.
	callbackCalled := make(chan string, 1)

	m := &Manager{
		sessions:    make(map[string]*Session),
		processes:   make(map[string]*runningProcess),
		subscribers: make(map[string][]chan struct{}),
		onFirstUserMessage: func(beanID string, message string) {
			callbackCalled <- message
		},
	}

	// SendMessage will try to spawn a process, which will fail since there's no
	// claude binary in test. But the callback should fire before spawning.
	_ = m.SendMessage("wt-test", "/tmp", "Fix the auth bug", nil)

	select {
	case msg := <-callbackCalled:
		if msg != "Fix the auth bug" {
			t.Errorf("expected message 'Fix the auth bug', got %q", msg)
		}
	case <-time.After(time.Second):
		t.Fatal("onFirstUserMessage callback was not called within timeout")
	}
}

func TestSendMessageNoCallbackOnExistingSession(t *testing.T) {
	// Verify the callback does NOT fire when the session already has user messages.
	callbackCalled := false

	m := &Manager{
		sessions:    make(map[string]*Session),
		processes:   make(map[string]*runningProcess),
		subscribers: make(map[string][]chan struct{}),
		onFirstUserMessage: func(beanID string, message string) {
			callbackCalled = true
		},
	}

	// Pre-create a session with an existing user message
	m.sessions["wt-test2"] = &Session{
		ID:           "wt-test2",
		AgentType:    "claude",
		Status:       StatusIdle,
		Messages:     []Message{{Role: RoleUser, Content: "Previous message"}},
		streamingIdx: -1,
	}

	_ = m.SendMessage("wt-test2", "/tmp", "Continue working", nil)

	// Give the goroutine a moment to fire (it shouldn't)
	time.Sleep(50 * time.Millisecond)

	if callbackCalled {
		t.Error("onFirstUserMessage should not be called when session already has user messages")
	}
}

func TestSendMessageCallbackFiresWithInfoOnlySession(t *testing.T) {
	// Verify the callback DOES fire when the session was pre-created by
	// AddInfoMessage (has info messages but no user messages). This is the
	// typical case: workspace setup completes and adds an info message before
	// the user sends their first message.
	callbackCalled := make(chan string, 1)

	m := &Manager{
		sessions:    make(map[string]*Session),
		processes:   make(map[string]*runningProcess),
		subscribers: make(map[string][]chan struct{}),
		onFirstUserMessage: func(beanID string, message string) {
			callbackCalled <- message
		},
	}

	// Pre-create a session with only info messages (simulates AddInfoMessage
	// from workspace setup completion)
	m.sessions["wt-test3"] = &Session{
		ID:           "wt-test3",
		AgentType:    "claude",
		Status:       StatusIdle,
		Messages:     []Message{{Role: RoleInfo, Content: "Workspace setup completed successfully."}},
		streamingIdx: -1,
	}

	_ = m.SendMessage("wt-test3", "/tmp", "Fix the auth bug", nil)

	select {
	case msg := <-callbackCalled:
		if msg != "Fix the auth bug" {
			t.Errorf("expected message 'Fix the auth bug', got %q", msg)
		}
	case <-time.After(time.Second):
		t.Fatal("onFirstUserMessage callback was not called for info-only session")
	}
}
