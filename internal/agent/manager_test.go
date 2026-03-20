package agent

import (
	"testing"
)

func TestNewManager(t *testing.T) {
	m := NewManager("", nil)
	if m == nil {
		t.Fatal("NewManager returned nil")
	}
	if m.sessions == nil || m.processes == nil || m.subscribers == nil {
		t.Fatal("NewManager didn't initialize maps")
	}
}

func TestGetSession_NotFound(t *testing.T) {
	m := NewManager("", nil)
	s := m.GetSession("nonexistent")
	if s != nil {
		t.Errorf("expected nil, got %+v", s)
	}
}

func TestGetSession_ReturnsSnapshot(t *testing.T) {
	m := NewManager("", nil)
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
	m := NewManager("", nil)
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
	m := NewManager("", nil)
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
	m := NewManager("", nil)
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
	m := NewManager("", nil)
	m.sessions["test"] = &Session{
		ID:           "test",
		streamingIdx: -1,
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

func TestAppendAssistantText_InterleavedUserMessage(t *testing.T) {
	m := NewManager("", nil)
	m.sessions["test"] = &Session{
		ID:           "test",
		streamingIdx: -1,
		Messages: []Message{
			{Role: RoleUser, Content: "hello"},
		},
	}

	// Start streaming first response
	m.appendAssistantText("test", "First response")
	s := m.sessions["test"]
	if len(s.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(s.Messages))
	}

	// User sends another message mid-turn (interleaved)
	s.Messages = append(s.Messages, Message{Role: RoleUser, Content: "follow-up"})

	// More deltas from the FIRST response should still go to message[1]
	m.appendAssistantText("test", " continued")
	if s.Messages[1].Content != "First response continued" {
		t.Errorf("content = %q, want %q", s.Messages[1].Content, "First response continued")
	}
	if len(s.Messages) != 3 {
		t.Errorf("expected 3 messages (no spurious assistant), got %d", len(s.Messages))
	}

	// Reset streamingIdx (simulates eventResult)
	s.streamingIdx = -1

	// New deltas for the SECOND response should create a new assistant message
	m.appendAssistantText("test", "Second response")
	if len(s.Messages) != 4 {
		t.Fatalf("expected 4 messages, got %d", len(s.Messages))
	}
	if s.Messages[3].Role != RoleAssistant {
		t.Errorf("msg[3] role = %q, want assistant", s.Messages[3].Role)
	}
	if s.Messages[3].Content != "Second response" {
		t.Errorf("msg[3] content = %q, want %q", s.Messages[3].Content, "Second response")
	}
}

func TestSetError(t *testing.T) {
	m := NewManager("", nil)
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
	m := NewManager("", nil)
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

func TestClearSession_RemovesSession(t *testing.T) {
	m := NewManager("", nil)
	m.sessions["test"] = &Session{
		ID:     "test",
		Status: StatusIdle,
		Messages: []Message{
			{Role: RoleUser, Content: "hello"},
			{Role: RoleAssistant, Content: "hi there"},
		},
	}

	err := m.ClearSession("test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := m.sessions["test"]; ok {
		t.Error("expected session to be removed from memory")
	}

	// GetSession should return nil for cleared session (no store)
	s := m.GetSession("test")
	if s != nil {
		t.Errorf("expected nil after clear, got %+v", s)
	}
}

func TestClearSession_Notifies(t *testing.T) {
	m := NewManager("", nil)
	ch := m.Subscribe("test")
	defer m.Unsubscribe("test", ch)

	m.sessions["test"] = &Session{
		ID:     "test",
		Status: StatusIdle,
	}

	// Drain any existing notification
	select {
	case <-ch:
	default:
	}

	_ = m.ClearSession("test")

	select {
	case <-ch:
		// Good — received notification
	default:
		t.Error("expected notification after clear")
	}
}

func TestClearSession_Nonexistent(t *testing.T) {
	m := NewManager("", nil)
	// Should not error on clearing a session that doesn't exist
	err := m.ClearSession("nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSetPlanMode_CreatesSession(t *testing.T) {
	m := NewManager("", nil)

	err := m.SetPlanMode("test", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	s := m.sessions["test"]
	if s == nil {
		t.Fatal("expected session to be created")
	}
	if !s.PlanMode {
		t.Error("expected PlanMode to be true")
	}
	if s.Status != StatusIdle {
		t.Errorf("status = %q, want %q", s.Status, StatusIdle)
	}
}

func TestSetPlanMode_TogglesExisting(t *testing.T) {
	m := NewManager("", nil)
	m.sessions["test"] = &Session{
		ID:        "test",
		Status:    StatusIdle,
		PlanMode:  false,
		SessionID: "sess-123",
	}

	err := m.SetPlanMode("test", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	s := m.sessions["test"]
	if !s.PlanMode {
		t.Error("expected PlanMode to be true")
	}
	// SessionID should be preserved so --resume maintains conversation context
	if s.SessionID != "sess-123" {
		t.Errorf("expected SessionID to be preserved, got %q", s.SessionID)
	}
}

func TestSetPlanMode_NoopWhenSame(t *testing.T) {
	m := NewManager("", nil)
	ch := m.Subscribe("test")
	defer m.Unsubscribe("test", ch)

	m.sessions["test"] = &Session{
		ID:       "test",
		Status:   StatusIdle,
		PlanMode: true,
	}

	// Drain any existing notification
	select {
	case <-ch:
	default:
	}

	err := m.SetPlanMode("test", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should NOT notify since nothing changed
	select {
	case <-ch:
		t.Error("expected no notification for noop")
	default:
	}
}

func TestSetPlanMode_IncludedInSnapshot(t *testing.T) {
	m := NewManager("", nil)
	m.sessions["test"] = &Session{
		ID:       "test",
		Status:   StatusIdle,
		PlanMode: true,
	}

	snap := m.GetSession("test")
	if !snap.PlanMode {
		t.Error("expected PlanMode=true in snapshot")
	}
}

func TestHandleBlockingTool_ExitPlan(t *testing.T) {
	m := NewManager("", nil)
	ch := m.Subscribe("test")
	defer m.Unsubscribe("test", ch)

	m.sessions["test"] = &Session{
		ID:        "test",
		Status:    StatusRunning,
		PlanMode:  true,
		SessionID: "sess-123",
	}

	m.handleBlockingTool("test", &PendingInteraction{Type: InteractionExitPlan})

	s := m.sessions["test"]
	// Mode is NOT toggled in handleBlockingTool — that happens on approval
	if !s.PlanMode {
		t.Error("expected PlanMode to remain true (toggled on approval, not detection)")
	}
	if s.SessionID != "sess-123" {
		t.Errorf("expected SessionID to be preserved, got %q", s.SessionID)
	}
	if s.PendingInteraction == nil {
		t.Fatal("expected PendingInteraction to be set")
	}
	if s.PendingInteraction.Type != InteractionExitPlan {
		t.Errorf("expected InteractionExitPlan, got %q", s.PendingInteraction.Type)
	}
	if s.Status != StatusIdle {
		t.Errorf("expected status idle, got %q", s.Status)
	}

	// Should have notified
	select {
	case <-ch:
	default:
		t.Error("expected notification")
	}
}

func TestHandleBlockingTool_EnterPlan(t *testing.T) {
	m := NewManager("", nil)
	m.sessions["test"] = &Session{
		ID:        "test",
		Status:    StatusRunning,
		PlanMode:  false,
		SessionID: "sess-456",
	}

	m.handleBlockingTool("test", &PendingInteraction{Type: InteractionEnterPlan})

	s := m.sessions["test"]
	// Mode is NOT toggled in handleBlockingTool — that happens on approval
	if s.PlanMode {
		t.Error("expected PlanMode to remain false (toggled on approval, not detection)")
	}
	if s.SessionID != "sess-456" {
		t.Errorf("expected SessionID to be preserved, got %q", s.SessionID)
	}
	if s.PendingInteraction == nil || s.PendingInteraction.Type != InteractionEnterPlan {
		t.Error("expected InteractionEnterPlan pending interaction")
	}
}

func TestSendMessage_ClearsPendingInteraction(t *testing.T) {
	m := NewManager("", nil)
	m.sessions["test"] = &Session{
		ID:        "test",
		Status:    StatusIdle,
		WorkDir:   "/tmp/test",
		SessionID: "sess-123",
		PendingInteraction: &PendingInteraction{
			Type: InteractionExitPlan,
		},
	}

	// SendMessage will try to spawn a process — that will fail because
	// there's no claude binary in test. But we can check the session state
	// was updated before the spawn.
	_ = m.SendMessage("test", "/tmp/test", "proceed", nil)

	s := m.sessions["test"]
	if s.PendingInteraction != nil {
		t.Error("expected PendingInteraction to be cleared after SendMessage")
	}
}

func TestHandleBlockingTool_AskUser(t *testing.T) {
	m := NewManager("", nil)
	m.sessions["test"] = &Session{
		ID:        "test",
		Status:    StatusRunning,
		PlanMode:  false,
		SessionID: "sess-789",
	}

	m.handleBlockingTool("test", &PendingInteraction{Type: InteractionAskUser})

	s := m.sessions["test"]
	// Plan mode should not change for AskUser
	if s.PlanMode {
		t.Error("expected PlanMode to remain false for AskUser")
	}
	if s.SessionID != "sess-789" {
		t.Errorf("expected SessionID to be preserved, got %q", s.SessionID)
	}
	if s.PendingInteraction == nil || s.PendingInteraction.Type != InteractionAskUser {
		t.Error("expected InteractionAskUser pending interaction")
	}
	if s.Status != StatusIdle {
		t.Errorf("expected status idle, got %q", s.Status)
	}
}

func TestAutoApproveModeSwitch_EnterPlan(t *testing.T) {
	m := NewManager("", nil)
	ch := m.Subscribe("test")
	defer m.Unsubscribe("test", ch)

	m.sessions["test"] = &Session{
		ID:        "test",
		Status:    StatusRunning,
		PlanMode:  false,
		ActMode:   true, // default for new sessions
		SessionID: "sess-456",
		WorkDir:   "/tmp/test",
	}

	m.autoApproveModeSwitch("test", &PendingInteraction{Type: InteractionEnterPlan})

	s := m.sessions["test"]
	if !s.PlanMode {
		t.Error("expected PlanMode to be true after EnterPlanMode")
	}
	if s.ActMode {
		t.Error("expected ActMode to be false after EnterPlanMode (so process respawns with --permission-mode plan)")
	}
	if s.PendingInteraction != nil {
		t.Error("expected no PendingInteraction (auto-approved)")
	}
	if s.SessionID != "sess-456" {
		t.Errorf("expected SessionID to be preserved, got %q", s.SessionID)
	}

	// Should have notified
	select {
	case <-ch:
	default:
		t.Error("expected notification")
	}
}

func TestBlockingInteraction(t *testing.T) {
	inPlanMode := &Session{PlanMode: true}
	notInPlanMode := &Session{PlanMode: false}
	inActMode := &Session{PlanMode: true, ActMode: true}

	tests := []struct {
		name     string
		toolName string
		session  *Session
		expected *PendingInteraction
	}{
		{"ExitPlanMode when in plan mode", "ExitPlanMode", inPlanMode, &PendingInteraction{Type: InteractionExitPlan}},
		{"ExitPlanMode when not in plan mode (no-op)", "ExitPlanMode", notInPlanMode, nil},
		{"ExitPlanMode when in act mode (no-op after approval)", "ExitPlanMode", inActMode, nil},
		{"EnterPlanMode when not in plan mode", "EnterPlanMode", notInPlanMode, &PendingInteraction{Type: InteractionEnterPlan}},
		{"EnterPlanMode when already in plan mode (no-op)", "EnterPlanMode", inPlanMode, nil},
		{"AskUserQuestion", "AskUserQuestion", notInPlanMode, &PendingInteraction{Type: InteractionAskUser}},
		{"regular tool Read", "Read", notInPlanMode, nil},
		{"regular tool Write", "Write", notInPlanMode, nil},
		{"empty tool name", "", notInPlanMode, nil},
		{"nil session", "ExitPlanMode", nil, &PendingInteraction{Type: InteractionExitPlan}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := blockingInteraction(tt.toolName, tt.session)
			if tt.expected == nil && got != nil {
				t.Errorf("blockingInteraction(%q) = %v, want nil", tt.toolName, got)
			} else if tt.expected != nil && (got == nil || got.Type != tt.expected.Type) {
				t.Errorf("blockingInteraction(%q) = %v, want %v", tt.toolName, got, tt.expected)
			}
		})
	}
}

func TestFindPlanFilePath(t *testing.T) {
	tests := []struct {
		name        string
		invocations []ToolInvocation
		expected    string
	}{
		{
			name: "finds plan file from Write invocation",
			invocations: []ToolInvocation{
				{Tool: "Read", Input: "/some/file.go"},
				{Tool: "Write", Input: "/Users/test/.claude/plans/cool-plan.md"},
				{Tool: "ExitPlanMode", Input: ""},
			},
			expected: "/Users/test/.claude/plans/cool-plan.md",
		},
		{
			name: "ignores Write to non-plan paths",
			invocations: []ToolInvocation{
				{Tool: "Write", Input: "/tmp/some-file.md"},
			},
			expected: "",
		},
		{
			name: "returns last plan file if multiple",
			invocations: []ToolInvocation{
				{Tool: "Write", Input: "/Users/test/.claude/plans/old-plan.md"},
				{Tool: "Write", Input: "/Users/test/.claude/plans/new-plan.md"},
			},
			expected: "/Users/test/.claude/plans/new-plan.md",
		},
		{
			name:        "returns empty for no invocations",
			invocations: nil,
			expected:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findPlanFilePath(tt.invocations)
			if got != tt.expected {
				t.Errorf("findPlanFilePath() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestBuildClaudeArgs_PlanMode(t *testing.T) {
	args := buildClaudeArgs(&Session{PlanMode: true})
	found := false
	for i, a := range args {
		if a == "--permission-mode" && i+1 < len(args) && args[i+1] == "plan" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected --permission-mode plan in args, got %v", args)
	}
}

func TestBuildClaudeArgs_NoPlanMode(t *testing.T) {
	// When neither PlanMode nor ActMode, should use Claude's default permissions (no flag)
	args := buildClaudeArgs(&Session{PlanMode: false})
	for _, a := range args {
		if a == "--permission-mode" || a == "--dangerously-skip-permissions" {
			t.Errorf("expected no permission flags in default mode args, got %v", args)
		}
	}
}

func TestLoadOrCreateSession_DefaultsToActMode(t *testing.T) {
	m := NewManager("", nil)
	m.mu.Lock()
	s := m.loadOrCreateSession("test", "/tmp/test")
	m.mu.Unlock()

	if !s.ActMode {
		t.Error("expected new sessions to default to ActMode=true")
	}
}

func TestNewManager_PlanMode(t *testing.T) {
	m := NewManager("", nil, DefaultModePlan)
	m.mu.Lock()
	s := m.loadOrCreateSession("test", "/tmp/test")
	m.mu.Unlock()

	if s.ActMode {
		t.Error("expected ActMode=false in plan mode")
	}
	if !s.PlanMode {
		t.Error("expected PlanMode=true in plan mode")
	}
}

func TestNewManager_ExplicitActMode(t *testing.T) {
	m := NewManager("", nil, DefaultModeAct)
	m.mu.Lock()
	s := m.loadOrCreateSession("test", "/tmp/test")
	m.mu.Unlock()

	if !s.ActMode {
		t.Error("expected ActMode=true")
	}
	if s.PlanMode {
		t.Error("expected PlanMode=false")
	}
}

func TestSetActMode_CreatesSession(t *testing.T) {
	m := NewManager("", nil)

	err := m.SetActMode("test", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	s := m.sessions["test"]
	if s == nil {
		t.Fatal("expected session to be created")
	}
	if !s.ActMode {
		t.Error("expected ActMode to be true")
	}
	if s.Status != StatusIdle {
		t.Errorf("status = %q, want %q", s.Status, StatusIdle)
	}
}

func TestSetActMode_TogglesExisting(t *testing.T) {
	m := NewManager("", nil)
	m.sessions["test"] = &Session{
		ID:        "test",
		Status:    StatusIdle,
		ActMode:  true,
		SessionID: "sess-123",
	}

	err := m.SetActMode("test", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	s := m.sessions["test"]
	if s.ActMode {
		t.Error("expected ActMode to be false")
	}
	if s.SessionID != "sess-123" {
		t.Errorf("expected SessionID to be preserved, got %q", s.SessionID)
	}
}

func TestSetActMode_NoopWhenSame(t *testing.T) {
	m := NewManager("", nil)
	ch := m.Subscribe("test")
	defer m.Unsubscribe("test", ch)

	m.sessions["test"] = &Session{
		ID:       "test",
		Status:   StatusIdle,
		ActMode: true,
	}

	// Drain any existing notification
	select {
	case <-ch:
	default:
	}

	err := m.SetActMode("test", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should NOT notify since nothing changed
	select {
	case <-ch:
		t.Error("expected no notification for noop")
	default:
	}
}

func TestSetActMode_IncludedInSnapshot(t *testing.T) {
	m := NewManager("", nil)
	m.sessions["test"] = &Session{
		ID:       "test",
		Status:   StatusIdle,
		ActMode: true,
	}

	snap := m.GetSession("test")
	if !snap.ActMode {
		t.Error("expected ActMode=true in snapshot")
	}
}

func TestBuildClaudeArgs_ActMode(t *testing.T) {
	args := buildClaudeArgs(&Session{ActMode: true})
	found := false
	for _, a := range args {
		if a == "--dangerously-skip-permissions" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected --dangerously-skip-permissions in args, got %v", args)
	}

	// Act mode should also include --allowedTools for sensitive Claude config files
	foundAllowed := false
	for i, a := range args {
		if a == "--allowedTools" {
			foundAllowed = true
			// Verify the expected tool patterns follow
			remaining := args[i+1:]
			expected := []string{
				"Edit(CLAUDE.md)", "Write(CLAUDE.md)",
				"Edit(.claude/**)", "Write(.claude/**)",
			}
			for _, e := range expected {
				found := false
				for _, r := range remaining {
					if r == e {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected %q in --allowedTools args, got %v", e, remaining)
				}
			}
			break
		}
	}
	if !foundAllowed {
		t.Errorf("expected --allowedTools in act mode args, got %v", args)
	}
}

func TestBuildClaudeArgs_ActOverridesPlan(t *testing.T) {
	// When both are set, Act takes precedence (no plan mode flag)
	args := buildClaudeArgs(&Session{ActMode: true, PlanMode: true})
	foundAct := false
	foundPlan := false
	for _, a := range args {
		if a == "--dangerously-skip-permissions" {
			foundAct = true
		}
		if a == "--permission-mode" {
			foundPlan = true
		}
	}
	if !foundAct {
		t.Error("expected --dangerously-skip-permissions in args")
	}
	if foundPlan {
		t.Error("unexpected --permission-mode when ActMode is set")
	}
}

func TestBuildClaudeArgs_Effort(t *testing.T) {
	args := buildClaudeArgs(&Session{Effort: "max"})
	found := false
	for i, a := range args {
		if a == "--effort" && i+1 < len(args) && args[i+1] == "max" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected --effort max in args, got %v", args)
	}
}

func TestBuildClaudeArgs_NoEffort(t *testing.T) {
	args := buildClaudeArgs(&Session{})
	for _, a := range args {
		if a == "--effort" {
			t.Errorf("expected no --effort flag when effort is empty, got %v", args)
		}
	}
}

func TestSetEffort_CreatesSession(t *testing.T) {
	m := NewManager("", nil)
	err := m.SetEffort("test-bean", "max")
	if err != nil {
		t.Fatalf("SetEffort: %v", err)
	}

	snap := m.GetSession("test-bean")
	if snap == nil {
		t.Fatal("expected session to exist")
	}
	if snap.Effort != "max" {
		t.Errorf("expected effort 'max', got %q", snap.Effort)
	}
}

func TestSetEffort_UpdatesExisting(t *testing.T) {
	m := NewManager("", nil)
	m.mu.Lock()
	m.sessions["test-bean"] = &Session{
		ID:           "test-bean",
		Effort:       "low",
		streamingIdx: -1,
	}
	m.mu.Unlock()

	err := m.SetEffort("test-bean", "max")
	if err != nil {
		t.Fatalf("SetEffort: %v", err)
	}

	snap := m.GetSession("test-bean")
	if snap.Effort != "max" {
		t.Errorf("expected effort 'max', got %q", snap.Effort)
	}
}

func TestSetEffort_NoopWhenSame(t *testing.T) {
	m := NewManager("", nil)
	m.mu.Lock()
	m.sessions["test-bean"] = &Session{
		ID:           "test-bean",
		Effort:       "high",
		streamingIdx: -1,
	}
	m.mu.Unlock()

	ch := m.Subscribe("test-bean")

	// Set the same effort — should be a no-op (no notification)
	err := m.SetEffort("test-bean", "high")
	if err != nil {
		t.Fatalf("SetEffort: %v", err)
	}

	select {
	case <-ch:
		t.Error("expected no notification when effort unchanged")
	default:
		// expected
	}
}


func TestGlobalSubscribeUnsubscribe(t *testing.T) {
	m := NewManager("", nil)
	ch := m.SubscribeGlobal()

	m.globalSubMu.Lock()
	if len(m.globalSubscribers) != 1 {
		t.Errorf("expected 1 global subscriber, got %d", len(m.globalSubscribers))
	}
	m.globalSubMu.Unlock()

	m.UnsubscribeGlobal(ch)

	_, ok := <-ch
	if ok {
		t.Error("expected channel to be closed")
	}

	m.globalSubMu.Lock()
	if len(m.globalSubscribers) != 0 {
		t.Errorf("expected 0 global subscribers, got %d", len(m.globalSubscribers))
	}
	m.globalSubMu.Unlock()
}

func TestNotify_GlobalSubscribers(t *testing.T) {
	m := NewManager("", nil)
	globalCh := m.SubscribeGlobal()
	defer m.UnsubscribeGlobal(globalCh)

	// Notifying any bean should also notify global subscribers
	m.notify("some-bean")

	select {
	case <-globalCh:
		// Good — received global notification
	default:
		t.Error("expected global notification")
	}
}

func TestListRunningSessions_Empty(t *testing.T) {
	m := NewManager("", nil)
	result := m.ListRunningSessions()
	if len(result) != 0 {
		t.Errorf("expected empty list, got %d items", len(result))
	}
}

func TestListRunningSessions_FiltersRunning(t *testing.T) {
	m := NewManager("", nil)
	m.sessions["running-1"] = &Session{ID: "running-1", Status: StatusRunning}
	m.sessions["idle-1"] = &Session{ID: "idle-1", Status: StatusIdle}
	m.sessions["running-2"] = &Session{ID: "running-2", Status: StatusRunning}
	m.sessions["error-1"] = &Session{ID: "error-1", Status: StatusError}

	result := m.ListRunningSessions()
	if len(result) != 2 {
		t.Fatalf("expected 2 running sessions, got %d", len(result))
	}

	ids := make(map[string]bool)
	for _, a := range result {
		ids[a.BeanID] = true
		if a.Status != StatusRunning {
			t.Errorf("expected StatusRunning, got %q for %s", a.Status, a.BeanID)
		}
	}
	if !ids["running-1"] || !ids["running-2"] {
		t.Errorf("expected running-1 and running-2, got %v", ids)
	}
}

func TestShutdown(t *testing.T) {
	m := NewManager("", nil)
	// Just verify it doesn't panic with no processes
	m.Shutdown()
}

func TestSetDefaultEffort_AppliedToNewSession(t *testing.T) {
	m := NewManager("", nil)
	m.SetDefaultEffort(EffortLevelMedium)

	// loadOrCreateSession is called by SendMessage internally; invoke it directly
	// via the exported path by checking the session after AddInfoMessage creates one.
	m.AddInfoMessage("test-bean", "hello")

	snap := m.GetSession("test-bean")
	if snap == nil {
		t.Fatal("expected session to exist")
	}
	if snap.Effort != "medium" {
		t.Errorf("expected default effort 'medium', got %q", snap.Effort)
	}
}

func TestSetDefaultEffort_EmptyMeansNoEffort(t *testing.T) {
	m := NewManager("", nil)
	// default is empty — no effort applied

	m.AddInfoMessage("test-bean", "hello")

	snap := m.GetSession("test-bean")
	if snap == nil {
		t.Fatal("expected session to exist")
	}
	if snap.Effort != "" {
		t.Errorf("expected empty effort, got %q", snap.Effort)
	}
}
