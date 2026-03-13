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
			want:  parsedEvent{Type: eventTextDelta, Text: "Hello", SessionID: "abc"},
		},
		{
			name:  "stream_event with content_block_start text",
			input: `{"type":"stream_event","event":{"type":"content_block_start","index":0,"content_block":{"type":"text","text":"Hi"}},"session_id":"abc"}`,
			want:  parsedEvent{Type: eventNewTextBlock, Text: "Hi", SessionID: "abc"},
		},
		{
			name:  "stream_event with content_block_start empty text",
			input: `{"type":"stream_event","event":{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}},"session_id":"abc"}`,
			want:  parsedEvent{Type: eventNewTextBlock, SessionID: "abc"},
		},
		{
			name:  "stream_event with message_start (ignored)",
			input: `{"type":"stream_event","event":{"type":"message_start","message":{"model":"claude"}},"session_id":"abc"}`,
			want:  parsedEvent{Type: eventIgnored, SessionID: "abc"},
		},
		{
			name:  "stream_event with message_stop (ignored)",
			input: `{"type":"stream_event","event":{"type":"message_stop"},"session_id":"abc"}`,
			want:  parsedEvent{Type: eventIgnored, SessionID: "abc"},
		},
		{
			name:  "stream_event with message_delta (ignored)",
			input: `{"type":"stream_event","event":{"type":"message_delta","delta":{"stop_reason":"end_turn"}},"session_id":"abc"}`,
			want:  parsedEvent{Type: eventIgnored, SessionID: "abc"},
		},
		{
			name:  "stream_event with content_block_stop (ignored)",
			input: `{"type":"stream_event","event":{"type":"content_block_stop","index":0},"session_id":"abc"}`,
			want:  parsedEvent{Type: eventIgnored, SessionID: "abc"},
		},
		{
			name:  "user event (tool result)",
			input: `{"type":"user","message":{"role":"user","content":[{"tool_use_id":"toolu_123","type":"tool_result","content":"ok"}]}}`,
			want:  parsedEvent{Type: eventToolResult},
		},
		{
			name:  "stream_event with thinking_delta (ignored, from subagent)",
			input: `{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"Let me think..."}},"session_id":"sub-123"}`,
			want:  parsedEvent{Type: eventIgnored, SessionID: "sub-123"},
		},
		{
			name:  "stream_event with signature_delta (ignored)",
			input: `{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"signature_delta"}},"session_id":"sub-123"}`,
			want:  parsedEvent{Type: eventIgnored, SessionID: "sub-123"},
		},
		{
			name:  "stream_event with tool use",
			input: `{"type":"stream_event","event":{"type":"content_block_start","index":1,"content_block":{"type":"tool_use","name":"Read"}},"session_id":"abc"}`,
			want:  parsedEvent{Type: eventToolUse, ToolName: "Read", SessionID: "abc"},
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
			want:  parsedEvent{Type: eventToolInputDelta, Text: `{"file_path":`, SessionID: "abc"},
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
			name:  "system task_progress event",
			input: `{"type":"system","subtype":"task_progress","task_id":"abc123","tool_use_id":"toolu_01X","description":"Reading main.go","last_tool_name":"Read","session_id":"abc"}`,
			want:  parsedEvent{Type: eventTaskProgress, Text: "Reading main.go", ToolName: "Read", TaskID: "abc123"},
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
		name    string
		input   string
		workDir string
		want    string
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
			name:    "file_path with workDir stripped",
			input:   `{"file_path":"/home/user/project/src/main.go"}`,
			workDir: "/home/user/project",
			want:    "src/main.go",
		},
		{
			name:    "file_path outside workDir unchanged",
			input:   `{"file_path":"/other/path/main.go"}`,
			workDir: "/home/user/project",
			want:    "/other/path/main.go",
		},
		{
			name:    "empty workDir leaves path unchanged",
			input:   `{"file_path":"/src/main.go"}`,
			workDir: "",
			want:    "/src/main.go",
		},
		{
			name:    "non-file_path fields not stripped",
			input:   `{"pattern":"/home/user/project/TODO"}`,
			workDir: "/home/user/project",
			want:    "/home/user/project/TODO",
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
		{
			name:  "AskUserQuestion with questions array",
			input: `{"questions":[{"question":"Which approach should we use?","options":[{"label":"A"},{"label":"B"}]}]}`,
			want:  "Which approach should we use?",
		},
		{
			name:  "AskUserQuestion long question truncated",
			input: `{"questions":[{"question":"This is a very long question that goes on and on and should be truncated at eighty characters exactly right here"}]}`,
			want:  "This is a very long question that goes on and on and should be truncated at e...",
		},
		{
			name:  "AskUserQuestion empty questions array",
			input: `{"questions":[]}`,
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractToolSummary(tt.input, tt.workDir)
			if got != tt.want {
				t.Errorf("extractToolSummary() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseAskUserInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []AskUserQuestion
	}{
		{
			name: "single question with options",
			input: `{
				"questions": [{
					"header": "Approach",
					"question": "Which library should we use?",
					"multiSelect": false,
					"options": [
						{"label": "Option A", "description": "Fast but complex"},
						{"label": "Option B", "description": "Simple but slower"}
					]
				}]
			}`,
			want: []AskUserQuestion{
				{
					Header:      "Approach",
					Question:    "Which library should we use?",
					MultiSelect: false,
					Options: []AskUserOption{
						{Label: "Option A", Description: "Fast but complex"},
						{Label: "Option B", Description: "Simple but slower"},
					},
				},
			},
		},
		{
			name: "multiple questions with multiSelect",
			input: `{
				"questions": [
					{
						"header": "Auth",
						"question": "Which auth method?",
						"multiSelect": false,
						"options": [
							{"label": "JWT", "description": "Token-based"},
							{"label": "Session", "description": "Cookie-based"}
						]
					},
					{
						"header": "Features",
						"question": "Which features to enable?",
						"multiSelect": true,
						"options": [
							{"label": "Logging", "description": "Request logging"},
							{"label": "CORS", "description": "Cross-origin support"},
							{"label": "Rate limit", "description": "Throttle requests"}
						]
					}
				]
			}`,
			want: []AskUserQuestion{
				{
					Header: "Auth", Question: "Which auth method?",
					Options: []AskUserOption{
						{Label: "JWT", Description: "Token-based"},
						{Label: "Session", Description: "Cookie-based"},
					},
				},
				{
					Header: "Features", Question: "Which features to enable?", MultiSelect: true,
					Options: []AskUserOption{
						{Label: "Logging", Description: "Request logging"},
						{Label: "CORS", Description: "Cross-origin support"},
						{Label: "Rate limit", Description: "Throttle requests"},
					},
				},
			},
		},
		{
			name:  "malformed JSON",
			input: `not json`,
			want:  nil,
		},
		{
			name:  "empty questions array",
			input: `{"questions": []}`,
			want:  nil,
		},
		{
			name:  "missing questions field",
			input: `{"foo": "bar"}`,
			want:  nil,
		},
		{
			name:  "incomplete JSON from partial stream",
			input: `{"questions": [{"header": "Test"`,
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseAskUserInput(tt.input)
			if tt.want == nil {
				if got != nil {
					t.Errorf("expected nil, got %v", got)
				}
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %d questions, want %d", len(got), len(tt.want))
			}
			for i, q := range got {
				w := tt.want[i]
				if q.Header != w.Header {
					t.Errorf("q[%d].Header = %q, want %q", i, q.Header, w.Header)
				}
				if q.Question != w.Question {
					t.Errorf("q[%d].Question = %q, want %q", i, q.Question, w.Question)
				}
				if q.MultiSelect != w.MultiSelect {
					t.Errorf("q[%d].MultiSelect = %v, want %v", i, q.MultiSelect, w.MultiSelect)
				}
				if len(q.Options) != len(w.Options) {
					t.Fatalf("q[%d] got %d options, want %d", i, len(q.Options), len(w.Options))
				}
				for j, o := range q.Options {
					wo := w.Options[j]
					if o.Label != wo.Label {
						t.Errorf("q[%d].Options[%d].Label = %q, want %q", i, j, o.Label, wo.Label)
					}
					if o.Description != wo.Description {
						t.Errorf("q[%d].Options[%d].Description = %q, want %q", i, j, o.Description, wo.Description)
					}
				}
			}
		})
	}
}

func TestExtractFilePath(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"valid file_path", `{"file_path":"/src/main.go","content":"..."}`, "/src/main.go"},
		{"no file_path", `{"content":"..."}`, ""},
		{"incomplete JSON", `{"file_path":"/sr`, ""},
		{"empty object", `{}`, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractFilePath(tt.input)
			if got != tt.want {
				t.Errorf("extractFilePath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractFileContent(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"valid content", `{"file_path":"/x.go","content":"hello\nworld"}`, "hello\nworld"},
		{"no content", `{"file_path":"/x.go"}`, ""},
		{"incomplete JSON", `{"content":"hel`, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractFileContent(tt.input)
			if got != tt.want {
				t.Errorf("extractFileContent() = %q, want %q", got, tt.want)
			}
		})
	}
}
