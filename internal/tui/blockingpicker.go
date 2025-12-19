package tui

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/config"
	"github.com/hmans/beans/internal/graph"
	"github.com/hmans/beans/internal/ui"
)

// blockingConfirmedMsg is sent when blocking changes are confirmed
type blockingConfirmedMsg struct {
	beanID  string            // the bean we're modifying
	toAdd   []string          // IDs to add to blocking
	toRemove []string         // IDs to remove from blocking
}

// closeBlockingPickerMsg is sent when the blocking picker is cancelled
type closeBlockingPickerMsg struct{}

// openBlockingPickerMsg requests opening the blocking picker for a bean
type openBlockingPickerMsg struct {
	beanID          string
	beanTitle       string
	currentBlocking []string // IDs of beans currently being blocked
}

// blockingItem wraps a bean to implement list.Item for the blocking picker
type blockingItem struct {
	bean *bean.Bean
	cfg  *config.Config
}

func (i blockingItem) Title() string       { return i.bean.Title }
func (i blockingItem) Description() string { return i.bean.ID }
func (i blockingItem) FilterValue() string { return i.bean.Title + " " + i.bean.ID }

// blockingItemDelegate handles rendering of blocking picker items
type blockingItemDelegate struct {
	cfg             *config.Config
	pendingBlocking *map[string]bool // pointer to pending state for live updates
}

func (d blockingItemDelegate) Height() int                             { return 1 }
func (d blockingItemDelegate) Spacing() int                            { return 0 }
func (d blockingItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d blockingItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(blockingItem)
	if !ok {
		return
	}

	var cursor string
	if index == m.Index() {
		cursor = lipgloss.NewStyle().Foreground(ui.ColorPrimary).Bold(true).Render("▌") + " "
	} else {
		cursor = "  "
	}

	// Show blocking indicator - read from pending state for live updates
	isBlocking := (*d.pendingBlocking)[item.bean.ID]
	var blockingIndicator string
	if isBlocking {
		blockingIndicator = lipgloss.NewStyle().Foreground(ui.ColorDanger).Bold(true).Render("● ") // Red dot for blocking
	} else {
		blockingIndicator = lipgloss.NewStyle().Foreground(ui.ColorMuted).Render("○ ") // Empty circle for not blocking
	}

	// Get colors from config
	colors := d.cfg.GetBeanColors(item.bean.Status, item.bean.Type, item.bean.Priority)

	// Format: [indicator] [type] title (id)
	typeBadge := ui.RenderTypeText(item.bean.Type, colors.TypeColor)
	title := item.bean.Title
	if colors.IsArchive {
		title = ui.Muted.Render(title)
	}
	id := ui.Muted.Render(" (" + item.bean.ID + ")")

	fmt.Fprint(w, cursor+blockingIndicator+typeBadge+" "+title+id)
}

// blockingPickerModel is the model for the blocking picker view
type blockingPickerModel struct {
	list             list.Model
	beanID           string           // the bean we're setting blocking for
	beanTitle        string           // the bean's title
	originalBlocking map[string]bool  // original state (for computing diff)
	pendingBlocking  map[string]bool  // pending state (toggled by space)
	cfg              *config.Config
	width            int
	height           int
}

func newBlockingPickerModel(beanID, beanTitle string, currentBlocking []string, resolver *graph.Resolver, cfg *config.Config, width, height int) blockingPickerModel {
	// Fetch all beans
	allBeans, _ := resolver.Query().Beans(context.Background(), nil)

	// Create maps for original and pending state
	originalBlocking := make(map[string]bool)
	pendingBlocking := make(map[string]bool)
	for _, id := range currentBlocking {
		originalBlocking[id] = true
		pendingBlocking[id] = true
	}

	// Filter out the current bean and build items
	var eligibleBeans []*bean.Bean
	for _, b := range allBeans {
		if b.ID != beanID {
			eligibleBeans = append(eligibleBeans, b)
		}
	}

	// Sort by type order, then by title
	typeNames := cfg.TypeNames()
	typeOrder := make(map[string]int)
	for i, t := range typeNames {
		typeOrder[t] = i
	}
	sort.Slice(eligibleBeans, func(i, j int) bool {
		ti, tj := typeOrder[eligibleBeans[i].Type], typeOrder[eligibleBeans[j].Type]
		if ti != tj {
			return ti < tj
		}
		return strings.ToLower(eligibleBeans[i].Title) < strings.ToLower(eligibleBeans[j].Title)
	})

	// Build items list
	items := make([]list.Item, 0, len(eligibleBeans))
	for _, b := range eligibleBeans {
		items = append(items, blockingItem{
			bean: b,
			cfg:  cfg,
		})
	}

	// Calculate modal dimensions (60% width, 60% height, with min/max constraints)
	modalWidth := max(40, min(80, width*60/100))
	modalHeight := max(10, min(20, height*60/100))
	listWidth := modalWidth - 6
	// Account for: header(1) + subtitle(1) + blank(1) + blank(1) + description(1) + blank(1) + help(1) + border(2) = 9
	listHeight := modalHeight - 9

	// Create delegate with pointer to pending state (so it can read live updates)
	delegate := blockingItemDelegate{cfg: cfg, pendingBlocking: &pendingBlocking}

	l := list.New(items, delegate, listWidth, listHeight)
	l.Title = "Manage Blocking"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.Styles.Title = listTitleStyle
	l.Styles.TitleBar = lipgloss.NewStyle().Padding(0, 0, 0, 0)
	l.Styles.FilterPrompt = lipgloss.NewStyle().Foreground(ui.ColorPrimary)
	l.Styles.FilterCursor = lipgloss.NewStyle().Foreground(ui.ColorPrimary)

	return blockingPickerModel{
		list:             l,
		beanID:           beanID,
		beanTitle:        beanTitle,
		originalBlocking: originalBlocking,
		pendingBlocking:  pendingBlocking,
		cfg:              cfg,
		width:            width,
		height:           height,
	}
}

func (m blockingPickerModel) Init() tea.Cmd {
	return nil
}

func (m blockingPickerModel) Update(msg tea.Msg) (blockingPickerModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		modalWidth := max(40, min(80, msg.Width*60/100))
		modalHeight := max(10, min(20, msg.Height*60/100))
		listWidth := modalWidth - 6
		listHeight := modalHeight - 9 // Account for description line
		m.list.SetSize(listWidth, listHeight)

	case tea.KeyMsg:
		if m.list.FilterState() != list.Filtering {
			switch msg.String() {
			case " ":
				// Toggle the selected item's pending state
				// The delegate reads from pendingBlocking directly, so no need to update items
				if item, ok := m.list.SelectedItem().(blockingItem); ok {
					targetID := item.bean.ID
					if m.pendingBlocking[targetID] {
						delete(m.pendingBlocking, targetID)
					} else {
						m.pendingBlocking[targetID] = true
					}
				}
				return m, nil

			case "enter":
				// Confirm changes - compute diff and send message
				var toAdd, toRemove []string

				// Find additions (in pending but not in original)
				for id := range m.pendingBlocking {
					if !m.originalBlocking[id] {
						toAdd = append(toAdd, id)
					}
				}

				// Find removals (in original but not in pending)
				for id := range m.originalBlocking {
					if !m.pendingBlocking[id] {
						toRemove = append(toRemove, id)
					}
				}

				return m, func() tea.Msg {
					return blockingConfirmedMsg{
						beanID:   m.beanID,
						toAdd:    toAdd,
						toRemove: toRemove,
					}
				}

			case "esc", "backspace":
				// Cancel - discard changes
				return m, func() tea.Msg {
					return closeBlockingPickerMsg{}
				}
			}
		}
	}

	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m blockingPickerModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	return renderPickerModal(pickerModalConfig{
		Title:       "Manage Blocking",
		BeanTitle:   m.beanTitle,
		BeanID:      m.beanID,
		ListContent: m.list.View(),
		Description: "space toggle, enter confirm, esc cancel",
		Width:       m.width,
		WidthPct:    60,
		MaxWidth:    80,
	})
}

// ModalView returns the picker rendered as a centered modal overlay on top of the background
func (m blockingPickerModel) ModalView(bgView string, fullWidth, fullHeight int) string {
	modal := m.View()
	return overlayModal(bgView, modal, fullWidth, fullHeight)
}
