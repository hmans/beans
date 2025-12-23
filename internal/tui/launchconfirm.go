package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hmans/beans/internal/config"
	"github.com/hmans/beans/internal/ui"
)

// launchConfirmedMsg is sent when user confirms the launch
type launchConfirmedMsg struct {
	launcher *config.Launcher
	beanIDs  []string
}

// launchCancelledMsg is sent when user cancels the launch
type launchCancelledMsg struct{}

// launchConfirm displays a confirmation dialog for launching multiple beans
type launchConfirm struct {
	launcher *config.Launcher
	beanIDs  []string
	width    int
	height   int
}

func newLaunchConfirm(launcher *config.Launcher, beanIDs []string) launchConfirm {
	return launchConfirm{
		launcher: launcher,
		beanIDs:  beanIDs,
	}
}

func (m launchConfirm) Init() tea.Cmd {
	return nil
}

func (m launchConfirm) Update(msg tea.Msg) (launchConfirm, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y", "enter":
			// Confirm
			return m, func() tea.Msg {
				return launchConfirmedMsg{
					launcher: m.launcher,
					beanIDs:  m.beanIDs,
				}
			}

		case "n", "N", "q", "esc":
			// Cancel
			return m, func() tea.Msg {
				return launchCancelledMsg{}
			}
		}
	}

	return m, nil
}

func (m launchConfirm) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	// Build the dialog
	count := len(m.beanIDs)
	title := fmt.Sprintf("Launch %s for %d beans?", m.launcher.Name, count)

	warning := "This will run the launcher in parallel for all selected beans."

	prompt := "Press y to confirm, n to cancel"

	// Style the dialog
	titleStyled := confirmTitleStyle.Render(title)
	warningStyled := confirmWarningStyle.Render(warning)
	promptStyled := confirmPromptStyle.Render(prompt)

	// Create a centered box
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		titleStyled,
		"",
		warningStyled,
		"",
		promptStyled,
	)

	// Add padding and border
	dialog := confirmDialogStyle.Render(content)

	// Center in the screen
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		dialog,
	)
}

// Styles for confirmation dialog
var (
	confirmDialogStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ui.ColorPrimary).
				Padding(1, 2).
				Width(60)

	confirmTitleStyle = lipgloss.NewStyle().
				Foreground(ui.ColorPrimary).
				Bold(true).
				Align(lipgloss.Center)

	confirmWarningStyle = lipgloss.NewStyle().
				Foreground(ui.ColorWarning)

	confirmPromptStyle = lipgloss.NewStyle().
				Foreground(ui.ColorMuted)
)
