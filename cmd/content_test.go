package cmd

import (
	"testing"
)

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
