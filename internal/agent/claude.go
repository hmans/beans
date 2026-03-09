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
)

// runningProcess wraps an active claude CLI process.
type runningProcess struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	cancel context.CancelFunc
}

// kill terminates the running process.
func (p *runningProcess) kill() {
	if p.cancel != nil {
		p.cancel()
	}
	if p.cmd != nil && p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
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

	proc := &runningProcess{cmd: cmd, stdin: stdin, cancel: cancel}

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

	log.Printf("[agent:%s] spawned claude process (pid=%d, dir=%s)", beanID, cmd.Process.Pid, session.WorkDir)

	// Send the initial user message
	lastMsg := session.Messages[len(session.Messages)-1]
	if err := m.sendToProcess(proc, lastMsg.Content); err != nil {
		m.setError(beanID, fmt.Sprintf("send initial message: %v", err))
		proc.kill()
		return
	}

	// Read stdout line by line
	m.readOutput(beanID, stdout)

	// Process exited — clean up
	_ = cmd.Wait()

	m.mu.Lock()
	delete(m.processes, beanID)
	if s, ok := m.sessions[beanID]; ok && s.Status == StatusRunning {
		s.Status = StatusIdle
	}
	m.mu.Unlock()

	m.notify(beanID)
}

// readOutput reads Claude Code's stream-json output line by line,
// updates the session state, and notifies subscribers.
func (m *Manager) readOutput(beanID string, stdout io.Reader) {
	scanner := bufio.NewScanner(stdout)
	// Increase buffer for long lines (1MB)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		ev := parseStreamLine(line)

		switch ev.Type {
		case eventAssistantMessage:
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

		case eventTextDelta:
			// Streaming text delta (with --include-partial-messages)
			m.appendAssistantText(beanID, ev.Text)
			m.notify(beanID)

		case eventResult:
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
				// Reset streaming target so next turn creates a new message
				s.streamingIdx = -1
				s.Status = StatusIdle
			}
			m.mu.Unlock()
			m.notify(beanID)

		case eventError:
			m.setError(beanID, ev.Error)
		}
	}
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

// buildClaudeArgs constructs the CLI arguments for spawning claude.
func buildClaudeArgs(session *Session) []string {
	args := []string{
		"-p",
		"--verbose",
		"--output-format", "stream-json",
		"--input-format", "stream-json",
		"--include-partial-messages",
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
