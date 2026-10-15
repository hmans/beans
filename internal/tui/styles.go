package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/hmans/beans/internal/ui"
)

var (
	// List title style
	listTitleStyle = lipgloss.NewStyle().
			Foreground(ui.ColorPrimary).
			Bold(true)

	// Detail title style
	detailTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#fff")).
				Background(ui.ColorPrimary).
				Padding(0, 1)

	// Help text style
	helpStyle = lipgloss.NewStyle().
			Foreground(ui.ColorMuted)

	// Help key style
	helpKeyStyle = lipgloss.NewStyle().
			Foreground(ui.ColorPrimary).
			Bold(true)

	paneBorderStyle = lipgloss.NewStyle().Foreground(ui.ColorMuted)
)

// withBorderTitle replaces the top border line of a pane rendered with
// lipgloss.RoundedBorder by one that carries title, truncated to fit.
func withBorderTitle(pane, title string) string {
	top, rest, _ := strings.Cut(pane, "\n")
	width := lipgloss.Width(top)
	// "╭─ " + title + " " + fill + "╮", with at least one "─" of fill.
	const chrome = 5
	if width <= chrome+1 {
		return pane
	}
	title = ansi.Truncate(title, width-chrome-1, "…")
	b := lipgloss.RoundedBorder()
	fill := width - chrome - lipgloss.Width(title)
	top = paneBorderStyle.Render(b.TopLeft+b.Top+" ") + title +
		paneBorderStyle.Render(" "+strings.Repeat(b.Top, fill)+b.TopRight)
	return top + "\n" + rest
}
