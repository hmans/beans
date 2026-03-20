package agent

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestSendToProcessImageOnlyNoEmptyTextBlock verifies that sending an image
// with no text does not produce an empty text content block, which would cause
// the Anthropic API to reject the message with "text content blocks must be non-empty".
func TestSendToProcessImageOnlyNoEmptyTextBlock(t *testing.T) {
	// Set up a temp store with a fake image file
	tmpDir := t.TempDir()
	beanID := "beans-test1"
	attachDir := filepath.Join(tmpDir, "attachments", beanID)
	if err := os.MkdirAll(attachDir, 0o755); err != nil {
		t.Fatal(err)
	}
	imgID := "test-image.png"
	if err := os.WriteFile(filepath.Join(attachDir, imgID), []byte("fake-png-data"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := &Manager{
		sessions:    make(map[string]*Session),
		processes:   make(map[string]*runningProcess),
		subscribers: make(map[string][]chan struct{}),
		store:       &store{dir: tmpDir},
	}

	// Use a pipe to capture what sendToProcess writes
	pr, pw := io.Pipe()
	proc := &runningProcess{stdin: pw, done: make(chan struct{})}

	images := []ImageRef{{ID: imgID, MediaType: "image/png"}}

	// Read output in a goroutine
	var output []byte
	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		output, _ = io.ReadAll(pr)
	}()

	err := m.sendToProcess(proc, beanID, "", images)
	if err != nil {
		t.Fatalf("sendToProcess failed: %v", err)
	}
	pw.Close()
	<-readDone

	// Parse the JSON output
	var msg map[string]interface{}
	if err := json.Unmarshal(output, &msg); err != nil {
		t.Fatalf("failed to parse output JSON: %v\nraw: %s", err, output)
	}

	message := msg["message"].(map[string]interface{})
	content := message["content"].([]interface{})

	// Verify: no text block should be present
	for i, block := range content {
		b := block.(map[string]interface{})
		if b["type"] == "text" {
			text := b["text"].(string)
			if text == "" {
				t.Errorf("content[%d] is an empty text block — this will be rejected by the API", i)
			}
		}
	}

	// Verify: should have exactly one image block
	if len(content) != 1 {
		t.Errorf("expected 1 content block (image only), got %d", len(content))
	}
	first := content[0].(map[string]interface{})
	if first["type"] != "image" {
		t.Errorf("expected image block, got %q", first["type"])
	}
}

// TestSendToProcessImageWithText verifies that when both text and image are
// provided, both blocks are included in the correct order.
func TestSendToProcessImageWithText(t *testing.T) {
	tmpDir := t.TempDir()
	beanID := "beans-test2"
	attachDir := filepath.Join(tmpDir, "attachments", beanID)
	if err := os.MkdirAll(attachDir, 0o755); err != nil {
		t.Fatal(err)
	}
	imgID := "test-image.png"
	if err := os.WriteFile(filepath.Join(attachDir, imgID), []byte("fake-png-data"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := &Manager{
		sessions:    make(map[string]*Session),
		processes:   make(map[string]*runningProcess),
		subscribers: make(map[string][]chan struct{}),
		store:       &store{dir: tmpDir},
	}

	pr, pw := io.Pipe()
	proc := &runningProcess{stdin: pw, done: make(chan struct{})}

	images := []ImageRef{{ID: imgID, MediaType: "image/png"}}

	var output []byte
	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		output, _ = io.ReadAll(pr)
	}()

	err := m.sendToProcess(proc, beanID, "Describe this image", images)
	if err != nil {
		t.Fatalf("sendToProcess failed: %v", err)
	}
	pw.Close()
	<-readDone

	var msg map[string]interface{}
	if err := json.Unmarshal(output, &msg); err != nil {
		t.Fatalf("failed to parse output JSON: %v", err)
	}

	message := msg["message"].(map[string]interface{})
	content := message["content"].([]interface{})

	// Should have text + image = 2 blocks
	if len(content) != 2 {
		t.Fatalf("expected 2 content blocks, got %d", len(content))
	}

	// First block should be text
	textBlock := content[0].(map[string]interface{})
	if textBlock["type"] != "text" || textBlock["text"] != "Describe this image" {
		t.Errorf("expected text block with message, got %v", textBlock)
	}

	// Second block should be image
	imgBlock := content[1].(map[string]interface{})
	if imgBlock["type"] != "image" {
		t.Errorf("expected image block, got %v", imgBlock)
	}
}

// TestReadOutputMessageOrder verifies that tool messages appear between
// the assistant text that precedes and follows them, not grouped at the end.
func TestReadOutputMessageOrder(t *testing.T) {
	// Simulate Claude Code stream-json output:
	// 1. Assistant starts typing "Before tool"
	// 2. Tool "Read" is invoked
	// 3. Assistant continues with "After tool"
	// 4. Result event closes the turn
	lines := strings.Join([]string{
		// First text block starts
		`{"type":"content_block_start","content_block":{"type":"text","text":""}}`,
		`{"type":"content_block_delta","delta":{"type":"text_delta","text":"Before tool"}}`,
		// Tool use
		`{"type":"content_block_start","content_block":{"type":"tool_use","name":"Read"}}`,
		`{"type":"content_block_delta","delta":{"type":"input_json_delta","partial_json":"{\"file_path\":\"/tmp/test\"}"}}`,
		// New text block after tool
		`{"type":"content_block_start","content_block":{"type":"text","text":""}}`,
		`{"type":"content_block_delta","delta":{"type":"text_delta","text":"After tool"}}`,
		// Turn complete
		`{"type":"result","session_id":"sess-1"}`,
	}, "\n")

	m := &Manager{
		sessions:    make(map[string]*Session),
		processes:   make(map[string]*runningProcess),
		subscribers: make(map[string][]chan struct{}),
	}

	session := &Session{
		ID:           "bean-test",
		AgentType:    "claude",
		Status:       StatusRunning,
		Messages:     []Message{{Role: RoleUser, Content: "hello"}},
		streamingIdx: -1,
	}
	m.sessions["bean-test"] = session
	proc := &runningProcess{done: make(chan struct{})}
	m.processes["bean-test"] = proc

	m.readOutput("bean-test", strings.NewReader(lines), "", proc)

	// Expected message order:
	// [0] USER: "hello"          (pre-existing)
	// [1] ASSISTANT: "Before tool"
	// [2] TOOL: "Read: /tmp/test"
	// [3] ASSISTANT: "After tool"
	msgs := session.Messages

	if len(msgs) != 4 {
		for i, m := range msgs {
			t.Logf("  [%d] %s: %q", i, m.Role, m.Content)
		}
		t.Fatalf("expected 4 messages, got %d", len(msgs))
	}

	tests := []struct {
		idx     int
		role    MessageRole
		contain string
	}{
		{0, RoleUser, "hello"},
		{1, RoleAssistant, "Before tool"},
		{2, RoleTool, "Read"},
		{3, RoleAssistant, "After tool"},
	}

	for _, tt := range tests {
		msg := msgs[tt.idx]
		if msg.Role != tt.role {
			t.Errorf("msgs[%d].Role = %q, want %q", tt.idx, msg.Role, tt.role)
		}
		if !strings.Contains(msg.Content, tt.contain) {
			t.Errorf("msgs[%d].Content = %q, want it to contain %q", tt.idx, msg.Content, tt.contain)
		}
	}
}

// TestReadOutputMultiTurnResetsStatus verifies that when Claude Code starts a
// new turn within the same process (e.g. after a Stop hook), the session status
// transitions back to Running from Idle.
func TestReadOutputMultiTurnResetsStatus(t *testing.T) {
	// Use a pipe so we can feed lines one at a time and observe status between events.
	pr, pw := io.Pipe()

	m := &Manager{
		sessions:    make(map[string]*Session),
		processes:   make(map[string]*runningProcess),
		subscribers: make(map[string][]chan struct{}),
	}

	session := &Session{
		ID:           "bean-multi-turn",
		AgentType:    "claude",
		Status:       StatusRunning,
		Messages:     []Message{{Role: RoleUser, Content: "hello"}},
		streamingIdx: -1,
	}
	m.sessions["bean-multi-turn"] = session
	proc := &runningProcess{done: make(chan struct{})}
	m.processes["bean-multi-turn"] = proc

	// Run readOutput in a goroutine since it blocks
	done := make(chan struct{})
	go func() {
		defer close(done)
		m.readOutput("bean-multi-turn", pr, "", proc)
	}()

	// Helper to write a line and wait for it to be processed
	writeLine := func(line string) {
		_, _ = pw.Write([]byte(line + "\n"))
	}

	awaitStatus := func(want SessionStatus) SessionStatus {
		deadline := time.After(500 * time.Millisecond)
		for {
			m.mu.RLock()
			s := m.sessions["bean-multi-turn"].Status
			m.mu.RUnlock()
			if s == want {
				return s
			}
			select {
			case <-deadline:
				return s
			case <-time.After(time.Millisecond):
			}
		}
	}

	// Turn 1: text delta + result → should go Idle
	writeLine(`{"type":"content_block_start","content_block":{"type":"text","text":""}}`)
	writeLine(`{"type":"content_block_delta","delta":{"type":"text_delta","text":"Turn 1"}}`)
	writeLine(`{"type":"result","session_id":"sess-1"}`)

	status := awaitStatus(StatusIdle)
	if status != StatusIdle {
		t.Fatalf("after turn 1 result, expected Idle, got %s", status)
	}

	// Turn 2: new text delta arrives → should transition back to Running
	writeLine(`{"type":"content_block_start","content_block":{"type":"text","text":""}}`)

	status = awaitStatus(StatusRunning)
	if status != StatusRunning {
		t.Fatalf("after turn 2 starts, expected Running, got %s", status)
	}

	// Turn 2 completes
	writeLine(`{"type":"content_block_delta","delta":{"type":"text_delta","text":"Turn 2"}}`)
	writeLine(`{"type":"result","session_id":"sess-1"}`)

	status = awaitStatus(StatusIdle)
	if status != StatusIdle {
		t.Fatalf("after turn 2 result, expected Idle, got %s", status)
	}

	// Close the pipe to let readOutput exit
	pw.Close()
	<-done
}

// TestReadOutputMultipleTools verifies ordering with multiple tool uses in a single turn.
func TestReadOutputMultipleTools(t *testing.T) {
	lines := strings.Join([]string{
		`{"type":"content_block_start","content_block":{"type":"text","text":""}}`,
		`{"type":"content_block_delta","delta":{"type":"text_delta","text":"Step 1"}}`,
		`{"type":"content_block_start","content_block":{"type":"tool_use","name":"Bash"}}`,
		`{"type":"content_block_start","content_block":{"type":"text","text":""}}`,
		`{"type":"content_block_delta","delta":{"type":"text_delta","text":"Step 2"}}`,
		`{"type":"content_block_start","content_block":{"type":"tool_use","name":"Read"}}`,
		`{"type":"content_block_start","content_block":{"type":"text","text":""}}`,
		`{"type":"content_block_delta","delta":{"type":"text_delta","text":"Step 3"}}`,
		`{"type":"result","session_id":"sess-2"}`,
	}, "\n")

	m := &Manager{
		sessions:    make(map[string]*Session),
		processes:   make(map[string]*runningProcess),
		subscribers: make(map[string][]chan struct{}),
	}

	session := &Session{
		ID:           "bean-multi",
		AgentType:    "claude",
		Status:       StatusRunning,
		Messages:     []Message{{Role: RoleUser, Content: "do stuff"}},
		streamingIdx: -1,
	}
	m.sessions["bean-multi"] = session
	proc := &runningProcess{done: make(chan struct{})}
	m.processes["bean-multi"] = proc

	m.readOutput("bean-multi", strings.NewReader(lines), "", proc)

	// Expected: USER, ASSISTANT(Step 1), TOOL(Bash), ASSISTANT(Step 2), TOOL(Read), ASSISTANT(Step 3)
	msgs := session.Messages
	if len(msgs) != 6 {
		t.Fatalf("expected 6 messages, got %d", len(msgs))
	}

	expected := []struct {
		role    MessageRole
		contain string
	}{
		{RoleUser, "do stuff"},
		{RoleAssistant, "Step 1"},
		{RoleTool, "Bash"},
		{RoleAssistant, "Step 2"},
		{RoleTool, "Read"},
		{RoleAssistant, "Step 3"},
	}

	for i, tt := range expected {
		if msgs[i].Role != tt.role {
			t.Errorf("msgs[%d].Role = %q, want %q", i, msgs[i].Role, tt.role)
		}
		if !strings.Contains(msgs[i].Content, tt.contain) {
			t.Errorf("msgs[%d].Content = %q, want it to contain %q", i, msgs[i].Content, tt.contain)
		}
	}
}

// TestReadOutputAskUserQuestionStaysIdle verifies that after AskUserQuestion is
// handled, the session stays IDLE even if more events arrive before the process exits.
func TestReadOutputAskUserQuestionStaysIdle(t *testing.T) {
	pr, pw := io.Pipe()

	m := &Manager{
		sessions:    make(map[string]*Session),
		processes:   make(map[string]*runningProcess),
		subscribers: make(map[string][]chan struct{}),
	}

	session := &Session{
		ID:           "bean-ask",
		AgentType:    "claude",
		Status:       StatusRunning,
		Messages:     []Message{{Role: RoleUser, Content: "hello"}},
		streamingIdx: -1,
	}
	m.sessions["bean-ask"] = session
	proc := &runningProcess{done: make(chan struct{})}
	m.processes["bean-ask"] = proc

	done := make(chan struct{})
	go func() {
		defer close(done)
		m.readOutput("bean-ask", pr, "", proc)
	}()

	writeLine := func(line string) {
		_, _ = pw.Write([]byte(line + "\n"))
	}

	awaitStatus := func(want SessionStatus) SessionStatus {
		deadline := time.After(500 * time.Millisecond)
		for {
			m.mu.RLock()
			s := m.sessions["bean-ask"].Status
			m.mu.RUnlock()
			if s == want {
				return s
			}
			select {
			case <-deadline:
				return s
			case <-time.After(time.Millisecond):
			}
		}
	}

	// Agent starts working, then invokes AskUserQuestion
	writeLine(`{"type":"content_block_start","content_block":{"type":"text","text":""}}`)
	writeLine(`{"type":"content_block_delta","delta":{"type":"text_delta","text":"Let me ask"}}`)
	writeLine(`{"type":"content_block_start","content_block":{"type":"tool_use","name":"AskUserQuestion"}}`)
	writeLine(`{"type":"content_block_delta","delta":{"type":"input_json_delta","partial_json":"{\"question\":\"Pick one\",\"options\":[{\"value\":\"a\",\"label\":\"Option A\"}]}"}}`)
	// Next non-delta event triggers deferred AskUserQuestion handling
	writeLine(`{"type":"content_block_start","content_block":{"type":"text","text":""}}`)

	status := awaitStatus(StatusIdle)
	if status != StatusIdle {
		t.Fatalf("after AskUserQuestion, expected Idle, got %s", status)
	}

	// Verify pending interaction was set
	m.mu.RLock()
	pending := m.sessions["bean-ask"].PendingInteraction
	m.mu.RUnlock()
	if pending == nil || pending.Type != InteractionAskUser {
		t.Fatalf("expected AskUser pending interaction, got %v", pending)
	}

	// More events arrive (process hasn't exited yet) — status must NOT flip back to Running
	writeLine(`{"type":"content_block_delta","delta":{"type":"text_delta","text":"trailing"}}`)
	writeLine(`{"type":"result","session_id":"sess-1"}`)

	// Give time for events to be processed
	time.Sleep(50 * time.Millisecond)

	m.mu.RLock()
	finalStatus := m.sessions["bean-ask"].Status
	m.mu.RUnlock()
	if finalStatus != StatusIdle {
		t.Errorf("after trailing events, expected Idle, got %s", finalStatus)
	}

	pw.Close()
	<-done
}

// TestReadOutputStaleProcessDoesNotResetStatus verifies that when a process has
// been replaced (e.g. after autoApproveModeSwitch), its result event does NOT
// reset the session status to Idle if a new process is already running.
func TestReadOutputStaleProcessDoesNotResetStatus(t *testing.T) {
	pr, pw := io.Pipe()

	m := &Manager{
		sessions:    make(map[string]*Session),
		processes:   make(map[string]*runningProcess),
		subscribers: make(map[string][]chan struct{}),
	}

	session := &Session{
		ID:           "bean-stale",
		AgentType:    "claude",
		Status:       StatusRunning,
		Messages:     []Message{{Role: RoleUser, Content: "hello"}},
		streamingIdx: -1,
	}
	m.sessions["bean-stale"] = session
	oldProc := &runningProcess{done: make(chan struct{})}
	m.processes["bean-stale"] = oldProc

	done := make(chan struct{})
	go func() {
		defer close(done)
		m.readOutput("bean-stale", pr, "", oldProc)
	}()

	writeLine := func(line string) {
		_, _ = pw.Write([]byte(line + "\n"))
	}

	awaitStatus := func(want SessionStatus) SessionStatus {
		deadline := time.After(500 * time.Millisecond)
		for {
			m.mu.RLock()
			s := m.sessions["bean-stale"].Status
			m.mu.RUnlock()
			if s == want {
				return s
			}
			select {
			case <-deadline:
				return s
			case <-time.After(time.Millisecond):
			}
		}
	}

	// First turn produces text
	writeLine(`{"type":"content_block_start","content_block":{"type":"text","text":""}}`)
	writeLine(`{"type":"content_block_delta","delta":{"type":"text_delta","text":"planning..."}}`)

	// Simulate process replacement: remove old proc from map and add a new one
	// (as autoApproveModeSwitch would do)
	m.mu.Lock()
	newProc := &runningProcess{done: make(chan struct{})}
	m.processes["bean-stale"] = newProc
	session.Status = StatusRunning // new process sets this
	m.mu.Unlock()

	// Old process emits a result event as it dies
	writeLine(`{"type":"result","session_id":"sess-1"}`)

	// Give time for the event to be processed
	time.Sleep(50 * time.Millisecond)

	// Status should still be Running (new process is active), NOT reset to Idle
	status := awaitStatus(StatusRunning)
	if status != StatusRunning {
		t.Fatalf("after stale result event, expected Running (new process active), got %s", status)
	}

	pw.Close()
	<-done
}
