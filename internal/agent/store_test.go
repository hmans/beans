package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewStore_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	beansDir := filepath.Join(dir, ".beans")
	if err := os.MkdirAll(beansDir, 0o755); err != nil {
		t.Fatal(err)
	}

	s, err := newStore(beansDir)
	if err != nil {
		t.Fatalf("newStore: %v", err)
	}

	// Conversations dir should exist
	convDir := filepath.Join(beansDir, ".conversations")
	if info, err := os.Stat(convDir); err != nil || !info.IsDir() {
		t.Fatalf(".conversations dir not created")
	}

	_ = s // used
}

func TestStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s, err := newStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	beanID := "bean-abc"

	// Initially empty
	msgs, sessionID, err := s.load(beanID)
	if err != nil {
		t.Fatalf("load empty: %v", err)
	}
	if len(msgs) != 0 || sessionID != "" {
		t.Fatalf("expected empty, got %d messages, sessionID=%q", len(msgs), sessionID)
	}

	// Append messages
	if err := s.appendMessage(beanID, Message{Role: RoleUser, Content: "hello"}); err != nil {
		t.Fatalf("append user: %v", err)
	}
	if err := s.appendMessage(beanID, Message{Role: RoleAssistant, Content: "Hi there!"}); err != nil {
		t.Fatalf("append assistant: %v", err)
	}
	if err := s.saveSessionID(beanID, "session-123"); err != nil {
		t.Fatalf("save session id: %v", err)
	}

	// Load back
	msgs, sessionID, err = s.load(beanID)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].Role != RoleUser || msgs[0].Content != "hello" {
		t.Errorf("msg[0] = %+v, want user/hello", msgs[0])
	}
	if msgs[1].Role != RoleAssistant || msgs[1].Content != "Hi there!" {
		t.Errorf("msg[1] = %+v, want assistant/Hi there!", msgs[1])
	}
	if sessionID != "session-123" {
		t.Errorf("sessionID = %q, want %q", sessionID, "session-123")
	}
}

func TestStoreSessionIDUpdates(t *testing.T) {
	dir := t.TempDir()
	s, err := newStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	beanID := "bean-xyz"

	// Save two different session IDs — last one wins on load
	s.saveSessionID(beanID, "old-session")
	s.saveSessionID(beanID, "new-session")

	_, sessionID, _ := s.load(beanID)
	if sessionID != "new-session" {
		t.Errorf("sessionID = %q, want %q", sessionID, "new-session")
	}
}

func TestStoreSkipsMalformedLines(t *testing.T) {
	dir := t.TempDir()
	s, err := newStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	beanID := "bean-bad"

	// Write a valid message, then garbage, then another valid message
	f, _ := os.Create(s.path(beanID))
	f.WriteString(`{"type":"message","role":"user","content":"hello"}` + "\n")
	f.WriteString("not json\n")
	f.WriteString(`{"type":"message","role":"assistant","content":"world"}` + "\n")
	f.Close()

	msgs, _, err := s.load(beanID)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages (skipping bad line), got %d", len(msgs))
	}
}

func TestToolMessagePersistsWithSummary(t *testing.T) {
	dir := t.TempDir()
	s, err := newStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	beanID := "bean-tool"

	// Simulate what readOutput now does: persist tool message with summary included
	s.appendMessage(beanID, Message{Role: RoleUser, Content: "build the project"})
	s.appendMessage(beanID, Message{Role: RoleAssistant, Content: "I'll build it."})
	// Tool message persisted AFTER summary was extracted (deferred persistence)
	s.appendMessage(beanID, Message{Role: RoleTool, Content: "Bash: Build beans binary"})
	s.appendMessage(beanID, Message{Role: RoleTool, Content: "Glob: **/main.go"})
	s.appendMessage(beanID, Message{Role: RoleAssistant, Content: "Done!"})

	// Reload and verify summaries survive
	msgs, _, err := s.load(beanID)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(msgs) != 5 {
		t.Fatalf("expected 5 messages, got %d", len(msgs))
	}
	if msgs[2].Content != "Bash: Build beans binary" {
		t.Errorf("tool msg[2] = %q, want %q", msgs[2].Content, "Bash: Build beans binary")
	}
	if msgs[3].Content != "Glob: **/main.go" {
		t.Errorf("tool msg[3] = %q, want %q", msgs[3].Content, "Glob: **/main.go")
	}
}

func TestManagerPersistence(t *testing.T) {
	dir := t.TempDir()

	// Create a manager with persistence
	m := NewManager(dir, nil)
	if m.store == nil {
		t.Fatal("expected store to be initialized")
	}

	// Manually add a session with messages
	m.sessions["bean-1"] = &Session{
		ID:        "bean-1",
		AgentType: "claude",
		Status:    StatusIdle,
		Messages: []Message{
			{Role: RoleUser, Content: "test"},
		},
	}

	// Persist a message through the store
	m.store.appendMessage("bean-1", Message{Role: RoleUser, Content: "test"})
	m.store.appendMessage("bean-1", Message{Role: RoleAssistant, Content: "response"})
	m.store.saveSessionID("bean-1", "sess-abc")

	// Create a new manager (simulating restart) — should load from disk
	m2 := NewManager(dir, nil)
	s := m2.GetSession("bean-1")
	if s == nil {
		t.Fatal("expected session to be loaded from disk")
	}
	if len(s.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(s.Messages))
	}
	if s.SessionID != "sess-abc" {
		t.Errorf("sessionID = %q, want %q", s.SessionID, "sess-abc")
	}
}
