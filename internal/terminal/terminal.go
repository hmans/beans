package terminal

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sync"

	gopty "github.com/aymanbagabas/go-pty"
)

const scrollbackSize = 64 * 1024 // 64KB

// RingBuffer is a fixed-size circular buffer for terminal scrollback.
type RingBuffer struct {
	buf []byte
	cap int
	w   int // write position
	len int // bytes stored (capped at cap)
}

// NewRingBuffer creates a ring buffer with the given capacity.
func NewRingBuffer(size int) *RingBuffer {
	return &RingBuffer{buf: make([]byte, size), cap: size}
}

// Write appends data to the ring buffer, overwriting oldest data if full.
func (r *RingBuffer) Write(p []byte) {
	if len(p) == 0 {
		return
	}

	if len(p) >= r.cap {
		// Data larger than buffer — keep only the last cap bytes
		copy(r.buf, p[len(p)-r.cap:])
		r.w = 0
		r.len = r.cap
		return
	}

	// Copy in up to two segments (before and after wrap)
	n := copy(r.buf[r.w:], p)
	if n < len(p) {
		copy(r.buf, p[n:])
	}
	r.w = (r.w + len(p)) % r.cap
	r.len += len(p)
	if r.len > r.cap {
		r.len = r.cap
	}
}

// Bytes returns the buffer contents in chronological order.
func (r *RingBuffer) Bytes() []byte {
	if r.len == 0 {
		return nil
	}
	if r.len < r.cap {
		return append([]byte(nil), r.buf[:r.len]...)
	}
	// Wrapped: oldest data starts at write position
	result := make([]byte, r.cap)
	copy(result, r.buf[r.w:])
	copy(result[r.cap-r.w:], r.buf[:r.w])
	return result
}

// Session represents an active PTY session with scrollback buffering.
type Session struct {
	id  string
	pty gopty.Pty
	cmd *gopty.Cmd
	mu  sync.Mutex

	scrollback *RingBuffer
	scrollMu   sync.Mutex

	// Client attachment: the current output channel for a connected WebSocket.
	// Protected by attachMu. Nil when no client is attached.
	attachMu sync.Mutex
	clientCh chan []byte

	done chan struct{} // closed when readLoop exits (shell exited)
}

// Write sends input to the PTY.
func (s *Session) Write(data []byte) (int, error) {
	return s.pty.Write(data)
}

// Resize changes the PTY window size.
func (s *Session) Resize(cols, rows uint16) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pty.Resize(int(cols), int(rows))
}

// Attach connects a client to receive PTY output.
// Returns the current scrollback contents and a channel for new output.
// Calling Attach again replaces the previous client (the old channel is orphaned).
func (s *Session) Attach() ([]byte, <-chan []byte) {
	s.attachMu.Lock()
	defer s.attachMu.Unlock()

	ch := make(chan []byte, 256)
	s.clientCh = ch

	s.scrollMu.Lock()
	sb := s.scrollback.Bytes()
	s.scrollMu.Unlock()

	return sb, ch
}

// Detach disconnects the current client if it matches the given channel.
// This prevents a slow-to-exit old handler from clearing a new client's channel.
func (s *Session) Detach(ch <-chan []byte) {
	s.attachMu.Lock()
	defer s.attachMu.Unlock()

	// Only clear if this is still the active client
	if s.clientCh != nil {
		// Compare by reading from the same underlying channel
		var asChan <-chan []byte = s.clientCh
		if asChan == ch {
			s.clientCh = nil
		}
	}
}

// Done returns a channel that is closed when the shell process exits.
func (s *Session) Done() <-chan struct{} {
	return s.done
}

// Alive returns true if the shell process is still running.
func (s *Session) Alive() bool {
	select {
	case <-s.done:
		return false
	default:
		return true
	}
}

// readLoop continuously reads PTY output, writes to scrollback buffer,
// and forwards to the attached client channel.
func (s *Session) readLoop() {
	defer close(s.done)
	buf := make([]byte, 4096)
	for {
		n, err := s.pty.Read(buf)
		if n > 0 {
			data := make([]byte, n)
			copy(data, buf[:n])

			s.scrollMu.Lock()
			s.scrollback.Write(data)
			s.scrollMu.Unlock()

			s.attachMu.Lock()
			ch := s.clientCh
			s.attachMu.Unlock()

			if ch != nil {
				select {
				case ch <- data:
				default:
					// Client too slow — data is preserved in scrollback
				}
			}
		}
		if err != nil {
			return
		}
	}
}

// Close kills the process group and closes the PTY. go-pty sets Setsid on
// every spawned command, so the PID is always the process group leader.
func (s *Session) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cmd.Process != nil {
		killProcessGroup(s.cmd.Process.Pid, s.done)
	}
	_ = s.pty.Close()
	_ = s.cmd.Wait()
}

// closeSlave closes the slave end of a Unix PTY in the parent process after
// the child has been started. This is necessary on Linux so the master gets
// EOF when the child exits. On non-Unix platforms this is a no-op.
func closeSlave(p gopty.Pty) {
	if up, ok := p.(gopty.UnixPty); ok {
		up.Slave().Close()
	}
}

// defaultShell returns the default shell for the current platform.
func defaultShell() string {
	if runtime.GOOS == "windows" {
		if ps, err := exec.LookPath("pwsh.exe"); err == nil {
			return ps
		}
		return "cmd.exe"
	}
	if shell := os.Getenv("SHELL"); shell != "" {
		return shell
	}
	return "/bin/sh"
}

// EnvFunc returns extra environment variables for a given session ID.
// Used to inject workspace-specific env vars (e.g. BEANS_WORKSPACE_PORT).
type EnvFunc func(sessionID string) []string

// Manager manages PTY sessions keyed by session ID.
type Manager struct {
	mu       sync.Mutex
	sessions map[string]*Session
	envFunc  EnvFunc
}

// NewManager creates a new terminal session manager.
// An optional EnvFunc can provide extra environment variables per session.
func NewManager(envFunc EnvFunc) *Manager {
	return &Manager{sessions: make(map[string]*Session), envFunc: envFunc}
}

// Create spawns a new PTY session, replacing any existing session with the same ID.
func (m *Manager) Create(sessionID, workDir string, cols, rows uint16) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if existing, ok := m.sessions[sessionID]; ok {
		existing.Close()
		delete(m.sessions, sessionID)
	}

	return m.createLocked(sessionID, workDir, cols, rows)
}

// GetOrCreate returns an existing alive session or creates a new one.
// The bool return indicates whether an existing session was reused (reconnection).
func (m *Manager) GetOrCreate(sessionID, workDir string, cols, rows uint16) (*Session, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if existing, ok := m.sessions[sessionID]; ok {
		if existing.Alive() {
			_ = existing.Resize(cols, rows)
			return existing, true, nil
		}
		// Shell exited — clean up dead session
		existing.Close()
		delete(m.sessions, sessionID)
	}

	sess, err := m.createLocked(sessionID, workDir, cols, rows)
	if err != nil {
		return nil, false, err
	}
	return sess, false, nil
}

func (m *Manager) createLocked(sessionID, workDir string, cols, rows uint16) (*Session, error) {
	shell := defaultShell()

	p, err := gopty.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create PTY: %w", err)
	}

	if err := p.Resize(int(cols), int(rows)); err != nil {
		p.Close()
		return nil, fmt.Errorf("failed to resize PTY: %w", err)
	}

	env := append(os.Environ(), "TERM=xterm-256color")
	if m.envFunc != nil {
		env = append(env, m.envFunc(sessionID)...)
	}

	cmd := p.Command(shell, "-l")
	cmd.Dir = workDir
	cmd.Env = env

	if err := cmd.Start(); err != nil {
		p.Close()
		return nil, fmt.Errorf("failed to start PTY: %w", err)
	}

	closeSlave(p)

	sess := &Session{
		id:         sessionID,
		pty:        p,
		cmd:        cmd,
		scrollback: NewRingBuffer(scrollbackSize),
		done:       make(chan struct{}),
	}

	go sess.readLoop()

	m.sessions[sessionID] = sess
	return sess, nil
}

// CreateWithCommand spawns a new PTY session that runs a specific command
// (via shell -c) instead of an interactive login shell. The session ends
// when the command exits.
func (m *Manager) CreateWithCommand(sessionID, workDir string, cols, rows uint16, command string) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if existing, ok := m.sessions[sessionID]; ok {
		existing.Close()
		delete(m.sessions, sessionID)
	}

	shell := defaultShell()

	p, err := gopty.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create PTY: %w", err)
	}

	if err := p.Resize(int(cols), int(rows)); err != nil {
		p.Close()
		return nil, fmt.Errorf("failed to resize PTY: %w", err)
	}

	env := append(os.Environ(), "TERM=xterm-256color")
	if m.envFunc != nil {
		env = append(env, m.envFunc(sessionID)...)
	}

	cmd := p.Command(shell, "-l", "-c", command)
	cmd.Dir = workDir
	cmd.Env = env

	if err := cmd.Start(); err != nil {
		p.Close()
		return nil, fmt.Errorf("failed to start PTY: %w", err)
	}

	closeSlave(p)

	sess := &Session{
		id:         sessionID,
		pty:        p,
		cmd:        cmd,
		scrollback: NewRingBuffer(scrollbackSize),
		done:       make(chan struct{}),
	}

	go sess.readLoop()

	m.sessions[sessionID] = sess
	return sess, nil
}

// Get retrieves an existing session.
func (m *Manager) Get(sessionID string) *Session {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sessions[sessionID]
}

// Close closes and removes a specific session.
func (m *Manager) Close(sessionID string) {
	m.mu.Lock()
	sess, ok := m.sessions[sessionID]
	if ok {
		delete(m.sessions, sessionID)
	}
	m.mu.Unlock()
	if ok {
		sess.Close()
	}
}

// Shutdown closes all active sessions.
func (m *Manager) Shutdown() {
	m.mu.Lock()
	sessions := make(map[string]*Session)
	for k, v := range m.sessions {
		sessions[k] = v
	}
	m.sessions = make(map[string]*Session)
	m.mu.Unlock()

	for _, s := range sessions {
		s.Close()
	}
}
