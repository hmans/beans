package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

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

func TestRenderBeanRow_ClosedAncestorMark(t *testing.T) {
	const title = "A title far too long for the title column"
	tests := []struct {
		name       string
		status     string
		cfg        BeanRowConfig
		wantStatus string // status column content, mark included
	}{
		{"short names", "todo", BeanRowConfig{MaxTitleWidth: 20}, "↑T"},
		{"short names with priority", "todo", BeanRowConfig{MaxTitleWidth: 20, Priority: "high"}, "↑T"},
		{"full names", "todo", BeanRowConfig{MaxTitleWidth: 20, TypeColWidth: ColWidthTypeFull, StatusColWidth: ColWidthStatusFull}, "↑todo"},
		{"longest full name", "in-progress", BeanRowConfig{MaxTitleWidth: 20, TypeColWidth: ColWidthTypeFull, StatusColWidth: ColWidthStatusFull}, "↑in-progress"},
		{"cut names", "in-progress", BeanRowConfig{MaxTitleWidth: 20, TypeColWidth: ColWidthNameMin, StatusColWidth: ColWidthNameMin}, "↑in-pr"},
		{"with tags", "todo", BeanRowConfig{MaxTitleWidth: 20, ShowTags: true, TagsColWidth: 24, MaxTags: 1, Tags: []string{"idea"}}, "↑T"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plain := RenderBeanRow("abc123", tt.status, "task", title, tt.cfg)
			cfg := tt.cfg
			cfg.ImplicitStatus = "completed"
			marked := RenderBeanRow("abc123", tt.status, "task", title, cfg)

			if !strings.Contains(ansi.Strip(marked), " "+tt.wantStatus+" ") {
				t.Errorf("status column %q missing in %q", tt.wantStatus, ansi.Strip(marked))
			}
			if strings.Contains(ansi.Strip(marked), "completed") {
				t.Errorf("ancestor status must not be spelled out: %q", ansi.Strip(marked))
			}
			if got, want := strings.Replace(ansi.Strip(marked), "↑", " ", 1), ansi.Strip(plain); got != want {
				t.Errorf("mark must take the place of the space before the status\nplain:  %q\nmarked: %q",
					want, ansi.Strip(marked))
			}
		})
	}

	t.Run("dimmed context row has no mark", func(t *testing.T) {
		row := RenderBeanRow("abc123", "todo", "task", title, BeanRowConfig{MaxTitleWidth: 20, Dimmed: true, ImplicitStatus: "completed"})
		if strings.Contains(row, "↑") {
			t.Errorf("unexpected mark in dimmed row: %q", ansi.Strip(row))
		}
	})
}

func TestRenderBeanRow_BlockedMark(t *testing.T) {
	const title = "A title far too long for the title column"
	tests := []struct {
		name       string
		cfg        BeanRowConfig
		wantStatus string // status column content, mark included
	}{
		{"short names", BeanRowConfig{MaxTitleWidth: 20}, "⊘T"},
		{"full names", BeanRowConfig{MaxTitleWidth: 20, TypeColWidth: ColWidthTypeFull, StatusColWidth: ColWidthStatusFull}, "⊘todo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plain := RenderBeanRow("abc123", "todo", "task", title, tt.cfg)
			cfg := tt.cfg
			cfg.Blocked = true
			marked := ansi.Strip(RenderBeanRow("abc123", "todo", "task", title, cfg))

			if !strings.Contains(marked, " "+tt.wantStatus+" ") {
				t.Errorf("status column %q missing in %q", tt.wantStatus, marked)
			}
			if got, want := strings.Replace(marked, "⊘", " ", 1), ansi.Strip(plain); got != want {
				t.Errorf("mark must take the place of the space before the status\nplain:  %q\nmarked: %q", want, marked)
			}
		})
	}

	t.Run("closed ancestor mark wins", func(t *testing.T) {
		row := ansi.Strip(RenderBeanRow("abc123", "todo", "task", title, BeanRowConfig{MaxTitleWidth: 20, Blocked: true, ImplicitStatus: "completed"}))
		if !strings.Contains(row, "↑T") || strings.Contains(row, "⊘") {
			t.Errorf("want ↑ and no ⊘ in %q", row)
		}
	})

	t.Run("dimmed context row has no mark", func(t *testing.T) {
		row := RenderBeanRow("abc123", "todo", "task", title, BeanRowConfig{MaxTitleWidth: 20, Dimmed: true, Blocked: true})
		if strings.Contains(row, "⊘") {
			t.Errorf("unexpected mark in dimmed row: %q", ansi.Strip(row))
		}
	})
}

func TestCalculateResponsiveColumns(t *testing.T) {
	tests := []struct {
		width                int
		wantType, wantStatus int
		wantTags             int // 0 = no tags column
	}{
		{80, ColWidthType, ColWidthStatus, 0},
		{119, ColWidthType, ColWidthStatus, 0},
		{120, ColWidthNameMin, ColWidthNameMin, 0},
		{139, 8, 8, 0},
		{140, 9, 8, ColWidthTags},
		{160, ColWidthTypeFull, ColWidthStatusFull, ColWidthTags},
		{190, ColWidthTypeFull, ColWidthStatusFull, 47},
		{220, ColWidthTypeFull, ColWidthStatusFull, ColWidthTagsMax},
		{300, ColWidthTypeFull, ColWidthStatusFull, ColWidthTagsMax},
	}
	for _, tt := range tests {
		cols := CalculateResponsiveColumns(tt.width, true)
		if cols.Type != tt.wantType || cols.Status != tt.wantStatus {
			t.Errorf("width %d: type/status = %d/%d, want %d/%d", tt.width, cols.Type, cols.Status, tt.wantType, tt.wantStatus)
		}
		if cols.ShowTags != (tt.wantTags > 0) || cols.Tags != tt.wantTags {
			t.Errorf("width %d: tags = %d (shown %v), want %d", tt.width, cols.Tags, cols.ShowTags, tt.wantTags)
		}
	}

	if cols := CalculateResponsiveColumns(200, false); cols.ShowTags {
		t.Error("tags column shown without tags")
	}
}

func TestCalculateResponsiveColumns_TitleNeverShrinks(t *testing.T) {
	titleWidth := func(width int) int {
		cols := CalculateResponsiveColumns(width, true)
		return width - cols.ID - cols.Type - cols.Status - cols.Tags
	}
	for width := 60; width <= 260; width++ {
		if width == minWidthForNames || width == minWidthForTags {
			continue
		}
		if prev, cur := titleWidth(width-1), titleWidth(width); cur < prev {
			t.Errorf("title width shrinks from %d to %d at width %d", prev, cur, width)
		}
	}
}

func TestRenderBeanRow_TypeStatusWidth(t *testing.T) {
	tests := []struct {
		typeName, status string
		width            int
		want             string // type and status columns with their separators
	}{
		{"epic", "todo", ColWidthType, " E   T   "},
		{"epic", "todo", ColWidthNameMin, " epic  todo  "},
		{"milestone", "in-progress", ColWidthNameMin, " miles in-pr "},
		{"milestone", "in-progress", 8, " mileston in-progr "},
		{"milestone", "in-progress", ColWidthTypeFull, " milestone    in-progress  "},
	}
	for _, tt := range tests {
		row := ansi.Strip(RenderBeanRow("abc", tt.status, tt.typeName, "Title", BeanRowConfig{
			IDColWidth: 3, TypeColWidth: tt.width, StatusColWidth: tt.width,
		}))
		if want := "abc" + tt.want + "Title"; row != want {
			t.Errorf("width %d: row = %q, want %q", tt.width, row, want)
		}
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
