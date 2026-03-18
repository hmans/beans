package agent

import "testing"

func TestParseQuickReplies(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "clean lines",
			input: "Yes, implement this\nShow me the code first\nWhat about error handling?\nLet's skip this for now",
			want:  []string{"Yes, implement this", "Show me the code first", "What about error handling?", "Let's skip this for now"},
		},
		{
			name:  "with bullets",
			input: "- Yes, implement this\n- Show me the code\n- Skip this",
			want:  []string{"Yes, implement this", "Show me the code", "Skip this"},
		},
		{
			name:  "with numbering",
			input: "1. Yes, implement this\n2. Show me the code\n3. Skip this",
			want:  []string{"Yes, implement this", "Show me the code", "Skip this"},
		},
		{
			name:  "with quotes",
			input: "\"Yes, implement this\"\n\"Show me the code\"",
			want:  []string{"Yes, implement this", "Show me the code"},
		},
		{
			name:  "empty lines filtered",
			input: "Yes, implement this\n\n\nShow me the code\n\n",
			want:  []string{"Yes, implement this", "Show me the code"},
		},
		{
			name:  "caps at 4",
			input: "One\nTwo\nThree\nFour\nFive\nSix",
			want:  []string{"One", "Two", "Three", "Four"},
		},
		{
			name:  "empty input",
			input: "",
			want:  nil,
		},
		{
			name:  "whitespace only",
			input: "  \n  \n  ",
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseQuickReplies(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("parseQuickReplies() returned %d replies, want %d: %v", len(got), len(tt.want), got)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("reply[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestSendMessageClearsQuickReplies(t *testing.T) {
	m := &Manager{
		sessions:    make(map[string]*Session),
		processes:   make(map[string]*runningProcess),
		subscribers: make(map[string][]chan struct{}),
	}

	// Pre-create a session with quick replies
	m.sessions["wt-test"] = &Session{
		ID:           "wt-test",
		AgentType:    "claude",
		Status:       StatusIdle,
		Messages:     []Message{{Role: RoleUser, Content: "First message"}},
		QuickReplies: []string{"Yes", "No", "Maybe"},
		streamingIdx: -1,
	}

	_ = m.SendMessage("wt-test", "/tmp", "Second message", nil)

	s := m.GetSession("wt-test")
	if len(s.QuickReplies) != 0 {
		t.Errorf("QuickReplies should be cleared after SendMessage, got %v", s.QuickReplies)
	}
}

func TestSnapshotCopiesQuickReplies(t *testing.T) {
	s := &Session{
		ID:           "test",
		AgentType:    "claude",
		Status:       StatusIdle,
		QuickReplies: []string{"Yes", "No"},
		streamingIdx: -1,
	}

	snap := s.snapshot()

	// Modify original
	s.QuickReplies[0] = "Modified"

	if snap.QuickReplies[0] != "Yes" {
		t.Error("snapshot QuickReplies should be independent of original")
	}
}
