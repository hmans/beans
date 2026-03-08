package agent

import (
	"fmt"
	"sync"
)

// Manager manages agent sessions — one per worktree (keyed by beanID).
// It holds sessions in memory and provides pub/sub for session updates.
type Manager struct {
	mu        sync.RWMutex
	sessions  map[string]*Session
	processes map[string]*runningProcess

	subMu       sync.Mutex
	subscribers map[string][]chan struct{}
}

// NewManager creates a new agent session manager.
func NewManager() *Manager {
	return &Manager{
		sessions:    make(map[string]*Session),
		processes:   make(map[string]*runningProcess),
		subscribers: make(map[string][]chan struct{}),
	}
}

// GetSession returns a snapshot of the session for the given beanID, or nil.
func (m *Manager) GetSession(beanID string) *Session {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[beanID]
	if !ok {
		return nil
	}
	snap := s.snapshot()
	return &snap
}

// SendMessage sends a user message to the agent for the given worktree.
// If no session exists, one is created. If no process is running, one is spawned.
func (m *Manager) SendMessage(beanID, workDir, message string) error {
	m.mu.Lock()

	// Get or create session
	session, ok := m.sessions[beanID]
	if !ok {
		session = &Session{
			ID:        beanID,
			AgentType: "claude",
			Status:    StatusIdle,
			WorkDir:   workDir,
		}
		m.sessions[beanID] = session
	}

	// Don't allow sending while already running
	if session.Status == StatusRunning {
		m.mu.Unlock()
		return fmt.Errorf("agent is busy")
	}

	// Append user message
	session.Messages = append(session.Messages, Message{
		Role:    RoleUser,
		Content: message,
	})
	session.Status = StatusRunning
	session.Error = ""

	// Check if we have a running process
	proc, hasProc := m.processes[beanID]
	m.mu.Unlock()

	// Notify subscribers that we have a new user message + running status
	m.notify(beanID)

	if hasProc && proc != nil {
		// Send message to existing process via stdin
		return m.sendToProcess(proc, message)
	}

	// Spawn a new process
	go m.spawnAndRun(beanID, session)
	return nil
}

// StopSession kills the running process for a session and sets it to idle.
func (m *Manager) StopSession(beanID string) error {
	m.mu.Lock()
	proc, hasProc := m.processes[beanID]
	session, hasSession := m.sessions[beanID]
	if hasSession {
		session.Status = StatusIdle
	}
	if hasProc {
		delete(m.processes, beanID)
	}
	m.mu.Unlock()

	if hasProc && proc != nil {
		proc.kill()
	}

	m.notify(beanID)
	return nil
}

// Subscribe returns a channel that receives a signal whenever the session
// for the given beanID changes. Call Unsubscribe when done.
func (m *Manager) Subscribe(beanID string) chan struct{} {
	m.subMu.Lock()
	defer m.subMu.Unlock()
	ch := make(chan struct{}, 1)
	m.subscribers[beanID] = append(m.subscribers[beanID], ch)
	return ch
}

// Unsubscribe removes a subscription channel.
func (m *Manager) Unsubscribe(beanID string, ch chan struct{}) {
	m.subMu.Lock()
	defer m.subMu.Unlock()
	subs := m.subscribers[beanID]
	for i, sub := range subs {
		if sub == ch {
			m.subscribers[beanID] = append(subs[:i], subs[i+1:]...)
			close(ch)
			return
		}
	}
}

// notify sends a signal to all subscribers for the given beanID.
func (m *Manager) notify(beanID string) {
	m.subMu.Lock()
	defer m.subMu.Unlock()
	for _, ch := range m.subscribers[beanID] {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

// Shutdown kills all running processes. Call on server shutdown.
func (m *Manager) Shutdown() {
	m.mu.Lock()
	procs := make(map[string]*runningProcess, len(m.processes))
	for k, v := range m.processes {
		procs[k] = v
	}
	m.processes = make(map[string]*runningProcess)
	m.mu.Unlock()

	for _, proc := range procs {
		proc.kill()
	}
}
