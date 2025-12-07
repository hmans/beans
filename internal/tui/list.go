package tui

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"hmans.dev/beans/internal/bean"
	"hmans.dev/beans/internal/beancore"
	"hmans.dev/beans/internal/config"
	"hmans.dev/beans/internal/ui"
)

// beanItem wraps a Bean to implement list.Item
type beanItem struct {
	bean *bean.Bean
	cfg  *config.Config
}

func (i beanItem) Title() string       { return i.bean.Title }
func (i beanItem) Description() string { return i.bean.ID + " · " + i.bean.Status }
func (i beanItem) FilterValue() string { return i.bean.Title + " " + i.bean.ID }

// itemDelegate handles rendering of list items
type itemDelegate struct {
	cfg     *config.Config
	hasTags bool
	width   int
	cols    ui.ResponsiveColumns // cached responsive columns
}

func newItemDelegate(cfg *config.Config) itemDelegate {
	return itemDelegate{cfg: cfg, hasTags: false, width: 0}
}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(beanItem)
	if !ok {
		return
	}

	// Get status color from config
	statusCfg := d.cfg.GetStatus(item.bean.Status)
	statusColor := "gray"
	if statusCfg != nil {
		statusColor = statusCfg.Color
	}
	isArchive := d.cfg.IsArchiveStatus(item.bean.Status)

	// Get type color from config
	typeColor := ""
	if typeCfg := d.cfg.GetType(item.bean.Type); typeCfg != nil {
		typeColor = typeCfg.Color
	}

	// Calculate max title width using responsive columns
	baseWidth := d.cols.ID + d.cols.Status + d.cols.Type + 4 // 4 for cursor + padding
	if d.cols.ShowTags {
		baseWidth += d.cols.Tags
	}
	maxTitleWidth := max(0, m.Width()-baseWidth)

	str := ui.RenderBeanRow(
		item.bean.ID,
		item.bean.Status,
		item.bean.Type,
		item.bean.Title,
		ui.BeanRowConfig{
			StatusColor:   statusColor,
			TypeColor:     typeColor,
			IsArchive:     isArchive,
			MaxTitleWidth: maxTitleWidth,
			ShowCursor:    true,
			IsSelected:    index == m.Index(),
			Tags:          item.bean.Tags,
			ShowTags:      d.cols.ShowTags,
			TagsColWidth:  d.cols.Tags,
			MaxTags:       d.cols.MaxTags,
		},
	)

	fmt.Fprint(w, str)
}

// listModel is the model for the bean list view
type listModel struct {
	list   list.Model
	core   *beancore.Core
	config *config.Config
	width  int
	height int
	err    error

	// Responsive column state
	hasTags bool                 // whether any beans have tags
	cols    ui.ResponsiveColumns // calculated responsive columns

	// Active filters
	tagFilter string // if set, only show beans with this tag
}

func newListModel(core *beancore.Core, cfg *config.Config) listModel {
	delegate := newItemDelegate(cfg)

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "Beans"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)
	l.Styles.Title = listTitleStyle
	l.Styles.TitleBar = lipgloss.NewStyle().Padding(0, 0, 1, 2)
	l.Styles.FilterPrompt = lipgloss.NewStyle().Foreground(ui.ColorPrimary)
	l.Styles.FilterCursor = lipgloss.NewStyle().Foreground(ui.ColorPrimary)

	return listModel{
		list:   l,
		core:   core,
		config: cfg,
	}
}

// beansLoadedMsg is sent when beans are loaded
type beansLoadedMsg struct {
	beans []*bean.Bean
}

// errMsg is sent when an error occurs
type errMsg struct {
	err error
}

// selectBeanMsg is sent when a bean is selected
type selectBeanMsg struct {
	bean *bean.Bean
}

func (m listModel) Init() tea.Cmd {
	return m.loadBeans
}

func (m listModel) loadBeans() tea.Msg {
	// Core already has beans in memory
	allBeans := m.core.All()

	// Apply tag filter if set
	if m.tagFilter != "" {
		var filtered []*bean.Bean
		for _, b := range allBeans {
			if b.HasTag(m.tagFilter) {
				filtered = append(filtered, b)
			}
		}
		return beansLoadedMsg{filtered}
	}

	return beansLoadedMsg{allBeans}
}

// setTagFilter sets the tag filter
func (m *listModel) setTagFilter(tag string) {
	m.tagFilter = tag
}

// clearFilter clears all active filters
func (m *listModel) clearFilter() {
	m.tagFilter = ""
}

// hasActiveFilter returns true if any filter is active
func (m *listModel) hasActiveFilter() bool {
	return m.tagFilter != ""
}

func (m listModel) Update(msg tea.Msg) (listModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Reserve space for border and footer
		m.list.SetSize(msg.Width-2, msg.Height-4)
		// Recalculate responsive columns
		m.cols = ui.CalculateResponsiveColumns(msg.Width, m.hasTags)
		m.updateDelegate()

	case beansLoadedMsg:
		sortBeans(msg.beans, m.config.StatusNames())
		items := make([]list.Item, len(msg.beans))
		// Check if any beans have tags
		m.hasTags = false
		for i, b := range msg.beans {
			items[i] = beanItem{bean: b, cfg: m.config}
			if len(b.Tags) > 0 {
				m.hasTags = true
			}
		}
		m.list.SetItems(items)
		// Calculate responsive columns based on hasTags and width
		m.cols = ui.CalculateResponsiveColumns(m.width, m.hasTags)
		m.updateDelegate()
		return m, nil

	case errMsg:
		m.err = msg.err
		return m, nil

	case tea.KeyMsg:
		if m.list.FilterState() != list.Filtering {
			switch msg.String() {
			case "enter":
				if item, ok := m.list.SelectedItem().(beanItem); ok {
					return m, func() tea.Msg {
						return selectBeanMsg{bean: item.bean}
					}
				}
			case "esc", "backspace":
				// If we have an active filter, clear it instead of quitting
				if m.hasActiveFilter() {
					return m, func() tea.Msg {
						return clearFilterMsg{}
					}
				}
			}
		}
	}

	// Always forward to the list component
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// updateDelegate updates the list delegate with current responsive columns
func (m *listModel) updateDelegate() {
	delegate := itemDelegate{
		cfg:     m.config,
		hasTags: m.hasTags,
		width:   m.width,
		cols:    m.cols,
	}
	m.list.SetDelegate(delegate)
}

func (m listModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress q to quit.", m.err)
	}

	if m.width == 0 {
		return "Loading..."
	}

	// Update title based on active filter
	if m.tagFilter != "" {
		m.list.Title = fmt.Sprintf("Beans [tag: %s]", m.tagFilter)
	} else {
		m.list.Title = "Beans"
	}

	// Simple bordered container
	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorMuted).
		Width(m.width - 2).
		Height(m.height - 4)

	content := border.Render(m.list.View())

	// Footer - show different help based on filter state
	var help string
	if m.hasActiveFilter() {
		help = helpKeyStyle.Render("enter") + " " + helpStyle.Render("view") + "  " +
			helpKeyStyle.Render("esc") + " " + helpStyle.Render("clear filter") + "  " +
			helpKeyStyle.Render("q") + " " + helpStyle.Render("quit")
	} else {
		help = helpKeyStyle.Render("enter") + " " + helpStyle.Render("view") + "  " +
			helpKeyStyle.Render("/") + " " + helpStyle.Render("filter") + "  " +
			helpKeyStyle.Render("g t") + " " + helpStyle.Render("filter by tag") + "  " +
			helpKeyStyle.Render("q") + " " + helpStyle.Render("quit")
	}

	return content + "\n" + help
}

// sortBeans sorts beans by status order
func sortBeans(beans []*bean.Bean, statusNames []string) {
	statusOrder := make(map[string]int)
	for i, s := range statusNames {
		statusOrder[s] = i
	}
	sort.Slice(beans, func(i, j int) bool {
		oi, oj := statusOrder[beans[i].Status], statusOrder[beans[j].Status]
		if oi != oj {
			return oi < oj
		}
		return strings.ToLower(beans[i].Title) < strings.ToLower(beans[j].Title)
	})
}
