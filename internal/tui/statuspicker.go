package tui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hmans/beans/internal/config"
	"github.com/hmans/beans/internal/ui"
)

// statusSelectedMsg is sent when a status is selected from the picker
type statusSelectedMsg struct {
	beanIDs []string
	status  string
}

// closeStatusPickerMsg is sent when the status picker is cancelled
type closeStatusPickerMsg struct{}

// openStatusPickerMsg requests opening the status picker for bean(s)
type openStatusPickerMsg struct {
	beanIDs       []string // IDs of beans to update
	beanTitle     string   // Display title (single title or "N beans")
	currentStatus string   // Only meaningful for single bean
}

// statusItem wraps a status to implement list.Item
type statusItem struct {
	name        string
	description string
	color       string
	isArchive   bool
	isCurrent   bool
}

func (i statusItem) Title() string       { return i.name }
func (i statusItem) Description() string { return i.description }
func (i statusItem) FilterValue() string { return i.name + " " + i.description }

// statusItemDelegate handles rendering of status picker items
type statusItemDelegate struct{}

func (d statusItemDelegate) Height() int                             { return 1 }
func (d statusItemDelegate) Spacing() int                            { return 0 }
func (d statusItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d statusItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(statusItem)
	if !ok {
		return
	}

	var cursor string
	if index == m.Index() {
		cursor = lipgloss.NewStyle().Foreground(ui.ColorPrimary).Bold(true).Render("▌") + " "
	} else {
		cursor = "  "
	}

	// Render status with color (text only)
	statusText := ui.RenderStatusTextWithColor(item.name, item.color, item.isArchive)

	// Add current indicator
	var currentIndicator string
	if item.isCurrent {
		currentIndicator = ui.Muted.Render(" (current)")
	}

	fmt.Fprint(w, cursor+statusText+currentIndicator)
}

// statusPickerModel is the model for the status picker view
type statusPickerModel struct {
	list          list.Model
	beanIDs       []string
	beanTitle     string
	currentStatus string
	width         int
	height        int
}

func newStatusPickerModel(beanIDs []string, beanTitle, currentStatus string, cfg *config.Config, width, height int) statusPickerModel {
	// Get all statuses (hardcoded in config package)
	statuses := config.DefaultStatuses

	delegate := statusItemDelegate{}

	// Build items list
	items := make([]list.Item, 0, len(statuses))
	selectedIndex := 0

	for i, s := range statuses {
		isCurrent := s.Name == currentStatus
		if isCurrent {
			selectedIndex = i
		}
		items = append(items, statusItem{
			name:        s.Name,
			description: s.Description,
			color:       s.Color,
			isArchive:   s.Archive,
			isCurrent:   isCurrent,
		})
	}

	// Calculate modal dimensions
	modalWidth := max(40, min(60, width*50/100))
	modalHeight := max(10, min(16, height*50/100))
	listWidth := modalWidth - 6
	listHeight := modalHeight - 7

	l := list.New(items, delegate, listWidth, listHeight)
	l.Title = "Select Status"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.Styles.Title = listTitleStyle
	l.Styles.TitleBar = lipgloss.NewStyle().Padding(0, 0, 0, 0)
	l.Styles.FilterPrompt = lipgloss.NewStyle().Foreground(ui.ColorPrimary)
	l.Styles.FilterCursor = lipgloss.NewStyle().Foreground(ui.ColorPrimary)

	// Select the current status
	if selectedIndex < len(items) {
		l.Select(selectedIndex)
	}

	return statusPickerModel{
		list:          l,
		beanIDs:       beanIDs,
		beanTitle:     beanTitle,
		currentStatus: currentStatus,
		width:         width,
		height:        height,
	}
}

func (m statusPickerModel) Init() tea.Cmd {
	return nil
}

func (m statusPickerModel) Update(msg tea.Msg) (statusPickerModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		modalWidth := max(40, min(60, msg.Width*50/100))
		modalHeight := max(10, min(16, msg.Height*50/100))
		listWidth := modalWidth - 6
		listHeight := modalHeight - 7
		m.list.SetSize(listWidth, listHeight)

	case tea.KeyMsg:
		if m.list.FilterState() != list.Filtering {
			switch msg.String() {
			case "enter":
				if item, ok := m.list.SelectedItem().(statusItem); ok {
					return m, func() tea.Msg {
						return statusSelectedMsg{beanIDs: m.beanIDs, status: item.name}
					}
				}
			case "esc", "backspace":
				return m, func() tea.Msg {
					return closeStatusPickerMsg{}
				}
			}
		}
	}

	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m statusPickerModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	// Get description of currently selected status
	var description string
	if item, ok := m.list.SelectedItem().(statusItem); ok && item.description != "" {
		description = item.description
	}

	// For multi-select, don't show individual bean ID
	var beanID string
	if len(m.beanIDs) == 1 {
		beanID = m.beanIDs[0]
	}

	return renderPickerModal(pickerModalConfig{
		Title:       "Select Status",
		BeanTitle:   m.beanTitle,
		BeanID:      beanID,
		ListContent: m.list.View(),
		Description: description,
		Width:       m.width,
	})
}

// ModalView returns the picker rendered as a centered modal overlay on top of the background
func (m statusPickerModel) ModalView(bgView string, fullWidth, fullHeight int) string {
	modal := m.View()
	return overlayModal(bgView, modal, fullWidth, fullHeight)
}
