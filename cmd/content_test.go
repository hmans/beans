package cmd

import (
	"testing"

	"github.com/hmans/beans/internal/bean"
)

func TestApplyTags(t *testing.T) {
	tests := []struct {
		name     string
		initial  []string
		toAdd    []string
		wantTags []string
		wantErr  bool
	}{
		{
			name:     "add single tag",
			initial:  nil,
			toAdd:    []string{"bug"},
			wantTags: []string{"bug"},
		},
		{
			name:     "add multiple tags",
			initial:  nil,
			toAdd:    []string{"bug", "urgent"},
			wantTags: []string{"bug", "urgent"},
		},
		{
			name:     "add to existing tags",
			initial:  []string{"existing"},
			toAdd:    []string{"new"},
			wantTags: []string{"existing", "new"},
		},
		{
			name:     "empty tags list",
			initial:  []string{"existing"},
			toAdd:    []string{},
			wantTags: []string{"existing"},
		},
		{
			name:    "invalid tag with spaces",
			initial: nil,
			toAdd:   []string{"invalid tag"},
			wantErr: true,
		},
		{
			name:     "uppercase tag gets normalized",
			initial:  nil,
			toAdd:    []string{"InvalidTag"},
			wantTags: []string{"invalidtag"}, // normalized to lowercase
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &bean.Bean{Tags: tt.initial}
			err := applyTags(b, tt.toAdd)

			if tt.wantErr {
				if err == nil {
					t.Errorf("applyTags() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("applyTags() unexpected error: %v", err)
				return
			}

			if len(b.Tags) != len(tt.wantTags) {
				t.Errorf("applyTags() tags count = %d, want %d", len(b.Tags), len(tt.wantTags))
				return
			}

			for i, want := range tt.wantTags {
				if b.Tags[i] != want {
					t.Errorf("applyTags() tags[%d] = %q, want %q", i, b.Tags[i], want)
				}
			}
		})
	}
}

func TestFormatCycle(t *testing.T) {
	tests := []struct {
		path []string
		want string
	}{
		{[]string{"a", "b", "c", "a"}, "a → b → c → a"},
		{[]string{"x", "y"}, "x → y"},
		{[]string{"single"}, "single"},
		{[]string{}, ""},
	}

	for _, tt := range tests {
		got := formatCycle(tt.path)
		if got != tt.want {
			t.Errorf("formatCycle(%v) = %q, want %q", tt.path, got, tt.want)
		}
	}
}
