package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/glamour"
	gansi "github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/x/ansi"
)

func TestMarkdownStylesHaveNoDocumentMargin(t *testing.T) {
	tests := []struct {
		name  string
		style gansi.StyleConfig
	}{
		{"dark", DarkMarkdownStyle()},
		{"auto", AutoMarkdownStyle()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := glamour.NewTermRenderer(glamour.WithStyles(tt.style), glamour.WithWordWrap(20))
			if err != nil {
				t.Fatal(err)
			}
			out, err := r.Render("hello world")
			if err != nil {
				t.Fatal(err)
			}
			var text string
			for _, line := range strings.Split(ansi.Strip(out), "\n") {
				if strings.TrimSpace(line) != "" {
					text = line
					break
				}
			}
			if !strings.HasPrefix(text, "hello") {
				t.Errorf("first text line = %q, want it to start at column 0", text)
			}
		})
	}
}

func TestMarkdownStylesLeaveGlamourDefaultsUntouched(t *testing.T) {
	_ = DarkMarkdownStyle()
	_ = AutoMarkdownStyle()
	if m := styles.DarkStyleConfig.Document.Margin; m == nil || *m != 2 {
		t.Errorf("styles.DarkStyleConfig.Document.Margin changed to %v, want 2", m)
	}
}
