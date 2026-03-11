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

// ActiveAgent is a lightweight struct for reporting which beans have active agents.
type ActiveAgent struct {
	BeanID string
	Status SessionStatus
}

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

// SubagentActivity tracks the current state of a running subagent (Agent tool).
// This is transient — set while a subagent is active, cleared when it completes.
type SubagentActivity struct {
	TaskID      string // unique task ID from task_progress events
	Index       int    // sequential index (1-based) for display
	Description string // what the subagent is currently doing (from task_progress)
	CurrentTool string // tool currently being used by the subagent (empty when none)
}

// InteractionType identifies the kind of blocking interaction the agent is requesting.
type InteractionType string

const (
	InteractionExitPlan  InteractionType = "exit_plan"
	InteractionEnterPlan InteractionType = "enter_plan"
	InteractionAskUser   InteractionType = "ask_user"
)

// PendingInteraction represents a blocking tool call that requires user input.
// For plan/mode interactions, the process has been killed and will resume with --resume.
type PendingInteraction struct {
	Type        InteractionType
	PlanContent string // plan file content (for exit_plan only)
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
	PlanMode bool // when true, agent uses --permission-mode plan (read-only)
	ActMode  bool // when true, agent uses --dangerously-skip-permissions (fully autonomous)
	SystemStatus string // transient system status (e.g. "compacting"), empty when idle

	// ToolInvocations tracks structured tool calls in the current turn.
	// Reset on each new user message. Used to find plan files, etc.
	ToolInvocations []ToolInvocation

	// PendingInteraction is set when the agent calls a blocking tool
	// (ExitPlanMode, EnterPlanMode) and is waiting for user approval.
	PendingInteraction *PendingInteraction

	// SubagentActivities tracks what running subagents are doing.
	// Non-empty only while Agent tool calls are actively running.
	// Keyed by task_id from task_progress events.
	SubagentActivities []*SubagentActivity

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
		ActMode:           s.ActMode,
		SystemStatus:       s.SystemStatus,
		PendingInteraction: s.PendingInteraction,
	}
	copy(snap.Messages, s.Messages)
	if len(s.SubagentActivities) > 0 {
		snap.SubagentActivities = make([]*SubagentActivity, len(s.SubagentActivities))
		for i, sa := range s.SubagentActivities {
			copy := *sa
			snap.SubagentActivities[i] = &copy
		}
	}
	return snap
}
