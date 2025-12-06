package ui

import "github.com/charmbracelet/lipgloss"

// Color palette
var (
	ColorPrimary   = lipgloss.Color("#7C3AED") // Purple
	ColorSecondary = lipgloss.Color("#6B7280") // Gray
	ColorSuccess   = lipgloss.Color("#10B981") // Green
	ColorWarning   = lipgloss.Color("#F59E0B") // Amber
	ColorDanger    = lipgloss.Color("#EF4444") // Red
	ColorMuted     = lipgloss.Color("#9CA3AF") // Light gray
)

// Status badge styles (for inline use, like in show command)
var (
	StatusOpen = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#fff")).
			Background(ColorSuccess).
			Padding(0, 1).
			Bold(true)

	StatusClosed = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#fff")).
			Background(ColorSecondary).
			Padding(0, 1)

	StatusInProgress = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#fff")).
				Background(ColorWarning).
				Padding(0, 1).
				Bold(true)
)

// Status text styles (for table use, no background/padding)
var (
	StatusOpenText       = lipgloss.NewStyle().Foreground(ColorSuccess).Bold(true)
	StatusClosedText     = lipgloss.NewStyle().Foreground(ColorSecondary)
	StatusInProgressText = lipgloss.NewStyle().Foreground(ColorWarning).Bold(true)
)

// Text styles
var (
	Bold      = lipgloss.NewStyle().Bold(true)
	Muted     = lipgloss.NewStyle().Foreground(ColorMuted)
	Primary   = lipgloss.NewStyle().Foreground(ColorPrimary)
	Success   = lipgloss.NewStyle().Foreground(ColorSuccess)
	Warning   = lipgloss.NewStyle().Foreground(ColorWarning)
	Danger    = lipgloss.NewStyle().Foreground(ColorDanger)
	Secondary = lipgloss.NewStyle().Foreground(ColorSecondary)
)

// ID style - distinctive for bean IDs
var ID = lipgloss.NewStyle().
	Foreground(ColorPrimary).
	Bold(true)

// Title style
var Title = lipgloss.NewStyle().Bold(true)

// Path style - subdued
var Path = lipgloss.NewStyle().Foreground(ColorMuted)

// Header style for section headers
var Header = lipgloss.NewStyle().
	Foreground(ColorPrimary).
	Bold(true).
	MarginBottom(1)

// RenderStatus returns a styled status badge based on the status string
func RenderStatus(status string) string {
	switch status {
	case "open":
		return StatusOpen.Render(status)
	case "closed":
		return StatusClosed.Render(status)
	case "in-progress", "in_progress":
		return StatusInProgress.Render(status)
	default:
		return Muted.Render(status)
	}
}

// RenderStatusText returns styled status text (for tables, no background)
func RenderStatusText(status string) string {
	switch status {
	case "open":
		return StatusOpenText.Render(status)
	case "closed":
		return StatusClosedText.Render(status)
	case "in-progress", "in_progress":
		return StatusInProgressText.Render("in-progress")
	default:
		return Muted.Render(status)
	}
}
