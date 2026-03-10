package agent

import (
	"testing"
)

func TestNewManager(t *testing.T) {
	m := NewManager("")
	if m == nil {
		t.Fatal("NewManager returned nil")
	}
	if m.sessions == nil || m.processes == nil || m.subscribers == nil {
		t.Fatal("NewManager didn't initialize maps")
	}
}

func TestGetSession_NotFound(t *testing.T) {
	m := NewManager("")
	s := m.GetSession("nonexistent")
	if s != nil {
		t.Errorf("expected nil, got %+v", s)
	}
}

func TestGetSession_ReturnsSnapshot(t *testing.T) {
	m := NewManager("")
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
	m := NewManager("")
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
	m := NewManager("")
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
	m := NewManager("")
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
	m := NewManager("")
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
	m := NewManager("")
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
	m := NewManager("")
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
	m := NewManager("")
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
	m := NewManager("")
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
	m := NewManager("")
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
	m := NewManager("")
	// Should not error on clearing a session that doesn't exist
	err := m.ClearSession("nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSetPlanMode_CreatesSession(t *testing.T) {
	m := NewManager("")

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
	m := NewManager("")
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
	m := NewManager("")
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
	m := NewManager("")
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
	m := NewManager("")
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
	if s.PlanMode {
		t.Error("expected PlanMode to be false after ExitPlanMode")
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
	m := NewManager("")
	m.sessions["test"] = &Session{
		ID:        "test",
		Status:    StatusRunning,
		PlanMode:  false,
		SessionID: "sess-456",
	}

	m.handleBlockingTool("test", &PendingInteraction{Type: InteractionEnterPlan})

	s := m.sessions["test"]
	if !s.PlanMode {
		t.Error("expected PlanMode to be true after EnterPlanMode")
	}
	if s.SessionID != "sess-456" {
		t.Errorf("expected SessionID to be preserved, got %q", s.SessionID)
	}
	if s.PendingInteraction == nil || s.PendingInteraction.Type != InteractionEnterPlan {
		t.Error("expected InteractionEnterPlan pending interaction")
	}
}

func TestSendMessage_ClearsPendingInteraction(t *testing.T) {
	m := NewManager("")
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
	_ = m.SendMessage("test", "/tmp/test", "proceed")

	s := m.sessions["test"]
	if s.PendingInteraction != nil {
		t.Error("expected PendingInteraction to be cleared after SendMessage")
	}
}

func TestHandleBlockingTool_AskUser(t *testing.T) {
	m := NewManager("")
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

func TestBlockingInteraction(t *testing.T) {
	inPlanMode := &Session{PlanMode: true}
	notInPlanMode := &Session{PlanMode: false}

	tests := []struct {
		name     string
		toolName string
		session  *Session
		expected *PendingInteraction
	}{
		{"ExitPlanMode when in plan mode", "ExitPlanMode", inPlanMode, &PendingInteraction{Type: InteractionExitPlan}},
		{"ExitPlanMode when not in plan mode (no-op)", "ExitPlanMode", notInPlanMode, nil},
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
	// When not in plan mode, should use acceptEdits (not plan)
	args := buildClaudeArgs(&Session{PlanMode: false})
	for i, a := range args {
		if a == "--permission-mode" && i+1 < len(args) && args[i+1] == "plan" {
			t.Errorf("unexpected --permission-mode plan in non-plan args: %v", args)
		}
	}
}

func TestSetYoloMode_CreatesSession(t *testing.T) {
	m := NewManager("")

	err := m.SetYoloMode("test", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	s := m.sessions["test"]
	if s == nil {
		t.Fatal("expected session to be created")
	}
	if !s.YoloMode {
		t.Error("expected YoloMode to be true")
	}
	if s.Status != StatusIdle {
		t.Errorf("status = %q, want %q", s.Status, StatusIdle)
	}
}

func TestSetYoloMode_TogglesExisting(t *testing.T) {
	m := NewManager("")
	m.sessions["test"] = &Session{
		ID:        "test",
		Status:    StatusIdle,
		YoloMode:  false,
		SessionID: "sess-123",
	}

	err := m.SetYoloMode("test", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	s := m.sessions["test"]
	if !s.YoloMode {
		t.Error("expected YoloMode to be true")
	}
	if s.SessionID != "sess-123" {
		t.Errorf("expected SessionID to be preserved, got %q", s.SessionID)
	}
}

func TestSetYoloMode_NoopWhenSame(t *testing.T) {
	m := NewManager("")
	ch := m.Subscribe("test")
	defer m.Unsubscribe("test", ch)

	m.sessions["test"] = &Session{
		ID:       "test",
		Status:   StatusIdle,
		YoloMode: true,
	}

	// Drain any existing notification
	select {
	case <-ch:
	default:
	}

	err := m.SetYoloMode("test", true)
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

func TestSetYoloMode_IncludedInSnapshot(t *testing.T) {
	m := NewManager("")
	m.sessions["test"] = &Session{
		ID:       "test",
		Status:   StatusIdle,
		YoloMode: true,
	}

	snap := m.GetSession("test")
	if !snap.YoloMode {
		t.Error("expected YoloMode=true in snapshot")
	}
}

func TestBuildClaudeArgs_YoloMode(t *testing.T) {
	args := buildClaudeArgs(&Session{YoloMode: true})
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
}

func TestBuildClaudeArgs_YoloOverridesPlan(t *testing.T) {
	// When both are set, YOLO takes precedence (no plan mode flag)
	args := buildClaudeArgs(&Session{YoloMode: true, PlanMode: true})
	foundYolo := false
	foundPlan := false
	for _, a := range args {
		if a == "--dangerously-skip-permissions" {
			foundYolo = true
		}
		if a == "--permission-mode" {
			foundPlan = true
		}
	}
	if !foundYolo {
		t.Error("expected --dangerously-skip-permissions in args")
	}
	if foundPlan {
		t.Error("unexpected --permission-mode when YoloMode is set")
	}
}

func TestResolvePermission_Allow(t *testing.T) {
	m := NewManager("")
	ch := m.Subscribe("test")
	defer m.Unsubscribe("test", ch)

	m.sessions["test"] = &Session{
		ID:      "test",
		Status:  StatusRunning,
		WorkDir: "/tmp/test",
		PendingInteraction: &PendingInteraction{
			Type: InteractionPermission,
			PermissionDenials: []PermissionDenial{
				{ToolName: "Write", ToolInput: map[string]any{"file_path": "/tmp/test.txt"}},
			},
		},
	}

	// Drain any notification
	select {
	case <-ch:
	default:
	}

	err := m.ResolvePermission("test", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// PendingInteraction should be cleared
	s := m.sessions["test"]
	if s.PendingInteraction != nil {
		t.Error("expected PendingInteraction to be cleared")
	}

	// AllowedTools should contain the denied tool
	found := false
	for _, tool := range s.AllowedTools {
		if tool == "Write" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected Write in AllowedTools, got %v", s.AllowedTools)
	}

	// Should have notified
	select {
	case <-ch:
	default:
		t.Error("expected notification")
	}
}

func TestResolvePermission_Deny(t *testing.T) {
	m := NewManager("")

	m.sessions["test"] = &Session{
		ID:     "test",
		Status: StatusRunning,
		PendingInteraction: &PendingInteraction{
			Type: InteractionPermission,
			PermissionDenials: []PermissionDenial{
				{ToolName: "Bash"},
			},
		},
	}

	err := m.ResolvePermission("test", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	s := m.sessions["test"]
	if s.PendingInteraction != nil {
		t.Error("expected PendingInteraction to be cleared")
	}
	// AllowedTools should NOT contain the denied tool
	if len(s.AllowedTools) > 0 {
		t.Errorf("expected empty AllowedTools on deny, got %v", s.AllowedTools)
	}
}

func TestResolvePermission_NoSession(t *testing.T) {
	m := NewManager("")
	err := m.ResolvePermission("nonexistent", true)
	if err == nil {
		t.Error("expected error for nonexistent session")
	}
}

func TestResolvePermission_NoPending(t *testing.T) {
	m := NewManager("")
	m.sessions["test"] = &Session{
		ID:     "test",
		Status: StatusIdle,
	}
	err := m.ResolvePermission("test", true)
	if err == nil {
		t.Error("expected error when no pending permission")
	}
}

func TestBuildClaudeArgs_ActMode(t *testing.T) {
	// Act mode: not plan, not yolo — should use acceptEdits and allow beans CLI
	args := buildClaudeArgs(&Session{ID: "bean-123"})
	for _, a := range args {
		if a == "--dangerously-skip-permissions" {
			t.Errorf("unexpected --dangerously-skip-permissions in act mode")
		}
	}
	// Should have --permission-mode acceptEdits
	foundAcceptEdits := false
	for i, a := range args {
		if a == "--permission-mode" && i+1 < len(args) && args[i+1] == "acceptEdits" {
			foundAcceptEdits = true
		}
	}
	if !foundAcceptEdits {
		t.Errorf("expected --permission-mode acceptEdits in act mode, got %v", args)
	}
	// Should have --allowedTools Bash(beans:*)
	foundBeans := false
	for i, a := range args {
		if a == "--allowedTools" && i+1 < len(args) && args[i+1] == "Bash(beans:*)" {
			foundBeans = true
		}
	}
	if !foundBeans {
		t.Errorf("expected --allowedTools Bash(beans:*) in act mode, got %v", args)
	}
}

func TestBuildClaudeArgs_AllowedTools(t *testing.T) {
	args := buildClaudeArgs(&Session{
		ID:           "bean-123",
		AllowedTools: []string{"Read", "Bash"},
	})
	// Should have --allowedTools for each user tool + the default Bash(beans:*)
	count := 0
	for i, a := range args {
		if a == "--allowedTools" && i+1 < len(args) {
			count++
		}
	}
	if count != 3 {
		t.Errorf("expected 3 --allowedTools flags (1 default + 2 user), got %d in %v", count, args)
	}
}

func TestShutdown(t *testing.T) {
	m := NewManager("")
	// Just verify it doesn't panic with no processes
	m.Shutdown()
}
