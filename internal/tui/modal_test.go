package tui

import (
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestAnsiStrip(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no ANSI codes",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "simple color code",
			input:    "\x1b[31mred text\x1b[0m",
			expected: "red text",
		},
		{
			name:     "multiple ANSI codes",
			input:    "\x1b[1m\x1b[31mbold red\x1b[0m normal",
			expected: "bold red normal",
		},
		{
			name:     "ANSI codes with background",
			input:    "\x1b[41;37mwhite on red\x1b[0m",
			expected: "white on red",
		},
		{
			name:     "mixed content with ANSI",
			input:    "normal \x1b[32mgreen\x1b[0m back to normal",
			expected: "normal green back to normal",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only ANSI codes",
			input:    "\x1b[31m\x1b[0m",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ansi.Strip(tt.input)
			if result != tt.expected {
				t.Errorf("ansi.Strip(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
