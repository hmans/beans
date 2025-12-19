package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/hmans/beans/internal/ui"
)

// pickerModalConfig holds configuration for rendering a picker modal
type pickerModalConfig struct {
	Title       string // e.g., "Select Status"
	BeanTitle   string // the bean's title
	BeanID      string // the bean's ID
	ListContent string // the rendered list
	Description string // optional description shown below list
	Width       int    // screen width
	WidthPct    int    // modal width percentage (default 50)
	MaxWidth    int    // max modal width (default 60)
}

// renderPickerModal renders a standard picker modal with consistent styling
func renderPickerModal(cfg pickerModalConfig) string {
	// Default values
	widthPct := cfg.WidthPct
	if widthPct == 0 {
		widthPct = 50
	}
	maxWidth := cfg.MaxWidth
	if maxWidth == 0 {
		maxWidth = 60
	}

	modalWidth := max(40, min(maxWidth, cfg.Width*widthPct/100))

	// Header with bean title (truncated if needed)
	titleWidth := modalWidth - 4
	beanTitle := cfg.BeanTitle
	if len(beanTitle) > titleWidth {
		beanTitle = beanTitle[:titleWidth-3] + "..."
	}
	header := lipgloss.NewStyle().Bold(true).Render(beanTitle)

	// Subtitle with bean ID
	subtitle := ui.Muted.Render(cfg.BeanID)

	// Help footer
	help := helpKeyStyle.Render("enter") + " " + helpStyle.Render("select") + "  " +
		helpKeyStyle.Render("/") + " " + helpStyle.Render("filter") + "  " +
		helpKeyStyle.Render("esc") + " " + helpStyle.Render("cancel")

	// Border style
	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorPrimary).
		Padding(0, 1).
		Width(modalWidth)

	// Assemble content
	var content string
	if cfg.Description != "" {
		content = header + "\n" + subtitle + "\n\n" + cfg.ListContent + "\n\n" + cfg.Description + "\n\n" + help
	} else {
		content = header + "\n" + subtitle + "\n\n" + cfg.ListContent + "\n\n" + help
	}

	return border.Render(content)
}

// overlayModal places a modal on top of a background view
func overlayModal(bgView, modal string, width, height int) string {
	// Split background into lines
	bgLines := strings.Split(bgView, "\n")

	// Pad or truncate background to fill the screen
	for len(bgLines) < height {
		bgLines = append(bgLines, "")
	}
	if len(bgLines) > height {
		bgLines = bgLines[:height]
	}

	// Dim the background
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#555"))
	for i, line := range bgLines {
		bgLines[i] = dimStyle.Render(stripAnsi(line))
	}

	// Split modal into lines
	modalLines := strings.Split(modal, "\n")
	modalHeight := len(modalLines)
	modalWidth := lipgloss.Width(modal)

	// Calculate center position
	startY := (height - modalHeight) / 2
	startX := (width - modalWidth) / 2
	if startY < 0 {
		startY = 0
	}
	if startX < 0 {
		startX = 0
	}

	// Overlay modal onto background
	for i, modalLine := range modalLines {
		bgY := startY + i
		if bgY >= 0 && bgY < len(bgLines) {
			bgLines[bgY] = overlayLine(bgLines[bgY], modalLine, startX, width)
		}
	}

	return strings.Join(bgLines, "\n")
}

// overlayLine places a modal line on top of a background line at position x
func overlayLine(bgLine, modalLine string, startX, maxWidth int) string {
	bgRunes := []rune(stripAnsi(bgLine))
	for len(bgRunes) < maxWidth {
		bgRunes = append(bgRunes, ' ')
	}

	prefix := string(bgRunes[:startX])
	modalWidth := lipgloss.Width(modalLine)
	suffixStart := startX + modalWidth
	suffix := ""
	if suffixStart < len(bgRunes) {
		suffix = string(bgRunes[suffixStart:])
	}

	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#555"))
	return dimStyle.Render(prefix) + modalLine + dimStyle.Render(suffix)
}

// stripAnsi removes ANSI escape codes from a string
func stripAnsi(s string) string {
	result := strings.Builder{}
	inEscape := false
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEscape = false
			}
			continue
		}
		result.WriteRune(r)
	}
	return result.String()
}
