package tui

import (
	"fmt"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hmans/beans/internal/config"
	"github.com/hmans/beans/internal/ui"
)

// Default launcher configurations
type defaultLauncher struct {
	name        string
	command     string
	description string
}

var defaultLaunchers = []defaultLauncher{
	{
		name:        "opencode",
		command:     "opencode -p \"Work on task $BEANS_ID\"",
		description: "Open task in OpenCode",
	},
	{
		name:        "claude",
		command:     "claude \"Work on task $BEANS_ID\"",
		description: "Open task in Claude Code",
	},
	{
		name:        "crush",
		command:     "crush run \"Work on task $BEANS_ID\"",
		description: "Open task in Crush",
	},
}

// openConfigureLaunchersMsg requests opening the configure launchers dialog
type openConfigureLaunchersMsg struct {
	beanID    string
	beanTitle string
}

// launchersConfiguredMsg is sent when launchers have been selected and added to config
type launchersConfiguredMsg struct {
	launchers []config.Launcher
}

// closeConfigureLaunchersMsg is sent when the dialog is cancelled
type closeConfigureLaunchersMsg struct{}

// configWriteErrorMsg is sent when writing to config fails
type configWriteErrorMsg struct {
	err error
}

// launcherChecklistItem represents a single launcher option in the checklist
type launcherChecklistItem struct {
	launcher  defaultLauncher
	installed bool
	selected  bool
}

// configureLaunchersModel is the model for the configure launchers dialog
type configureLaunchersModel struct {
	items     []launcherChecklistItem
	cursor    int
	beanID    string
	beanTitle string
	width     int
	height    int
}

// newConfigureLaunchersModel creates a new configure launchers dialog
func newConfigureLaunchersModel(beanID, beanTitle string, width, height int) configureLaunchersModel {
	items := make([]launcherChecklistItem, 0, len(defaultLaunchers))

	// Check which default launchers are installed and pre-select them
	for _, dl := range defaultLaunchers {
		cmdName := extractMainCommand(dl.command)
		_, err := exec.LookPath(cmdName)
		installed := err == nil

		items = append(items, launcherChecklistItem{
			launcher:  dl,
			installed: installed,
			selected:  installed, // Pre-select installed tools
		})
	}

	return configureLaunchersModel{
		items:     items,
		cursor:    0,
		beanID:    beanID,
		beanTitle: beanTitle,
		width:     width,
		height:    height,
	}
}

func (m configureLaunchersModel) Init() tea.Cmd {
	return nil
}

func (m configureLaunchersModel) Update(msg tea.Msg) (configureLaunchersModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}

		case "space", "enter":
			// Toggle selection of current item
			if m.cursor < len(m.items) {
				m.items[m.cursor].selected = !m.items[m.cursor].selected
			}

		case "y":
			// Confirm selections
			var selectedLaunchers []config.Launcher
			for _, item := range m.items {
				if item.selected {
					selectedLaunchers = append(selectedLaunchers, config.Launcher{
						Name:        item.launcher.name,
						Command:     item.launcher.command,
						Description: item.launcher.description,
					})
				}
			}

			// If nothing selected, treat as cancel
			if len(selectedLaunchers) == 0 {
				return m, func() tea.Msg {
					return closeConfigureLaunchersMsg{}
				}
			}

			return m, func() tea.Msg {
				return launchersConfiguredMsg{launchers: selectedLaunchers}
			}

		case "esc", "q":
			return m, func() tea.Msg {
				return closeConfigureLaunchersMsg{}
			}
		}
	}

	return m, nil
}

func (m configureLaunchersModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	modalWidth := max(50, min(70, m.width*60/100))

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(ui.ColorPrimary).
		Render("Configure Default Launchers")

	subtitle := ui.Muted.Render(m.beanID + ": " + m.beanTitle)

	message := "No launchers configured. Select tools to add to .beans.yml:"

	// Render checklist
	var itemsStr strings.Builder
	for i, item := range m.items {
		cursor := "  "
		if i == m.cursor {
			cursor = ui.Primary.Render("▌") + " "
		}

		checkbox := "[ ]"
		if item.selected {
			checkbox = "[✓]"
		}

		status := ""
		if item.installed {
			status = " " + ui.Muted.Render("✓ installed")
		} else {
			status = " " + ui.Muted.Render("✗ not found")
		}

		itemsStr.WriteString(cursor + checkbox + " " + item.launcher.name + status + "\n")
	}

	help := helpKeyStyle.Render("y") + " " + helpStyle.Render("confirm") + "  " +
		helpKeyStyle.Render("space") + " " + helpStyle.Render("toggle") + "  " +
		helpKeyStyle.Render("esc") + " " + helpStyle.Render("cancel")

	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorPrimary).
		Padding(1, 2).
		Width(modalWidth)

	content := title + "\n" + subtitle + "\n\n" + message + "\n\n" + itemsStr.String() + "\n" + help

	return border.Render(content)
}

// ModalView returns the dialog as a centered modal on top of the background
func (m configureLaunchersModel) ModalView(bgView string, fullWidth, fullHeight int) string {
	modal := m.View()
	return overlayModal(bgView, modal, fullWidth, fullHeight)
}

// configWriteErrorModel displays an error when writing to config fails
type configWriteErrorModel struct {
	errorMsg string
	width    int
	height   int
}

func newConfigWriteErrorModel(err error, width, height int) configWriteErrorModel {
	return configWriteErrorModel{
		errorMsg: err.Error(),
		width:    width,
		height:   height,
	}
}

func (m configWriteErrorModel) View() string {
	modalWidth := max(40, min(60, m.width*50/100))

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#ff0000")).
		Render("Configuration Error")

	message := fmt.Sprintf("Failed to write to .beans.yml:\n\n%s", m.errorMsg)

	help := helpKeyStyle.Render("any key") + " " + helpStyle.Render("dismiss")

	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#ff0000")).
		Padding(1, 2).
		Width(modalWidth)

	content := title + "\n\n" + message + "\n\n" + help

	return border.Render(content)
}

// ModalView returns the error modal as a centered modal on top of the background
func (m configWriteErrorModel) ModalView(bgView string, fullWidth, fullHeight int) string {
	modal := m.View()
	return overlayModal(bgView, modal, fullWidth, fullHeight)
}
