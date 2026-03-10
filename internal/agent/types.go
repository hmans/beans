// Package agent manages AI coding agent sessions within worktrees.
package agent

// MessageRole identifies who sent a message.
type MessageRole string

const (
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
	RoleTool      MessageRole = "tool"
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

// ToolInvocation records a tool call with its name and input summary.
type ToolInvocation struct {
	Tool  string
	Input string // summary of tool input (e.g. file path, command)
}

// InteractionType identifies the kind of blocking interaction the agent is requesting.
type InteractionType string

const (
	InteractionExitPlan   InteractionType = "exit_plan"
	InteractionEnterPlan  InteractionType = "enter_plan"
	InteractionAskUser    InteractionType = "ask_user"
	InteractionPermission InteractionType = "permission_request"
)

// PermissionDenial represents a tool call that Claude Code auto-denied.
// Parsed from the permission_denials field in result events.
type PermissionDenial struct {
	ToolName  string         `json:"tool_name"`
	ToolInput map[string]any `json:"tool_input"`
}

// PendingInteraction represents a blocking tool call that requires user input.
// For plan/mode interactions, the process has been killed and will resume with --resume.
// For permission denials, the turn completed and the user must approve tools before retrying.
type PendingInteraction struct {
	Type              InteractionType
	PlanContent       string             // plan file content (for exit_plan only)
	PermissionDenials []PermissionDenial // denied tools (for permission_request only)
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
	PlanMode     bool   // when true, agent uses --permission-mode plan (read-only)
	YoloMode     bool   // when true, agent uses --dangerously-skip-permissions (fully autonomous)
	SystemStatus string // transient system status (e.g. "compacting"), empty when idle

	// ToolInvocations tracks structured tool calls in the current turn.
	// Reset on each new user message. Used to find plan files, etc.
	ToolInvocations []ToolInvocation

	// AllowedTools accumulates tool patterns approved by the user (e.g. "Read", "Bash(ls)").
	// Passed as --allowedTools on next process spawn.
	AllowedTools []string

	// PendingInteraction is set when the agent calls a blocking tool
	// (ExitPlanMode, EnterPlanMode) and is waiting for user approval.
	PendingInteraction *PendingInteraction

	// streamingIdx tracks the message index currently being streamed to.
	// This ensures deltas from an ongoing turn go to the correct assistant
	// message even if user messages are interleaved mid-turn. -1 means
	// no active streaming target.
	streamingIdx int
}

// snapshot returns a deep copy of the session for safe concurrent reads.
func (s *Session) snapshot() Session {
	snap := Session{
		ID:                 s.ID,
		AgentType:          s.AgentType,
		SessionID:          s.SessionID,
		Status:             s.Status,
		Messages:           make([]Message, len(s.Messages)),
		Error:              s.Error,
		WorkDir:            s.WorkDir,
		PlanMode:           s.PlanMode,
		YoloMode:           s.YoloMode,
		SystemStatus:       s.SystemStatus,
		PendingInteraction: s.PendingInteraction,
	}
	copy(snap.Messages, s.Messages)
	return snap
}
