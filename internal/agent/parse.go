package agent

import (
	"encoding/json"
	"strings"
)

// streamEvent represents a single JSON line from Claude Code's stream-json output.
// The format varies by event type — we use a permissive struct and inspect fields.
type streamEvent struct {
	Type    string `json:"type"`
	Subtype string `json:"subtype,omitempty"`

	// For "assistant" events — contains the full message
	Message *messagePayload `json:"message,omitempty"`

	// For "stream_event" — wraps Anthropic API events (content_block_delta, etc.)
	Event *innerEvent `json:"event,omitempty"`

	// For top-level "content_block_delta" events (legacy/direct)
	Delta *deltaPayload `json:"delta,omitempty"`

	// For top-level "content_block_start" events (legacy/direct)
	ContentBlock *contentBlockPayload `json:"content_block,omitempty"`

	// For "result" events
	SessionID          string              `json:"session_id,omitempty"`
	Result             string              `json:"result,omitempty"`
	IsError            bool                `json:"is_error,omitempty"`
	CostUSD float64 `json:"total_cost_usd,omitempty"`

	// For error events
	Error *errorPayload `json:"error,omitempty"`

	// For "system" status events (e.g. compacting)
	Status string `json:"status,omitempty"`

	// For "system" task_progress events (subagent activity)
	TaskID       string `json:"task_id,omitempty"`
	Description  string `json:"description,omitempty"`
	LastToolName string `json:"last_tool_name,omitempty"`
}

// innerEvent is the Anthropic API event nested inside a "stream_event" wrapper.
type innerEvent struct {
	Type         string               `json:"type"`
	Delta        *deltaPayload        `json:"delta,omitempty"`
	ContentBlock *contentBlockPayload `json:"content_block,omitempty"`
}

type messagePayload struct {
	Role    string `json:"role,omitempty"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text,omitempty"`
	} `json:"content,omitempty"`
}

type deltaPayload struct {
	Type        string `json:"type"`
	Text        string `json:"text,omitempty"`
	PartialJSON string `json:"partial_json,omitempty"`
	Thinking    string `json:"thinking,omitempty"` // for thinking_delta from subagents
}

type contentBlockPayload struct {
	Type string `json:"type"`
	Name string `json:"name,omitempty"`
	Text string `json:"text,omitempty"`
}

type errorPayload struct {
	Message string `json:"message"`
}

// parsedEvent is the normalized result of parsing a stream-json line.
type parsedEvent struct {
	Type      parsedEventType
	Text      string // for TextDelta / AssistantMessage / Result / TaskProgress (description)
	ToolName  string // for ToolUse / TaskProgress (last_tool_name)
	TaskID    string // for TaskProgress — unique subagent task identifier
	SessionID string // for Result / AssistantMessage
	Error     string // for Error
}

type parsedEventType int

const (
	eventUnknown parsedEventType = iota
	eventIgnored  // recognized but not actionable (e.g. message_stop, content_block_stop)
	eventTextDelta
	eventNewTextBlock // content_block_start with type=text (signals paragraph break needed)
	eventAssistantMessage
	eventToolUse
	eventToolInputDelta // input_json_delta for tool_use — accumulates tool input JSON
	eventResult
	eventError
	eventSystemStatus  // system status change (e.g. "compacting")
	eventTaskProgress  // system task_progress — subagent activity update
	eventToolResult    // "user" event — tool result returned (signals subagent completion)
)

// parseStreamLine parses a single JSON line from Claude Code's stream-json output.
func parseStreamLine(line []byte) parsedEvent {
	var ev streamEvent
	if err := json.Unmarshal(line, &ev); err != nil {
		return parsedEvent{Type: eventUnknown}
	}

	switch ev.Type {
	case "stream_event":
		// Unwrap the nested Anthropic API event, preserving the session_id
		// from the wrapper so callers can distinguish subagent events.
		pe := parseInnerEvent(ev.Event)
		if ev.SessionID != "" {
			pe.SessionID = ev.SessionID
		}
		return pe

	case "assistant":
		// Full assistant message — extract text from content blocks
		if ev.Message != nil {
			var text string
			for _, c := range ev.Message.Content {
				if c.Type == "text" {
					text += c.Text
				}
			}
			return parsedEvent{
				Type:      eventAssistantMessage,
				Text:      text,
				SessionID: ev.SessionID,
			}
		}

	case "content_block_delta":
		// Direct (non-wrapped) delta — kept for compatibility
		if ev.Delta != nil {
			if ev.Delta.Type == "text_delta" {
				return parsedEvent{Type: eventTextDelta, Text: ev.Delta.Text}
			}
			if ev.Delta.Type == "input_json_delta" {
				return parsedEvent{Type: eventToolInputDelta, Text: ev.Delta.PartialJSON}
			}
		}

	case "content_block_start":
		if ev.ContentBlock != nil {
			if ev.ContentBlock.Type == "text" {
				return parsedEvent{Type: eventNewTextBlock, Text: ev.ContentBlock.Text}
			}
			if ev.ContentBlock.Type == "tool_use" && ev.ContentBlock.Name != "" {
				return parsedEvent{Type: eventToolUse, ToolName: ev.ContentBlock.Name}
			}
		}

	case "result":
		if ev.IsError {
			return parsedEvent{Type: eventError, Error: ev.Result, SessionID: ev.SessionID}
		}
		return parsedEvent{Type: eventResult, Text: ev.Result, SessionID: ev.SessionID}

	case "error":
		msg := "unknown error"
		if ev.Error != nil {
			msg = ev.Error.Message
		}
		return parsedEvent{Type: eventError, Error: msg}

	case "system":
		if ev.Subtype == "status" && ev.Status != "" {
			return parsedEvent{Type: eventSystemStatus, Text: ev.Status}
		}
		if ev.Subtype == "task_progress" {
			return parsedEvent{
				Type:     eventTaskProgress,
				Text:     ev.Description,
				ToolName: ev.LastToolName,
				TaskID:   ev.TaskID,
			}
		}

	case "user":
		// Tool result / user message events — no UI action needed, but used
		// as a boundary signal to detect when subagent execution has completed.
		return parsedEvent{Type: eventToolResult}
	}

	return parsedEvent{Type: eventUnknown}
}

// parseInnerEvent extracts text deltas from the Anthropic API event nested in stream_event.
func parseInnerEvent(inner *innerEvent) parsedEvent {
	if inner == nil {
		return parsedEvent{Type: eventUnknown}
	}

	switch inner.Type {
	case "content_block_delta":
		if inner.Delta != nil {
			if inner.Delta.Type == "text_delta" {
				return parsedEvent{Type: eventTextDelta, Text: inner.Delta.Text}
			}
			if inner.Delta.Type == "input_json_delta" {
				return parsedEvent{Type: eventToolInputDelta, Text: inner.Delta.PartialJSON}
			}
			if inner.Delta.Type == "thinking_delta" || inner.Delta.Type == "signature_delta" {
				// Thinking/signature deltas (from subagents) — no actionable content
				return parsedEvent{Type: eventIgnored}
			}
		}
	case "content_block_start":
		if inner.ContentBlock != nil {
			if inner.ContentBlock.Type == "text" {
				return parsedEvent{Type: eventNewTextBlock, Text: inner.ContentBlock.Text}
			}
			if inner.ContentBlock.Type == "tool_use" && inner.ContentBlock.Name != "" {
				return parsedEvent{Type: eventToolUse, ToolName: inner.ContentBlock.Name}
			}
		}
	case "content_block_stop", "message_start", "message_delta", "message_stop", "ping":
		// Benign lifecycle events — no actionable content
		return parsedEvent{Type: eventIgnored}
	}

	return parsedEvent{Type: eventUnknown}
}

// toolInputSummaryFields are the JSON fields to look for (in order) when
// extracting a human-readable summary from tool input.
var toolInputSummaryFields = []string{
	"description", // Agent, Bash
	"file_path",   // Read, Edit, Write
	"pattern",     // Grep, Glob
	"command",     // Bash (fallback)
	"query",       // WebSearch, ToolSearch
	"skill",       // Skill
	"prompt",      // Agent (fallback — usually long, truncated)
}

// extractToolSummary tries to extract a short summary from accumulated
// tool input JSON. If workDir is non-empty, it is stripped from file_path
// values so the display shows relative paths. Returns empty string if
// nothing useful is found.
func extractToolSummary(inputJSON, workDir string) string {
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(inputJSON), &obj); err != nil {
		return ""
	}
	for _, field := range toolInputSummaryFields {
		if v, ok := obj[field]; ok {
			if s, ok := v.(string); ok && s != "" {
				// Strip workDir prefix from file paths
				if field == "file_path" && workDir != "" {
					prefix := workDir + "/"
					s = strings.TrimPrefix(s, prefix)
				}
				// Truncate long values
				if len(s) > 80 {
					s = s[:77] + "..."
				}
				return s
			}
		}
	}
	return ""
}
