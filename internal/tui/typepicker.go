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

// typeSelectedMsg is sent when a type is selected from the picker
type typeSelectedMsg struct {
	beanIDs  []string
	beanType string
}

// closeTypePickerMsg is sent when the type picker is cancelled
type closeTypePickerMsg struct{}

// openTypePickerMsg requests opening the type picker for bean(s)
type openTypePickerMsg struct {
	beanIDs     []string // IDs of beans to update
	beanTitle   string   // Display title (single title or "N beans")
	currentType string   // Only meaningful for single bean
}

// typeItem wraps a type to implement list.Item
type typeItem struct {
	name        string
	description string
	color       string
	isCurrent   bool
}

func (i typeItem) Title() string       { return i.name }
func (i typeItem) Description() string { return i.description }
func (i typeItem) FilterValue() string { return i.name + " " + i.description }

// typeItemDelegate handles rendering of type picker items
type typeItemDelegate struct{}

func (d typeItemDelegate) Height() int                             { return 1 }
func (d typeItemDelegate) Spacing() int                            { return 0 }
func (d typeItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d typeItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(typeItem)
	if !ok {
		return
	}

	var cursor string
	if index == m.Index() {
		cursor = lipgloss.NewStyle().Foreground(ui.ColorPrimary).Bold(true).Render("▌") + " "
	} else {
		cursor = "  "
	}

	// Render type with color
	typeText := ui.RenderTypeText(item.name, item.color)

	// Add current indicator
	var currentIndicator string
	if item.isCurrent {
		currentIndicator = ui.Muted.Render(" (current)")
	}

	fmt.Fprint(w, cursor+typeText+currentIndicator)
}

// typePickerModel is the model for the type picker view
type typePickerModel struct {
	list        list.Model
	beanIDs     []string
	beanTitle   string
	currentType string
	width       int
	height      int
}

func newTypePickerModel(beanIDs []string, beanTitle, currentType string, cfg *config.Config, width, height int) typePickerModel {
	// Get all types (hardcoded in config package)
	types := config.DefaultTypes

	delegate := typeItemDelegate{}

	// Build items list
	items := make([]list.Item, 0, len(types))
	selectedIndex := 0

	for i, t := range types {
		isCurrent := t.Name == currentType
		if isCurrent {
			selectedIndex = i
		}
		items = append(items, typeItem{
			name:        t.Name,
			description: t.Description,
			color:       t.Color,
			isCurrent:   isCurrent,
		})
	}

	// Calculate modal dimensions
	modalWidth := max(40, min(60, width*50/100))
	modalHeight := max(10, min(16, height*50/100))
	listWidth := modalWidth - 6
	listHeight := modalHeight - 7

	l := list.New(items, delegate, listWidth, listHeight)
	l.Title = "Select Type"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.Styles.Title = listTitleStyle
	l.Styles.TitleBar = lipgloss.NewStyle().Padding(0, 0, 0, 0)
	l.Styles.FilterPrompt = lipgloss.NewStyle().Foreground(ui.ColorPrimary)
	l.Styles.FilterCursor = lipgloss.NewStyle().Foreground(ui.ColorPrimary)

	// Select the current type
	if selectedIndex < len(items) {
		l.Select(selectedIndex)
	}

	return typePickerModel{
		list:        l,
		beanIDs:     beanIDs,
		beanTitle:   beanTitle,
		currentType: currentType,
		width:       width,
		height:      height,
	}
}

func (m typePickerModel) Init() tea.Cmd {
	return nil
}

func (m typePickerModel) Update(msg tea.Msg) (typePickerModel, tea.Cmd) {
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
				if item, ok := m.list.SelectedItem().(typeItem); ok {
					return m, func() tea.Msg {
						return typeSelectedMsg{beanIDs: m.beanIDs, beanType: item.name}
					}
				}
			case "esc", "backspace":
				return m, func() tea.Msg {
					return closeTypePickerMsg{}
				}
			}
		}
	}

	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m typePickerModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	// Get description of currently selected type
	var description string
	if item, ok := m.list.SelectedItem().(typeItem); ok && item.description != "" {
		description = item.description
	}

	// For multi-select, don't show individual bean ID
	var beanID string
	if len(m.beanIDs) == 1 {
		beanID = m.beanIDs[0]
	}

	return renderPickerModal(pickerModalConfig{
		Title:       "Select Type",
		BeanTitle:   m.beanTitle,
		BeanID:      beanID,
		ListContent: m.list.View(),
		Description: description,
		Width:       m.width,
	})
}

// ModalView returns the picker rendered as a centered modal overlay on top of the background
func (m typePickerModel) ModalView(bgView string, fullWidth, fullHeight int) string {
	modal := m.View()
	return overlayModal(bgView, modal, fullWidth, fullHeight)
}
