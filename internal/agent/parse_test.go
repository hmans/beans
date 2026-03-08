package agent

import "testing"

func TestParseStreamLine(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  parsedEvent
	}{
		{
			name:  "assistant message with text",
			input: `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Hi"}]},"session_id":"abc-123"}`,
			want:  parsedEvent{Type: eventAssistantMessage, Text: "Hi", SessionID: "abc-123"},
		},
		{
			name:  "assistant message with multiple text blocks",
			input: `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Hello "},{"type":"text","text":"world"}]},"session_id":"abc-123"}`,
			want:  parsedEvent{Type: eventAssistantMessage, Text: "Hello world", SessionID: "abc-123"},
		},
		{
			name:  "text delta",
			input: `{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}`,
			want:  parsedEvent{Type: eventTextDelta, Text: "Hello"},
		},
		{
			name:  "content block start with text",
			input: `{"type":"content_block_start","index":0,"content_block":{"type":"text","text":"Hi"}}`,
			want:  parsedEvent{Type: eventTextDelta, Text: "Hi"},
		},
		{
			name:  "content block start tool use (ignored)",
			input: `{"type":"content_block_start","index":1,"content_block":{"type":"tool_use","name":"Read"}}`,
			want:  parsedEvent{Type: eventUnknown},
		},
		{
			name:  "result success with session id",
			input: `{"type":"result","subtype":"success","is_error":false,"session_id":"def-456","result":"Hi","total_cost_usd":0.05}`,
			want:  parsedEvent{Type: eventResult, SessionID: "def-456", Text: "Hi"},
		},
		{
			name:  "result error",
			input: `{"type":"result","subtype":"error","is_error":true,"session_id":"def-456","result":"something broke"}`,
			want:  parsedEvent{Type: eventError, SessionID: "def-456", Error: "something broke"},
		},
		{
			name:  "error event",
			input: `{"type":"error","error":{"message":"rate limited"}}`,
			want:  parsedEvent{Type: eventError, Error: "rate limited"},
		},
		{
			name:  "system event (ignored)",
			input: `{"type":"system","subtype":"init","session_id":"abc"}`,
			want:  parsedEvent{Type: eventUnknown},
		},
		{
			name:  "unknown event type",
			input: `{"type":"ping"}`,
			want:  parsedEvent{Type: eventUnknown},
		},
		{
			name:  "invalid json",
			input: `not json at all`,
			want:  parsedEvent{Type: eventUnknown},
		},
		{
			name:  "empty line",
			input: ``,
			want:  parsedEvent{Type: eventUnknown},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseStreamLine([]byte(tt.input))
			if got.Type != tt.want.Type {
				t.Errorf("Type = %v, want %v", got.Type, tt.want.Type)
			}
			if got.Text != tt.want.Text {
				t.Errorf("Text = %q, want %q", got.Text, tt.want.Text)
			}
			if got.SessionID != tt.want.SessionID {
				t.Errorf("SessionID = %q, want %q", got.SessionID, tt.want.SessionID)
			}
			if got.Error != tt.want.Error {
				t.Errorf("Error = %q, want %q", got.Error, tt.want.Error)
			}
		})
	}
}
