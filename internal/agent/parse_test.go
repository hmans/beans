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
			name:  "stream_event with content_block_delta",
			input: `{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}},"session_id":"abc"}`,
			want:  parsedEvent{Type: eventTextDelta, Text: "Hello"},
		},
		{
			name:  "stream_event with content_block_start text",
			input: `{"type":"stream_event","event":{"type":"content_block_start","index":0,"content_block":{"type":"text","text":"Hi"}},"session_id":"abc"}`,
			want:  parsedEvent{Type: eventNewTextBlock, Text: "Hi"},
		},
		{
			name:  "stream_event with content_block_start empty text",
			input: `{"type":"stream_event","event":{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}},"session_id":"abc"}`,
			want:  parsedEvent{Type: eventNewTextBlock},
		},
		{
			name:  "stream_event with message_start (ignored)",
			input: `{"type":"stream_event","event":{"type":"message_start","message":{"model":"claude"}},"session_id":"abc"}`,
			want:  parsedEvent{Type: eventUnknown},
		},
		{
			name:  "stream_event with tool use",
			input: `{"type":"stream_event","event":{"type":"content_block_start","index":1,"content_block":{"type":"tool_use","name":"Read"}},"session_id":"abc"}`,
			want:  parsedEvent{Type: eventToolUse, ToolName: "Read"},
		},
		{
			name:  "direct text delta (legacy)",
			input: `{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}`,
			want:  parsedEvent{Type: eventTextDelta, Text: "Hello"},
		},
		{
			name:  "direct content block start with text (legacy)",
			input: `{"type":"content_block_start","index":0,"content_block":{"type":"text","text":"Hi"}}`,
			want:  parsedEvent{Type: eventNewTextBlock, Text: "Hi"},
		},
		{
			name:  "direct content block start tool use",
			input: `{"type":"content_block_start","index":1,"content_block":{"type":"tool_use","name":"Read"}}`,
			want:  parsedEvent{Type: eventToolUse, ToolName: "Read"},
		},
		{
			name:  "stream_event with input_json_delta",
			input: `{"type":"stream_event","event":{"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"{\"file_path\":"}},"session_id":"abc"}`,
			want:  parsedEvent{Type: eventToolInputDelta, Text: `{"file_path":`},
		},
		{
			name:  "direct input_json_delta (legacy)",
			input: `{"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"\"foo.go\"}"}}`,
			want:  parsedEvent{Type: eventToolInputDelta, Text: `"foo.go"}`},
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
			if got.ToolName != tt.want.ToolName {
				t.Errorf("ToolName = %q, want %q", got.ToolName, tt.want.ToolName)
			}
			if got.Error != tt.want.Error {
				t.Errorf("Error = %q, want %q", got.Error, tt.want.Error)
			}
		})
	}
}

func TestExtractToolSummary(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "agent with description",
			input: `{"description":"search codebase","prompt":"find all TODO comments"}`,
			want:  "search codebase",
		},
		{
			name:  "read with file_path",
			input: `{"file_path":"/src/main.go"}`,
			want:  "/src/main.go",
		},
		{
			name:  "bash with description",
			input: `{"command":"ls -la","description":"list files"}`,
			want:  "list files",
		},
		{
			name:  "grep with pattern",
			input: `{"pattern":"TODO","path":"/src"}`,
			want:  "TODO",
		},
		{
			name:  "long value truncated",
			input: `{"file_path":"/very/long/path/that/goes/on/and/on/and/on/and/on/and/on/and/on/forever/and/ever/amen.go"}`,
			want:  "/very/long/path/that/goes/on/and/on/and/on/and/on/and/on/and/on/forever/and/e...",
		},
		{
			name:  "incomplete JSON",
			input: `{"file_path":`,
			want:  "",
		},
		{
			name:  "empty object",
			input: `{}`,
			want:  "",
		},
		{
			name:  "no known fields",
			input: `{"foo":"bar"}`,
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractToolSummary(tt.input)
			if got != tt.want {
				t.Errorf("extractToolSummary() = %q, want %q", got, tt.want)
			}
		})
	}
}
