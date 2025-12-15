package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/hmans/beans/internal/ui"
)

// noLaunchersMultiModel displays an error when no multi-bean launchers are available
type noLaunchersMultiModel struct {
	width  int
	height int
}

func newNoLaunchersMultiModel(width, height int) noLaunchersMultiModel {
	return noLaunchersMultiModel{width: width, height: height}
}

func (m noLaunchersMultiModel) View() string {
	modalWidth := max(40, min(60, m.width*50/100))

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(ui.ColorMuted).
		Render("No Multi-Bean Launchers Available")

	message := "No launchers configured to handle multiple beans.\n\nConfigure launchers with 'multiple: true' in .beans.yml"

	help := helpKeyStyle.Render("any key") + " " + helpStyle.Render("dismiss")

	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorMuted).
		Padding(1, 2).
		Width(modalWidth)

	content := title + "\n\n" + message + "\n\n" + help

	return border.Render(content)
}

// ModalView returns the error message as a centered modal on top of the background
func (m noLaunchersMultiModel) ModalView(bgView string, fullWidth, fullHeight int) string {
	modal := m.View()
	return overlayModal(bgView, modal, fullWidth, fullHeight)
}
