package tui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hmans/beans/internal/ui"
)

// openLauncherPickerMsg requests opening the launcher picker
type openLauncherPickerMsg struct {
	beanIDs   []string
	beanTitle string
}

// launcherSelectedMsg is sent when a launcher is selected
type launcherSelectedMsg struct {
	launcher launcher
	beanIDs  []string
}

// closeLauncherPickerMsg is sent when the launcher picker is cancelled
type closeLauncherPickerMsg struct{}

// launcherFinishedMsg is sent when launcher execution completes
type launcherFinishedMsg struct {
	err          error
	launcherName string
}

// launcherItem wraps a launcher to implement list.Item
type launcherItem struct {
	launcher launcher
}

func (i launcherItem) Title() string       { return i.launcher.name }
func (i launcherItem) Description() string { return i.launcher.description }
func (i launcherItem) FilterValue() string {
	return i.launcher.name + " " + i.launcher.description
}

// launcherItemDelegate handles rendering of launcher picker items
type launcherItemDelegate struct{}

func (d launcherItemDelegate) Height() int                             { return 1 }
func (d launcherItemDelegate) Spacing() int                            { return 0 }
func (d launcherItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d launcherItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(launcherItem)
	if !ok {
		return
	}

	var cursor string
	if index == m.Index() {
		cursor = lipgloss.NewStyle().Foreground(ui.ColorPrimary).Bold(true).Render("▌") + " "
	} else {
		cursor = "  "
	}

	name := lipgloss.NewStyle().Bold(true).Render(item.launcher.name)

	if item.launcher.description != "" {
		desc := " " + ui.Muted.Render("- "+item.launcher.description)
		fmt.Fprint(w, cursor+name+desc)
	} else {
		fmt.Fprint(w, cursor+name)
	}
}

// launcherPickerModel is the model for the launcher picker view
type launcherPickerModel struct {
	list      list.Model
	beanIDs   []string
	beanTitle string
	width     int
	height    int
}

func newLauncherPickerModel(launchers []launcher, beanIDs []string, beanTitle string, width, height int) launcherPickerModel {
	delegate := launcherItemDelegate{}

	// Build items list
	items := make([]list.Item, len(launchers))
	for i, l := range launchers {
		items[i] = launcherItem{launcher: l}
	}

	// Calculate modal dimensions
	modalWidth := max(40, min(60, width*50/100))
	modalHeight := max(10, min(20, height*50/100))
	listWidth := modalWidth - 6
	listHeight := modalHeight - 7

	l := list.New(items, delegate, listWidth, listHeight)
	l.Title = "Select Launcher"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.Styles.Title = listTitleStyle
	l.Styles.TitleBar = lipgloss.NewStyle().Padding(0, 0, 0, 0)
	l.Styles.FilterPrompt = lipgloss.NewStyle().Foreground(ui.ColorPrimary)
	l.Styles.FilterCursor = lipgloss.NewStyle().Foreground(ui.ColorPrimary)

	return launcherPickerModel{
		list:      l,
		beanIDs:   beanIDs,
		beanTitle: beanTitle,
		width:     width,
		height:    height,
	}
}

func (m launcherPickerModel) Init() tea.Cmd {
	return nil
}

func (m launcherPickerModel) Update(msg tea.Msg) (launcherPickerModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		modalWidth := max(40, min(60, msg.Width*50/100))
		modalHeight := max(10, min(20, msg.Height*50/100))
		listWidth := modalWidth - 6
		listHeight := modalHeight - 7
		m.list.SetSize(listWidth, listHeight)

	case tea.KeyMsg:
		if m.list.FilterState() != list.Filtering {
			switch msg.String() {
			case "enter":
				if item, ok := m.list.SelectedItem().(launcherItem); ok {
					return m, func() tea.Msg {
						return launcherSelectedMsg{launcher: item.launcher, beanIDs: m.beanIDs}
					}
				}
			case "esc", "backspace":
				return m, func() tea.Msg {
					return closeLauncherPickerMsg{}
				}
			}
		}
	}

	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m launcherPickerModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	// For display, show the first bean ID if there's only one, or count if multiple
	beanIDDisplay := ""
	if len(m.beanIDs) == 1 {
		beanIDDisplay = m.beanIDs[0]
	} else if len(m.beanIDs) > 1 {
		beanIDDisplay = fmt.Sprintf("%d selected beans", len(m.beanIDs))
	}

	return renderPickerModal(pickerModalConfig{
		Title:       "Select Launcher",
		BeanTitle:   m.beanTitle,
		BeanID:      beanIDDisplay,
		ListContent: m.list.View(),
		Width:       m.width,
	})
}

// ModalView returns the picker rendered as a centered modal overlay on top of the background
func (m launcherPickerModel) ModalView(bgView string, fullWidth, fullHeight int) string {
	modal := m.View()
	return overlayModal(bgView, modal, fullWidth, fullHeight)
}

// noLaunchersModel displays an error when no launchers are available
type noLaunchersModel struct {
	width  int
	height int
}

func newNoLaunchersModel(width, height int) noLaunchersModel {
	return noLaunchersModel{width: width, height: height}
}

func (m noLaunchersModel) View() string {
	modalWidth := max(40, min(60, m.width*50/100))

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(ui.ColorMuted).
		Render("No Launchers Available")

	message := "No compatible launchers found.\n\nConfigure launchers in .beans.yml"

	help := helpKeyStyle.Render("any key") + " " + helpStyle.Render("dismiss")

	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorMuted).
		Padding(1, 2).
		Width(modalWidth)

	content := title + "\n\n" + message + "\n\n" + help

	return border.Render(content)
}

// ModalView returns the no launchers message as a centered modal on top of the background
func (m noLaunchersModel) ModalView(bgView string, fullWidth, fullHeight int) string {
	modal := m.View()
	return overlayModal(bgView, modal, fullWidth, fullHeight)
}

// launcherErrorModel displays an error after launcher execution fails
type launcherErrorModel struct {
	launcherName string
	errorMsg     string
	width        int
	height       int
}

func newLauncherErrorModel(launcherName string, err error, width, height int) launcherErrorModel {
	return launcherErrorModel{
		launcherName: launcherName,
		errorMsg:     err.Error(),
		width:        width,
		height:       height,
	}
}

func (m launcherErrorModel) View() string {
	modalWidth := max(40, min(60, m.width*50/100))

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#ff0000")).
		Render("Launcher Failed")

	message := fmt.Sprintf("Failed to run '%s':\n\n%s", m.launcherName, m.errorMsg)

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
func (m launcherErrorModel) ModalView(bgView string, fullWidth, fullHeight int) string {
	modal := m.View()
	return overlayModal(bgView, modal, fullWidth, fullHeight)
}
