package agent

import "encoding/json"

// streamEvent represents a single JSON line from Claude Code's stream-json output.
// The format varies by event type — we use a permissive struct and inspect fields.
type streamEvent struct {
	Type    string `json:"type"`
	Subtype string `json:"subtype,omitempty"`

	// For "assistant" events — contains the full message
	Message *messagePayload `json:"message,omitempty"`

	// For "content_block_delta" events (streaming with --include-partial-messages)
	Delta *deltaPayload `json:"delta,omitempty"`

	// For "content_block_start" events
	ContentBlock *contentBlockPayload `json:"content_block,omitempty"`

	// For "result" events
	SessionID string  `json:"session_id,omitempty"`
	Result    string  `json:"result,omitempty"`
	IsError   bool    `json:"is_error,omitempty"`
	CostUSD   float64 `json:"total_cost_usd,omitempty"`

	// For error events
	Error *errorPayload `json:"error,omitempty"`
}

type messagePayload struct {
	Role    string `json:"role,omitempty"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text,omitempty"`
	} `json:"content,omitempty"`
}

type deltaPayload struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
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
	Text      string // for TextDelta / AssistantMessage / Result
	SessionID string // for Result / AssistantMessage
	Error     string // for Error
}

type parsedEventType int

const (
	eventUnknown parsedEventType = iota
	eventTextDelta
	eventAssistantMessage
	eventResult
	eventError
)

// parseStreamLine parses a single JSON line from Claude Code's stream-json output.
func parseStreamLine(line []byte) parsedEvent {
	var ev streamEvent
	if err := json.Unmarshal(line, &ev); err != nil {
		return parsedEvent{Type: eventUnknown}
	}

	switch ev.Type {
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
		// Streaming text delta (with --include-partial-messages)
		if ev.Delta != nil && ev.Delta.Type == "text_delta" {
			return parsedEvent{Type: eventTextDelta, Text: ev.Delta.Text}
		}

	case "content_block_start":
		if ev.ContentBlock != nil && ev.ContentBlock.Type == "text" && ev.ContentBlock.Text != "" {
			return parsedEvent{Type: eventTextDelta, Text: ev.ContentBlock.Text}
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
	}

	return parsedEvent{Type: eventUnknown}
}
