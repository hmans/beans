package agent

import (
	"log"
	"sync"
)

// Manager manages agent sessions — one per worktree (keyed by beanID).
// It holds sessions in memory and provides pub/sub for session updates.
type Manager struct {
	mu        sync.RWMutex
	sessions  map[string]*Session
	processes map[string]*runningProcess
	store     *store // JSONL persistence (nil if no beansDir)

	subMu       sync.Mutex
	subscribers map[string][]chan struct{}
}

// NewManager creates a new agent session manager.
// If beansDir is non-empty, conversations are persisted to .beans/conversations/.
func NewManager(beansDir string) *Manager {
	m := &Manager{
		sessions:    make(map[string]*Session),
		processes:   make(map[string]*runningProcess),
		subscribers: make(map[string][]chan struct{}),
	}

	if beansDir != "" {
		s, err := newStore(beansDir)
		if err != nil {
			log.Printf("[agent] warning: conversation persistence disabled: %v", err)
		} else {
			m.store = s
		}
	}

	return m
}

// GetSession returns a snapshot of the session for the given beanID, or nil.
// If no in-memory session exists but a persisted conversation is found, it is loaded.
func (m *Manager) GetSession(beanID string) *Session {
	m.mu.RLock()
	s, ok := m.sessions[beanID]
	m.mu.RUnlock()

	if !ok {
		// Try loading from disk
		if m.store == nil {
			return nil
		}
		msgs, sessionID, err := m.store.load(beanID)
		if err != nil || len(msgs) == 0 {
			return nil
		}
		// Materialize the session in memory
		m.mu.Lock()
		// Double-check another goroutine didn't create it
		if s2, ok2 := m.sessions[beanID]; ok2 {
			snap := s2.snapshot()
			m.mu.Unlock()
			return &snap
		}
		s = &Session{
			ID:           beanID,
			AgentType:    "claude",
			Status:       StatusIdle,
			Messages:     msgs,
			SessionID:    sessionID,
			streamingIdx: -1,
		}
		m.sessions[beanID] = s
		m.mu.Unlock()
	}

	m.mu.RLock()
	snap := s.snapshot()
	m.mu.RUnlock()
	return &snap
}

// SendMessage sends a user message to the agent for the given worktree.
// If no session exists, one is created. If no process is running, one is spawned.
func (m *Manager) SendMessage(beanID, workDir, message string) error {
	m.mu.Lock()

	// Get or create session
	session, ok := m.sessions[beanID]
	if !ok {
		session = m.loadOrCreateSession(beanID, workDir)
		m.sessions[beanID] = session
	}

	// Ensure WorkDir is set (may be empty if loaded from disk by GetSession)
	if session.WorkDir == "" && workDir != "" {
		session.WorkDir = workDir
	}

	// Append user message
	userMsg := Message{Role: RoleUser, Content: message}
	session.Messages = append(session.Messages, userMsg)
	session.Error = ""

	// Persist user message
	if m.store != nil {
		if err := m.store.appendMessage(beanID, userMsg); err != nil {
			log.Printf("[agent:%s] failed to persist user message: %v", beanID, err)
		}
	}

	// Check if we have a running process
	proc, hasProc := m.processes[beanID]
	session.Status = StatusRunning
	m.mu.Unlock()

	// Notify subscribers that we have a new user message + running status
	m.notify(beanID)

	if hasProc && proc != nil {
		// Send message to existing process via stdin — Claude Code's stream-json
		// protocol handles interleaving even if the agent is mid-turn
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

// loadOrCreateSession loads a session from disk if persisted, or creates a new one.
// Must be called with m.mu held.
func (m *Manager) loadOrCreateSession(beanID, workDir string) *Session {
	session := &Session{
		ID:           beanID,
		AgentType:    "claude",
		Status:       StatusIdle,
		WorkDir:      workDir,
		streamingIdx: -1,
	}

	if m.store != nil {
		msgs, sessionID, err := m.store.load(beanID)
		if err != nil {
			log.Printf("[agent:%s] failed to load conversation: %v", beanID, err)
		} else if len(msgs) > 0 {
			session.Messages = msgs
			session.SessionID = sessionID
		}
	}

	return session
}
