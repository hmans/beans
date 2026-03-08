package agent

import (
	"testing"
)

func TestNewManager(t *testing.T) {
	m := NewManager()
	if m == nil {
		t.Fatal("NewManager returned nil")
	}
	if m.sessions == nil || m.processes == nil || m.subscribers == nil {
		t.Fatal("NewManager didn't initialize maps")
	}
}

func TestGetSession_NotFound(t *testing.T) {
	m := NewManager()
	s := m.GetSession("nonexistent")
	if s != nil {
		t.Errorf("expected nil, got %+v", s)
	}
}

func TestGetSession_ReturnsSnapshot(t *testing.T) {
	m := NewManager()
	m.sessions["test"] = &Session{
		ID:        "test",
		AgentType: "claude",
		Status:    StatusIdle,
		Messages: []Message{
			{Role: RoleUser, Content: "hello"},
		},
	}

	snap := m.GetSession("test")
	if snap == nil {
		t.Fatal("expected session, got nil")
	}
	if snap.ID != "test" {
		t.Errorf("ID = %q, want %q", snap.ID, "test")
	}
	if len(snap.Messages) != 1 {
		t.Errorf("Messages len = %d, want 1", len(snap.Messages))
	}

	// Mutating the snapshot shouldn't affect the original
	snap.Messages = append(snap.Messages, Message{Role: RoleAssistant, Content: "hi"})
	orig := m.GetSession("test")
	if len(orig.Messages) != 1 {
		t.Error("snapshot mutation leaked to original session")
	}
}

func TestSubscribeUnsubscribe(t *testing.T) {
	m := NewManager()
	ch := m.Subscribe("bean-1")

	// Should have one subscriber
	m.subMu.Lock()
	if len(m.subscribers["bean-1"]) != 1 {
		t.Errorf("expected 1 subscriber, got %d", len(m.subscribers["bean-1"]))
	}
	m.subMu.Unlock()

	m.Unsubscribe("bean-1", ch)

	// Channel should be closed
	_, ok := <-ch
	if ok {
		t.Error("expected channel to be closed")
	}

	m.subMu.Lock()
	if len(m.subscribers["bean-1"]) != 0 {
		t.Errorf("expected 0 subscribers after unsubscribe, got %d", len(m.subscribers["bean-1"]))
	}
	m.subMu.Unlock()
}

func TestNotify(t *testing.T) {
	m := NewManager()
	ch := m.Subscribe("bean-1")
	defer m.Unsubscribe("bean-1", ch)

	m.notify("bean-1")

	select {
	case <-ch:
		// Good — received notification
	default:
		t.Error("expected notification on channel")
	}
}

func TestNotify_NonBlocking(t *testing.T) {
	m := NewManager()
	ch := m.Subscribe("bean-1")
	defer m.Unsubscribe("bean-1", ch)

	// Fill the channel buffer
	m.notify("bean-1")
	// Second notify should not block
	m.notify("bean-1")

	// Drain
	<-ch

	// Channel should be empty now
	select {
	case <-ch:
		t.Error("expected channel to be empty after single drain")
	default:
	}
}

func TestAppendAssistantText(t *testing.T) {
	m := NewManager()
	m.sessions["test"] = &Session{
		ID: "test",
		Messages: []Message{
			{Role: RoleUser, Content: "hello"},
		},
	}

	// First append creates a new assistant message
	m.appendAssistantText("test", "Hi")
	s := m.sessions["test"]
	if len(s.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(s.Messages))
	}
	if s.Messages[1].Role != RoleAssistant {
		t.Errorf("role = %q, want %q", s.Messages[1].Role, RoleAssistant)
	}
	if s.Messages[1].Content != "Hi" {
		t.Errorf("content = %q, want %q", s.Messages[1].Content, "Hi")
	}

	// Second append extends the same message
	m.appendAssistantText("test", " there!")
	if s.Messages[1].Content != "Hi there!" {
		t.Errorf("content = %q, want %q", s.Messages[1].Content, "Hi there!")
	}
}

func TestSetAssistantText(t *testing.T) {
	m := NewManager()
	m.sessions["test"] = &Session{
		ID: "test",
		Messages: []Message{
			{Role: RoleUser, Content: "hello"},
		},
	}

	// Creates assistant message and sets content
	m.setAssistantText("test", "Final answer")
	s := m.sessions["test"]
	if len(s.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(s.Messages))
	}
	if s.Messages[1].Content != "Final answer" {
		t.Errorf("content = %q, want %q", s.Messages[1].Content, "Final answer")
	}

	// Replaces existing content
	m.setAssistantText("test", "Updated answer")
	if s.Messages[1].Content != "Updated answer" {
		t.Errorf("content = %q, want %q", s.Messages[1].Content, "Updated answer")
	}
}

func TestSetError(t *testing.T) {
	m := NewManager()
	ch := m.Subscribe("test")
	defer m.Unsubscribe("test", ch)

	m.sessions["test"] = &Session{
		ID:     "test",
		Status: StatusRunning,
	}

	m.setError("test", "something broke")

	s := m.sessions["test"]
	if s.Status != StatusError {
		t.Errorf("status = %q, want %q", s.Status, StatusError)
	}
	if s.Error != "something broke" {
		t.Errorf("error = %q, want %q", s.Error, "something broke")
	}

	// Should have notified
	select {
	case <-ch:
	default:
		t.Error("expected notification")
	}
}

func TestStopSession(t *testing.T) {
	m := NewManager()
	m.sessions["test"] = &Session{
		ID:     "test",
		Status: StatusRunning,
	}

	err := m.StopSession("test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	s := m.sessions["test"]
	if s.Status != StatusIdle {
		t.Errorf("status = %q, want %q", s.Status, StatusIdle)
	}
}

func TestShutdown(t *testing.T) {
	m := NewManager()
	// Just verify it doesn't panic with no processes
	m.Shutdown()
}
