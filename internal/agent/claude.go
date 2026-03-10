package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

// runningProcess wraps an active claude CLI process.
type runningProcess struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	cancel context.CancelFunc
	done   chan struct{} // closed by spawnAndRun when the process exits
}

// signal sends SIGINT and closes stdin to request a graceful shutdown.
// Does not block — the process may still be running when this returns.
// Used by handleBlockingTool (which runs inside the readOutput goroutine
// and cannot wait for the process to exit without deadlocking).
func (p *runningProcess) signal() {
	if p.cmd == nil || p.cmd.Process == nil {
		return
	}
	if p.stdin != nil {
		_ = p.stdin.Close()
	}
	_ = p.cmd.Process.Signal(syscall.SIGINT)
}

// kill gracefully terminates the running process and waits for it to exit.
// Sends SIGINT first to give Claude Code a chance to save its session state
// (needed for --resume to work). Falls back to SIGKILL after a timeout.
func (p *runningProcess) kill() {
	p.signal()

	// Wait up to 3 seconds for the process to exit cleanly.
	// The done channel is closed by spawnAndRun after cmd.Wait() returns.
	select {
	case <-p.done:
		// Process exited cleanly — session state should be saved
	case <-time.After(3 * time.Second):
		// Force kill after timeout
		if p.cancel != nil {
			p.cancel()
		}
		_ = p.cmd.Process.Kill()
		<-p.done // wait for spawnAndRun to finish cleanup
	}
}

// sendToProcess writes a user message to an existing process's stdin
// using Claude Code's stream-json input format.
func (m *Manager) sendToProcess(proc *runningProcess, message string) error {
	msg := map[string]interface{}{
		"type": "user",
		"message": map[string]interface{}{
			"role":    "user",
			"content": message,
		},
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}
	data = append(data, '\n')
	_, err = proc.stdin.Write(data)
	return err
}

// spawnAndRun spawns a claude CLI process and reads its output.
// This runs in a goroutine — it blocks until the process exits.
func (m *Manager) spawnAndRun(beanID string, session *Session) {
	ctx, cancel := context.WithCancel(context.Background())

	args := buildClaudeArgs(session)
	cmd := exec.CommandContext(ctx, "claude", args...)
	cmd.Dir = session.WorkDir
	cmd.Env = buildClaudeEnv()

	stdin, err := cmd.StdinPipe()
	if err != nil {
		m.setError(beanID, fmt.Sprintf("stdin pipe: %v", err))
		cancel()
		return
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		m.setError(beanID, fmt.Sprintf("stdout pipe: %v", err))
		cancel()
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		m.setError(beanID, fmt.Sprintf("stderr pipe: %v", err))
		cancel()
		return
	}

	if err := cmd.Start(); err != nil {
		m.setError(beanID, fmt.Sprintf("start claude: %v", err))
		cancel()
		return
	}

	proc := &runningProcess{cmd: cmd, stdin: stdin, cancel: cancel, done: make(chan struct{})}

	m.mu.Lock()
	m.processes[beanID] = proc
	m.mu.Unlock()

	// Log stderr in background
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			log.Printf("[agent:%s] stderr: %s", beanID, scanner.Text())
		}
	}()

	log.Printf("[agent:%s] spawned claude process (pid=%d, dir=%s, args=%v)", beanID, cmd.Process.Pid, session.WorkDir, args)

	// Send the initial user message, prepending bean context on first spawn
	initialMsg := session.Messages[len(session.Messages)-1].Content
	if session.SessionID == "" && m.contextProvider != nil {
		if ctx := m.contextProvider(beanID); ctx != "" {
			initialMsg = ctx + "\n\n---\n\n" + initialMsg
		}
	}
	if err := m.sendToProcess(proc, initialMsg); err != nil {
		m.setError(beanID, fmt.Sprintf("send initial message: %v", err))
		proc.kill()
		return
	}

	// Read stdout line by line
	m.readOutput(beanID, stdout, session.WorkDir)

	// Process exited — clean up.
	// Only modify state if this proc is still the current one for this beanID.
	// A new process may have already been spawned (e.g. after handleBlockingTool
	// signaled us and the user sent a new message), so we must not clobber it.
	_ = cmd.Wait()
	close(proc.done)

	m.mu.Lock()
	shouldNotify := false
	if m.processes[beanID] == proc {
		delete(m.processes, beanID)
		if s, ok := m.sessions[beanID]; ok && s.Status == StatusRunning {
			s.Status = StatusIdle
			shouldNotify = true
		}
	}
	m.mu.Unlock()

	if shouldNotify {
		m.notify(beanID)
	}
}

// readOutput reads Claude Code's stream-json output line by line,
// updates the session state, and notifies subscribers.
func (m *Manager) readOutput(beanID string, stdout io.Reader, workDir string) {
	scanner := bufio.NewScanner(stdout)
	// Increase buffer for long lines (1MB)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	// Track tool input accumulation for extracting summaries.
	// Tool messages are persisted lazily: we wait until the summary is available
	// (or the next event arrives) before writing to JSONL, so the persisted
	// content includes the human-readable description (e.g. "Bash: Build binary").
	var toolInputBuf strings.Builder
	var toolMsgIdx int = -1
	var toolName string
	var toolInvIdx int = -1 // index into session.ToolInvocations
	var pendingToolPersist bool // true when current tool msg hasn't been persisted yet

	// flushToolMsg persists the current tool message to JSONL if one is pending.
	// Called before persisting any other message or at end of stream.
	flushToolMsg := func() {
		if !pendingToolPersist || m.store == nil {
			pendingToolPersist = false
			return
		}
		pendingToolPersist = false
		m.mu.RLock()
		s, ok := m.sessions[beanID]
		var msg Message
		if ok && toolMsgIdx >= 0 && toolMsgIdx < len(s.Messages) {
			msg = s.Messages[toolMsgIdx]
		}
		m.mu.RUnlock()
		if msg.Role == RoleTool {
			if err := m.store.appendMessage(beanID, msg); err != nil {
				log.Printf("[agent:%s] failed to persist tool message: %v", beanID, err)
			}
		}
	}

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		ev := parseStreamLine(line)
		if ev.Type == eventUnknown {
			log.Printf("[agent:%s] unhandled event: %s", beanID, string(line))
		}

		switch ev.Type {
		case eventAssistantMessage:
			// Flush any pending tool message before the assistant message
			flushToolMsg()

			// Full assistant message — arrives after stream_event deltas.
			// Only use the text as fallback if deltas didn't already build it,
			// to avoid replacing streamed content with the same text (visual flash).
			if ev.Text != "" {
				m.mu.Lock()
				if s, ok := m.sessions[beanID]; ok {
					idx := s.streamingIdx
					hasStreamedContent := idx >= 0 && idx < len(s.Messages) &&
						s.Messages[idx].Role == RoleAssistant && s.Messages[idx].Content != ""
					if !hasStreamedContent {
						// No delta-built content — use the full message
						s.Messages = append(s.Messages, Message{Role: RoleAssistant, Content: ev.Text})
						s.streamingIdx = len(s.Messages) - 1
						m.mu.Unlock()
						m.notify(beanID)
					} else {
						m.mu.Unlock()
					}
				} else {
					m.mu.Unlock()
				}
			}
			if ev.SessionID != "" {
				m.mu.Lock()
				if s, ok := m.sessions[beanID]; ok {
					s.SessionID = ev.SessionID
				}
				m.mu.Unlock()
			}

		case eventToolUse:
			// Handle blocking tools that require user interaction.
			// Check session state to avoid re-intercepting mode switches
			// that already took effect (e.g. after --resume).
			m.mu.RLock()
			sess := m.sessions[beanID]
			m.mu.RUnlock()
			if interaction := blockingInteraction(ev.ToolName, sess); interaction != nil {
				m.handleBlockingTool(beanID, interaction)
			}

			// Flush any previous pending tool message before starting a new one
			flushToolMsg()

			// Tool use start — show tool name in the conversation.
			// Reset streamingIdx so subsequent text deltas create a
			// new assistant message *after* this tool message, preserving
			// chronological order.
			toolInputBuf.Reset()
			toolName = ev.ToolName
			m.mu.Lock()
			if s, ok := m.sessions[beanID]; ok {
				// Persist the pre-tool assistant message before resetting
				idx := s.streamingIdx
				if m.store != nil && idx >= 0 && idx < len(s.Messages) && s.Messages[idx].Role == RoleAssistant && s.Messages[idx].Content != "" {
					msg := s.Messages[idx]
					m.mu.Unlock()
					if err := m.store.appendMessage(beanID, msg); err != nil {
						log.Printf("[agent:%s] failed to persist pre-tool assistant message: %v", beanID, err)
					}
					m.mu.Lock()
					// Re-check session still exists after re-acquiring lock
					s = m.sessions[beanID]
					if s == nil {
						m.mu.Unlock()
						m.notify(beanID)
						continue
					}
				}
				s.streamingIdx = -1
				toolMsg := Message{Role: RoleTool, Content: ev.ToolName}
				s.Messages = append(s.Messages, toolMsg)
				toolMsgIdx = len(s.Messages) - 1
				// Track structured tool invocation
				s.ToolInvocations = append(s.ToolInvocations, ToolInvocation{Tool: ev.ToolName})
				toolInvIdx = len(s.ToolInvocations) - 1
				// Don't persist yet — wait for tool input summary
				pendingToolPersist = true
			}
			m.mu.Unlock()
			m.notify(beanID)

		case eventToolInputDelta:
			// Accumulate tool input JSON and try to extract a summary
			toolInputBuf.WriteString(ev.Text)
			if toolMsgIdx >= 0 {
				// Try parsing accumulated JSON (may be incomplete — that's fine)
				summary := extractToolSummary(toolInputBuf.String(), workDir)
				if summary != "" {
					m.mu.Lock()
					if s, ok := m.sessions[beanID]; ok && toolMsgIdx < len(s.Messages) {
						s.Messages[toolMsgIdx].Content = toolName + ": " + summary
						// Update structured tool invocation input
						if toolInvIdx >= 0 && toolInvIdx < len(s.ToolInvocations) {
							s.ToolInvocations[toolInvIdx].Input = summary
						}
					}
					m.mu.Unlock()
					m.notify(beanID)
				}
			}

		case eventNewTextBlock:
			// Flush any pending tool message before new text
			flushToolMsg()

			// New text content block starting — insert paragraph break if
			// the current message already has content (e.g. after tool use).
			m.mu.Lock()
			if s, ok := m.sessions[beanID]; ok {
				idx := s.streamingIdx
				if idx >= 0 && idx < len(s.Messages) &&
					s.Messages[idx].Role == RoleAssistant && s.Messages[idx].Content != "" {
					s.Messages[idx].Content += "\n\n"
				}
			}
			m.mu.Unlock()
			if ev.Text != "" {
				m.appendAssistantText(beanID, ev.Text)
				m.notify(beanID)
			}

		case eventTextDelta:
			// Flush any pending tool message before text starts
			flushToolMsg()

			// Streaming text delta (with --include-partial-messages)
			m.appendAssistantText(beanID, ev.Text)
			m.notify(beanID)

		case eventResult:
			// Flush any pending tool message before result
			flushToolMsg()

			if ev.SessionID != "" {
				m.mu.Lock()
				if s, ok := m.sessions[beanID]; ok {
					s.SessionID = ev.SessionID
				}
				m.mu.Unlock()

				// Persist session ID for --resume
				if m.store != nil {
					if err := m.store.saveSessionID(beanID, ev.SessionID); err != nil {
						log.Printf("[agent:%s] failed to persist session ID: %v", beanID, err)
					}
				}
			}

			// Persist the completed assistant message and reset streaming target
			m.mu.Lock()
			if s, ok := m.sessions[beanID]; ok {
				idx := s.streamingIdx
				if m.store != nil && idx >= 0 && idx < len(s.Messages) && s.Messages[idx].Role == RoleAssistant {
					msg := s.Messages[idx]
					m.mu.Unlock()
					if err := m.store.appendMessage(beanID, msg); err != nil {
						log.Printf("[agent:%s] failed to persist assistant message: %v", beanID, err)
					}
					m.mu.Lock()
				}
				// Reset streaming target and transient state
				s.streamingIdx = -1
				s.Status = StatusIdle
				s.SystemStatus = ""

				// If there were permission denials, set a pending interaction
				// so the UI can show the approval dialog
				if len(ev.PermissionDenials) > 0 {
					for _, d := range ev.PermissionDenials {
						log.Printf("[agent:%s] permission denied: tool=%s", beanID, d.ToolName)
					}
					s.PendingInteraction = &PendingInteraction{
						Type:              InteractionPermission,
						PermissionDenials: ev.PermissionDenials,
					}
				}
			}
			m.mu.Unlock()
			m.notify(beanID)

		case eventError:
			flushToolMsg()
			m.setError(beanID, ev.Error)

		case eventSystemStatus:
			m.mu.Lock()
			if s, ok := m.sessions[beanID]; ok {
				s.SystemStatus = ev.Text
			}
			m.mu.Unlock()
			m.notify(beanID)
		}
	}

	// Flush any remaining pending tool message at end of stream
	flushToolMsg()
}

// appendAssistantText appends text to the current streaming assistant message.
// Uses streamingIdx to ensure deltas from an ongoing turn always go to the
// correct message, even if user messages are interleaved mid-turn.
func (m *Manager) appendAssistantText(beanID, text string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[beanID]
	if !ok {
		return
	}

	idx := s.streamingIdx
	if idx < 0 || idx >= len(s.Messages) || s.Messages[idx].Role != RoleAssistant {
		// No valid streaming target — create a new assistant message
		s.Messages = append(s.Messages, Message{Role: RoleAssistant})
		idx = len(s.Messages) - 1
		s.streamingIdx = idx
	}

	s.Messages[idx].Content += text
}

// setError sets the session to error state and notifies subscribers.
func (m *Manager) setError(beanID, errMsg string) {
	m.mu.Lock()
	if s, ok := m.sessions[beanID]; ok {
		s.Status = StatusError
		s.Error = errMsg
	}
	m.mu.Unlock()
	m.notify(beanID)
}

// blockingInteraction returns a PendingInteraction if the tool name is a blocking
// tool that requires user approval, or nil for regular tools.
// Mode-switch tools (ExitPlanMode/EnterPlanMode) are only blocking when the
// session is actually in the mode being exited/entered — this prevents infinite
// loops when a resumed process retries the same tool call after we already
// toggled the mode.
func blockingInteraction(toolName string, session *Session) *PendingInteraction {
	switch toolName {
	case "ExitPlanMode":
		if session != nil && !session.PlanMode {
			return nil // already exited plan mode (e.g. after resume)
		}
		return &PendingInteraction{Type: InteractionExitPlan}
	case "EnterPlanMode":
		if session != nil && session.PlanMode {
			return nil // already in plan mode
		}
		return &PendingInteraction{Type: InteractionEnterPlan}
	case "AskUserQuestion":
		return &PendingInteraction{Type: InteractionAskUser}
	default:
		return nil
	}
}

// handleBlockingTool processes a blocking tool call by setting the pending
// interaction, killing the process, and notifying subscribers. The session ID
// is preserved so the conversation can be resumed with --resume.
func (m *Manager) handleBlockingTool(beanID string, interaction *PendingInteraction) {
	m.mu.Lock()
	s, ok := m.sessions[beanID]
	if !ok {
		m.mu.Unlock()
		return
	}

	// For ExitPlanMode, find and read the plan file from recent Write tool messages
	if interaction.Type == InteractionExitPlan {
		if path := findPlanFilePath(s.ToolInvocations); path != "" {
			if content, err := os.ReadFile(path); err == nil {
				interaction.PlanContent = string(content)
			}
		}
	}

	s.PendingInteraction = interaction
	s.Status = StatusIdle

	// Toggle plan mode for mode-switch interactions
	switch interaction.Type {
	case InteractionExitPlan:
		s.PlanMode = false
	case InteractionEnterPlan:
		s.PlanMode = true
	case InteractionAskUser:
		// No mode change — just pause for user input
	}

	proc, hasProc := m.processes[beanID]
	if hasProc {
		delete(m.processes, beanID)
	}
	m.mu.Unlock()

	if hasProc && proc != nil {
		// Use signal() not kill() — we're inside the readOutput goroutine
		// (same goroutine as spawnAndRun), so blocking on proc.done would deadlock.
		proc.signal()
	}

	m.notify(beanID)
}

// findPlanFilePath scans tool invocations for a Write to ~/.claude/plans/*.md
// and returns the file path, or empty string if not found.
func findPlanFilePath(invocations []ToolInvocation) string {
	for i := len(invocations) - 1; i >= 0; i-- {
		inv := invocations[i]
		if inv.Tool == "Write" && strings.Contains(inv.Input, "/.claude/plans/") && strings.HasSuffix(inv.Input, ".md") {
			return inv.Input
		}
	}
	return ""
}

// buildClaudeArgs constructs the CLI arguments for spawning claude.
func buildClaudeArgs(session *Session) []string {
	args := []string{
		"-p",
		"--verbose",
		"--output-format", "stream-json",
		"--input-format", "stream-json",
		"--include-partial-messages",
	}
	if session.YoloMode {
		args = append(args, "--dangerously-skip-permissions")
	} else if session.PlanMode {
		args = append(args, "--permission-mode", "plan")
	} else {
		// Act mode: auto-approve file reads/edits, prompt for Bash and network tools.
		args = append(args, "--permission-mode", "acceptEdits")
		args = append(args, "--allowedTools", "Bash(beans:*)")
	}
	// Pass any user-approved tool patterns (accumulated from permission denials).
	for _, tool := range session.AllowedTools {
		args = append(args, "--allowedTools", tool)
	}
	if session.SessionID != "" {
		args = append(args, "--resume", session.SessionID)
	}
	return args
}

// buildClaudeEnv creates the environment for the claude process,
// stripping CLAUDECODE to allow nested sessions.
func buildClaudeEnv() []string {
	env := os.Environ()
	filtered := make([]string, 0, len(env))
	for _, e := range env {
		if strings.HasPrefix(e, "CLAUDECODE=") {
			continue
		}
		filtered = append(filtered, e)
	}
	return filtered
}
