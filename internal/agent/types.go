// Package agent manages AI coding agent sessions within worktrees.
package agent

// MessageRole identifies who sent a message.
type MessageRole string

const (
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
)

// SessionStatus represents the current state of an agent session.
type SessionStatus string

const (
	StatusIdle    SessionStatus = "idle"
	StatusRunning SessionStatus = "running"
	StatusError   SessionStatus = "error"
)

// Message represents a single chat message in an agent conversation.
type Message struct {
	Role    MessageRole
	Content string
}

// Session represents an active or idle agent conversation for a worktree.
type Session struct {
	ID        string        // beanID — one session per worktree
	AgentType string        // "claude" for now
	SessionID string        // CLI session ID for --resume
	Status    SessionStatus // idle, running, error
	Messages  []Message
	Error     string // last error message, if status == error
	WorkDir   string // worktree filesystem path

	// streamingIdx tracks the message index currently being streamed to.
	// This ensures deltas from an ongoing turn go to the correct assistant
	// message even if user messages are interleaved mid-turn. -1 means
	// no active streaming target.
	streamingIdx int
}

// snapshot returns a deep copy of the session for safe concurrent reads.
func (s *Session) snapshot() Session {
	msgs := make([]Message, len(s.Messages))
	copy(msgs, s.Messages)
	return Session{
		ID:        s.ID,
		AgentType: s.AgentType,
		SessionID: s.SessionID,
		Status:    s.Status,
		Messages:  msgs,
		Error:     s.Error,
		WorkDir:   s.WorkDir,
	}
}
