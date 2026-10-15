package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hmans/beans/internal/ui"
	"github.com/hmans/beans/pkg/config"
)

// Preview ratio bounds for position below, in percent of the height above the footer.
const (
	DefaultPreviewPercent = 50
	MinPreviewPercent     = 10
	MaxPreviewPercent     = 90
	previewPercentStep    = 10
)

// tuiSettings holds the layout settings that last until the TUI exits.
type tuiSettings struct {
	previewPosition config.PreviewPosition
	// previewPercent is the preview's share of the height above the footer,
	// used only for position below.
	previewPercent int
}

var previewPositions = []config.PreviewPosition{
	config.PreviewPositionAuto,
	config.PreviewPositionRight,
	config.PreviewPositionBelow,
}

// Rows of the settings overlay.
const (
	settingPreviewPosition = iota
	settingPreviewPercent
	settingViewMode
	settingCount
)

// settingsChangedMsg carries the settings after each change in the overlay.
type settingsChangedMsg struct {
	settings tuiSettings
	viewMode viewMode
}

// closeSettingsMsg is sent when the settings overlay is closed.
type closeSettingsMsg struct{}

// settingsOverlayModel edits the layout settings and the list's view mode.
type settingsOverlayModel struct {
	settings tuiSettings
	viewMode viewMode
	cursor   int
	width    int
	height   int
}

func newSettingsOverlayModel(settings tuiSettings, mode viewMode, width, height int) settingsOverlayModel {
	return settingsOverlayModel{settings: settings, viewMode: mode, width: width, height: height}
}

func (m settingsOverlayModel) Update(msg tea.Msg) (settingsOverlayModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case ",", "esc":
			return m, func() tea.Msg { return closeSettingsMsg{} }
		case "up", "k":
			m.cursor = (m.cursor + settingCount - 1) % settingCount
		case "down", "j":
			m.cursor = (m.cursor + 1) % settingCount
		case "left", "h":
			return m.change(-1)
		case "right", "l":
			return m.change(1)
		}
	}
	return m, nil
}

// change steps the value of the selected row by delta. Positions and view
// modes wrap around; the ratio stops at its bounds.
func (m settingsOverlayModel) change(delta int) (settingsOverlayModel, tea.Cmd) {
	switch m.cursor {
	case settingPreviewPosition:
		i := 0
		for j, p := range previewPositions {
			if p == m.settings.previewPosition {
				i = j
			}
		}
		n := len(previewPositions)
		m.settings.previewPosition = previewPositions[(i+delta+n)%n]
	case settingPreviewPercent:
		p := m.settings.previewPercent + delta*previewPercentStep
		m.settings.previewPercent = max(MinPreviewPercent, min(MaxPreviewPercent, p))
	case settingViewMode:
		m.viewMode = (m.viewMode + viewMode(delta) + viewModeCount) % viewModeCount
	}
	settings, mode := m.settings, m.viewMode
	return m, func() tea.Msg { return settingsChangedMsg{settings: settings, viewMode: mode} }
}

func (m settingsOverlayModel) View() string {
	title := lipgloss.NewStyle().Bold(true).Foreground(ui.ColorPrimary).Render("Settings")

	positions := make([]string, len(previewPositions))
	for i, p := range previewPositions {
		positions[i] = string(p)
	}
	modes := make([]string, viewModeCount)
	for v := range viewModeCount {
		modes[v] = v.String()
	}

	ratioNote := ""
	if m.settings.previewPosition != config.PreviewPositionBelow {
		ratioNote = ui.Muted.Render("  (below only)")
	}
	rows := []struct{ label, value, note string }{
		{"preview at", choice(positions, string(m.settings.previewPosition)), ""},
		{"preview ratio", fmt.Sprintf("%d%%", m.settings.previewPercent), ratioNote},
		{"view mode", choice(modes, m.viewMode.String()), ""},
	}

	var content strings.Builder
	content.WriteString(title + "\n\n")
	for i, row := range rows {
		label := lipgloss.NewStyle().Width(15).Render(row.label)
		marker := "  "
		if i == m.cursor {
			label = lipgloss.NewStyle().Width(15).Bold(true).Foreground(ui.ColorPrimary).Render(row.label)
			marker = lipgloss.NewStyle().Foreground(ui.ColorPrimary).Render("▸ ")
		}
		content.WriteString(marker + label + "‹ " + row.value + " ›" + row.note + "\n")
	}
	content.WriteString("\n")
	content.WriteString(helpKeyStyle.Render("↑↓") + " " + helpStyle.Render("select") + "  " +
		helpKeyStyle.Render("←→") + " " + helpStyle.Render("change") + "  " +
		helpKeyStyle.Render(",/esc") + " " + helpStyle.Render("close"))

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorPrimary).
		Padding(1, 2).
		Render(content.String())
}

// choice renders the options with the current one highlighted.
func choice(options []string, current string) string {
	parts := make([]string, len(options))
	for i, o := range options {
		if o == current {
			parts[i] = lipgloss.NewStyle().Bold(true).Foreground(ui.ColorPrimary).Render(o)
		} else {
			parts[i] = ui.Muted.Render(o)
		}
	}
	return strings.Join(parts, " | ")
}

// ModalView returns the settings overlay centered on top of the background.
func (m settingsOverlayModel) ModalView(bgView string, fullWidth, fullHeight int) string {
	return overlayModal(bgView, m.View(), fullWidth, fullHeight)
}
