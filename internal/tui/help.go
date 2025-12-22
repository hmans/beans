package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hmans/beans/internal/ui"
)

// openHelpMsg requests opening the help overlay
type openHelpMsg struct{}

// closeHelpMsg is sent when the help overlay is closed
type closeHelpMsg struct{}

// helpOverlayModel displays keyboard shortcuts organized by context
type helpOverlayModel struct {
	width  int
	height int
}

func newHelpOverlayModel(width, height int) helpOverlayModel {
	return helpOverlayModel{
		width:  width,
		height: height,
	}
}

func (m helpOverlayModel) Init() tea.Cmd {
	return nil
}

func (m helpOverlayModel) Update(msg tea.Msg) (helpOverlayModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "?", "esc":
			return m, func() tea.Msg {
				return closeHelpMsg{}
			}
		}
	}

	return m, nil
}

func (m helpOverlayModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	// Calculate modal dimensions
	modalWidth := max(50, min(60, m.width*60/100))

	// Title
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(ui.ColorPrimary).
		Render("Keyboard Shortcuts")

	// Helper to create a shortcut line
	shortcut := func(key, desc string) string {
		keyStyle := lipgloss.NewStyle().
			Foreground(ui.ColorPrimary).
			Bold(true).
			Width(12).
			Render(key)
		descStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#fff")).
			Render(desc)
		return keyStyle + descStyle
	}

	var content strings.Builder
	content.WriteString(title + "\n\n")

	content.WriteString(shortcut("b", "Manage blocking") + "\n")
	content.WriteString(shortcut("c", "Create new bean") + "\n")
	content.WriteString(shortcut("e", "Edit in $EDITOR") + "\n")
	content.WriteString(shortcut("p", "Set parent") + "\n")
	content.WriteString(shortcut("P", "Change priority") + "\n")
	content.WriteString(shortcut("s", "Change status") + "\n")
	content.WriteString(shortcut("t", "Change type") + "\n")
	content.WriteString(shortcut("y", "Copy bean ID") + "\n")
	content.WriteString(shortcut("/", "Filter") + "\n")
	content.WriteString(shortcut("g t", "Filter by tag") + "\n")
	content.WriteString(shortcut("q", "Quit") + "\n")
	content.WriteString("\n")

	// Footer
	footer := helpKeyStyle.Render("?/esc") + " " + helpStyle.Render("close")

	// Border style
	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorPrimary).
		Padding(1, 2).
		Width(modalWidth)

	return border.Render(content.String() + footer)
}

// ModalView returns the help overlay as a centered modal on top of the background
func (m helpOverlayModel) ModalView(bgView string, fullWidth, fullHeight int) string {
	modal := m.View()
	return overlayModal(bgView, modal, fullWidth, fullHeight)
}
