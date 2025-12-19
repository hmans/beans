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

// prioritySelectedMsg is sent when a priority is selected from the picker
type prioritySelectedMsg struct {
	beanIDs  []string
	priority string
}

// closePriorityPickerMsg is sent when the priority picker is cancelled
type closePriorityPickerMsg struct{}

// openPriorityPickerMsg requests opening the priority picker for bean(s)
type openPriorityPickerMsg struct {
	beanIDs         []string // IDs of beans to update
	beanTitle       string   // Display title (single title or "N beans")
	currentPriority string   // Only meaningful for single bean
}

// priorityItem wraps a priority to implement list.Item
type priorityItem struct {
	name        string
	description string
	color       string
	isCurrent   bool
}

func (i priorityItem) Title() string       { return i.name }
func (i priorityItem) Description() string { return i.description }
func (i priorityItem) FilterValue() string { return i.name + " " + i.description }

// priorityItemDelegate handles rendering of priority picker items
type priorityItemDelegate struct{}

func (d priorityItemDelegate) Height() int                             { return 1 }
func (d priorityItemDelegate) Spacing() int                            { return 0 }
func (d priorityItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d priorityItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(priorityItem)
	if !ok {
		return
	}

	var cursor string
	if index == m.Index() {
		cursor = lipgloss.NewStyle().Foreground(ui.ColorPrimary).Bold(true).Render("▌") + " "
	} else {
		cursor = "  "
	}

	// Render priority with color
	priorityStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(item.color))
	priorityText := priorityStyle.Render(item.name)

	// Add current indicator
	var currentIndicator string
	if item.isCurrent {
		currentIndicator = ui.Muted.Render(" (current)")
	}

	fmt.Fprint(w, cursor+priorityText+currentIndicator)
}

// priorityPickerModel is the model for the priority picker view
type priorityPickerModel struct {
	list            list.Model
	beanIDs         []string
	beanTitle       string
	currentPriority string
	width           int
	height          int
}

func newPriorityPickerModel(beanIDs []string, beanTitle, currentPriority string, cfg *config.Config, width, height int) priorityPickerModel {
	// Get all priorities (hardcoded in config package)
	priorities := config.DefaultPriorities

	delegate := priorityItemDelegate{}

	// Build items list
	items := make([]list.Item, 0, len(priorities))
	selectedIndex := 0

	for i, p := range priorities {
		isCurrent := p.Name == currentPriority
		if isCurrent {
			selectedIndex = i
		}
		items = append(items, priorityItem{
			name:        p.Name,
			description: p.Description,
			color:       p.Color,
			isCurrent:   isCurrent,
		})
	}

	// Calculate modal dimensions
	modalWidth := max(40, min(60, width*50/100))
	modalHeight := max(10, min(16, height*50/100))
	listWidth := modalWidth - 6
	listHeight := modalHeight - 7

	l := list.New(items, delegate, listWidth, listHeight)
	l.Title = "Select Priority"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.Styles.Title = listTitleStyle
	l.Styles.TitleBar = lipgloss.NewStyle().Padding(0, 0, 0, 0)
	l.Styles.FilterPrompt = lipgloss.NewStyle().Foreground(ui.ColorPrimary)
	l.Styles.FilterCursor = lipgloss.NewStyle().Foreground(ui.ColorPrimary)

	// Select the current priority
	if selectedIndex < len(items) {
		l.Select(selectedIndex)
	}

	return priorityPickerModel{
		list:            l,
		beanIDs:         beanIDs,
		beanTitle:       beanTitle,
		currentPriority: currentPriority,
		width:           width,
		height:          height,
	}
}

func (m priorityPickerModel) Init() tea.Cmd {
	return nil
}

func (m priorityPickerModel) Update(msg tea.Msg) (priorityPickerModel, tea.Cmd) {
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
				if item, ok := m.list.SelectedItem().(priorityItem); ok {
					return m, func() tea.Msg {
						return prioritySelectedMsg{beanIDs: m.beanIDs, priority: item.name}
					}
				}
			case "esc", "backspace":
				return m, func() tea.Msg {
					return closePriorityPickerMsg{}
				}
			}
		}
	}

	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m priorityPickerModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	// Get description of currently selected priority
	var description string
	if item, ok := m.list.SelectedItem().(priorityItem); ok && item.description != "" {
		description = item.description
	}

	// For multi-select, don't show individual bean ID
	var beanID string
	if len(m.beanIDs) == 1 {
		beanID = m.beanIDs[0]
	}

	return renderPickerModal(pickerModalConfig{
		Title:       "Select Priority",
		BeanTitle:   m.beanTitle,
		BeanID:      beanID,
		ListContent: m.list.View(),
		Description: description,
		Width:       m.width,
	})
}

// ModalView returns the picker rendered as a centered modal overlay on top of the background
func (m priorityPickerModel) ModalView(bgView string, fullWidth, fullHeight int) string {
	modal := m.View()
	return overlayModal(bgView, modal, fullWidth, fullHeight)
}
