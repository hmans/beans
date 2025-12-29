package ui

import "testing"

func TestRenderBeanRow_NarrowWidth(t *testing.T) {
	// Test that RenderBeanRow doesn't panic with very small MaxTitleWidth values
	// This was a bug where MaxTitleWidth < 4 caused a slice bounds panic

	tests := []struct {
		name          string
		maxTitleWidth int
		title         string
	}{
		{"zero width", 0, "Test Title"},
		{"width 1", 1, "Test Title"},
		{"width 2", 2, "Test Title"},
		{"width 3", 3, "Test Title"},
		{"width 4", 4, "Test Title"},
		{"width 5", 5, "Test Title"},
		{"short title fits", 10, "Hi"},
		{"exact fit", 10, "0123456789"},
		{"needs truncation", 10, "This is a longer title"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("RenderBeanRow panicked with MaxTitleWidth=%d: %v", tt.maxTitleWidth, r)
				}
			}()

			cfg := BeanRowConfig{
				MaxTitleWidth: tt.maxTitleWidth,
				StatusColor:   "green",
				TypeColor:     "blue",
			}

			result := RenderBeanRow("abc123", "todo", "task", tt.title, cfg)
			if result == "" {
				t.Error("expected non-empty result")
			}
		})
	}
}

func TestRenderBeanRow_NarrowWidthWithPriority(t *testing.T) {
	// Priority symbol takes 2 extra chars, which reduces available title width
	// This tests that the adjustment doesn't cause negative slice bounds

	tests := []struct {
		name          string
		maxTitleWidth int
		priority      string
	}{
		{"width 1 with priority", 1, "high"},
		{"width 2 with priority", 2, "high"},
		{"width 3 with priority", 3, "critical"},
		{"width 4 with priority", 4, "high"},
		{"width 5 with priority", 5, "low"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("RenderBeanRow panicked with MaxTitleWidth=%d and priority=%s: %v",
						tt.maxTitleWidth, tt.priority, r)
				}
			}()

			cfg := BeanRowConfig{
				MaxTitleWidth: tt.maxTitleWidth,
				Priority:      tt.priority,
				PriorityColor: "red",
				StatusColor:   "green",
				TypeColor:     "blue",
			}

			result := RenderBeanRow("abc123", "todo", "task", "Long title that needs truncation", cfg)
			if result == "" {
				t.Error("expected non-empty result")
			}
		})
	}
}

func TestShortType(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"milestone", "M"},
		{"epic", "E"},
		{"bug", "B"},
		{"feature", "F"},
		{"task", "T"},
		{"unknown", "?"},
		{"", "?"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ShortType(tt.input)
			if result != tt.expected {
				t.Errorf("ShortType(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestShortStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"draft", "D"},
		{"todo", "T"},
		{"in-progress", "I"},
		{"completed", "C"},
		{"scrapped", "S"},
		{"unknown", "?"},
		{"", "?"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ShortStatus(tt.input)
			if result != tt.expected {
				t.Errorf("ShortStatus(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
