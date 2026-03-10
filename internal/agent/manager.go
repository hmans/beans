package agent

import (
	"fmt"
	"log"
	"sync"
)

// ContextProvider returns context text to inject into a new agent conversation
// for the given beanID. Return "" to skip injection.
type ContextProvider func(beanID string) string

// Manager manages agent sessions — one per worktree (keyed by beanID).
// It holds sessions in memory and provides pub/sub for session updates.
type Manager struct {
	mu              sync.RWMutex
	sessions        map[string]*Session
	processes       map[string]*runningProcess
	store           *store // JSONL persistence (nil if no beansDir)
	contextProvider ContextProvider

	subMu       sync.Mutex
	subscribers map[string][]chan struct{}

	globalSubMu       sync.Mutex
	globalSubscribers []chan struct{}
}

// NewManager creates a new agent session manager.
// If beansDir is non-empty, conversations are persisted to .beans/conversations/.
func NewManager(beansDir string, contextProvider ContextProvider) *Manager {
	m := &Manager{
		sessions:        make(map[string]*Session),
		processes:       make(map[string]*runningProcess),
		subscribers:     make(map[string][]chan struct{}),
		contextProvider: contextProvider,
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
			YoloMode:     true,
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

	// Append user message and clear turn state
	userMsg := Message{Role: RoleUser, Content: message}
	session.Messages = append(session.Messages, userMsg)
	session.Error = ""
	session.PendingInteraction = nil
	session.ToolInvocations = nil

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

// SubscribeGlobal returns a channel that receives a signal whenever any
// agent session changes. Call UnsubscribeGlobal when done.
func (m *Manager) SubscribeGlobal() chan struct{} {
	m.globalSubMu.Lock()
	defer m.globalSubMu.Unlock()
	ch := make(chan struct{}, 1)
	m.globalSubscribers = append(m.globalSubscribers, ch)
	return ch
}

// UnsubscribeGlobal removes a global subscription channel.
func (m *Manager) UnsubscribeGlobal(ch chan struct{}) {
	m.globalSubMu.Lock()
	defer m.globalSubMu.Unlock()
	for i, sub := range m.globalSubscribers {
		if sub == ch {
			m.globalSubscribers = append(m.globalSubscribers[:i], m.globalSubscribers[i+1:]...)
			close(ch)
			return
		}
	}
}

// ListRunningSessions returns the bean IDs and statuses of all in-memory sessions.
func (m *Manager) ListRunningSessions() []ActiveAgent {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []ActiveAgent
	for id, s := range m.sessions {
		if s.Status == StatusRunning {
			result = append(result, ActiveAgent{BeanID: id, Status: s.Status})
		}
	}
	return result
}

// notify sends a signal to all subscribers for the given beanID,
// and also notifies global subscribers.
func (m *Manager) notify(beanID string) {
	m.subMu.Lock()
	for _, ch := range m.subscribers[beanID] {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
	m.subMu.Unlock()

	m.globalSubMu.Lock()
	for _, ch := range m.globalSubscribers {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
	m.globalSubMu.Unlock()
}

// SetPlanMode toggles plan mode for a session, killing any running process
// since --permission-mode is a startup flag that requires respawning.
func (m *Manager) SetPlanMode(beanID string, planMode bool) error {
	m.mu.Lock()
	session, hasSession := m.sessions[beanID]
	if !hasSession {
		// Create session in memory so the mode is set before any messages
		session = &Session{
			ID:           beanID,
			AgentType:    "claude",
			Status:       StatusIdle,
			PlanMode:     planMode,
			streamingIdx: -1,
		}
		m.sessions[beanID] = session
		m.mu.Unlock()
		m.notify(beanID)
		return nil
	}

	if session.PlanMode == planMode {
		m.mu.Unlock()
		return nil
	}

	session.PlanMode = planMode

	proc, hasProc := m.processes[beanID]
	if hasProc {
		delete(m.processes, beanID)
		session.Status = StatusIdle
	}
	m.mu.Unlock()

	if hasProc && proc != nil {
		proc.kill()
	}

	m.notify(beanID)
	return nil
}

// SetYoloMode toggles YOLO mode for a session, killing any running process
// since --dangerously-skip-permissions is a startup flag that requires respawning.
func (m *Manager) SetYoloMode(beanID string, yoloMode bool) error {
	m.mu.Lock()
	session, hasSession := m.sessions[beanID]
	if !hasSession {
		session = &Session{
			ID:           beanID,
			AgentType:    "claude",
			Status:       StatusIdle,
			YoloMode:     yoloMode,
			streamingIdx: -1,
		}
		m.sessions[beanID] = session
		m.mu.Unlock()
		m.notify(beanID)
		return nil
	}

	if session.YoloMode == yoloMode {
		m.mu.Unlock()
		return nil
	}

	session.YoloMode = yoloMode

	proc, hasProc := m.processes[beanID]
	if hasProc {
		delete(m.processes, beanID)
		session.Status = StatusIdle
	}
	m.mu.Unlock()

	if hasProc && proc != nil {
		proc.kill()
	}

	m.notify(beanID)
	return nil
}

// ResolvePermission handles the user's response to a permission denial.
// If allow is true, the denied tool patterns are added to AllowedTools,
// the process is killed (so it respawns with --allowedTools), and a
// continuation message is sent. If deny, the interaction is simply cleared.
func (m *Manager) ResolvePermission(beanID string, allow bool) error {
	m.mu.Lock()
	s, ok := m.sessions[beanID]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("no session for bean %s", beanID)
	}
	pending := s.PendingInteraction
	if pending == nil || pending.Type != InteractionPermission {
		m.mu.Unlock()
		return fmt.Errorf("no pending permission request for bean %s", beanID)
	}

	if allow {
		// Add denied tool names to the allowed list for future spawns
		for _, d := range pending.PermissionDenials {
			s.AllowedTools = appendUnique(s.AllowedTools, d.ToolName)
		}
	}

	s.PendingInteraction = nil

	// Kill the running process so it respawns with updated --allowedTools
	proc, hasProc := m.processes[beanID]
	if hasProc {
		delete(m.processes, beanID)
		s.Status = StatusIdle
	}
	m.mu.Unlock()

	if hasProc && proc != nil {
		proc.kill()
	}

	m.notify(beanID)

	if allow {
		// Re-send "yes, proceed" to trigger respawn with --allowedTools + --resume
		go func() {
			_ = m.SendMessage(beanID, s.WorkDir, "yes, proceed")
		}()
	}

	return nil
}

// appendUnique adds a string to a slice if not already present.
func appendUnique(slice []string, val string) []string {
	for _, s := range slice {
		if s == val {
			return slice
		}
	}
	return append(slice, val)
}

// ClearSession stops any running process, removes the session from memory,
// and deletes the persisted conversation file.
func (m *Manager) ClearSession(beanID string) error {
	// Delete persisted conversation BEFORE removing the in-memory session
	// and killing the process. This prevents a race where spawnAndRun's
	// cleanup goroutine notifies subscribers, causing GetSession to
	// re-materialize the session from a still-existing JSONL file.
	if m.store != nil {
		if err := m.store.clear(beanID); err != nil {
			log.Printf("[agent:%s] failed to clear conversation file: %v", beanID, err)
		}
	}

	m.mu.Lock()
	proc, hasProc := m.processes[beanID]
	if hasProc {
		delete(m.processes, beanID)
	}
	delete(m.sessions, beanID)
	m.mu.Unlock()

	if hasProc && proc != nil {
		proc.kill()
	}

	m.notify(beanID)
	return nil
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
		YoloMode:     true,
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
