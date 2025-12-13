package tui

import (
	"context"
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/config"
	"github.com/hmans/beans/internal/graph"
	"github.com/hmans/beans/internal/graph/model"
	"github.com/hmans/beans/internal/ui"
)

// beanItem wraps a Bean to implement list.Item, with tree context
type beanItem struct {
	bean       *bean.Bean
	cfg        *config.Config
	treePrefix string // tree prefix for rendering (e.g., "├─" or "  └─")
	matched    bool   // true if bean matched filter (vs. ancestor shown for context)
}

func (i beanItem) Title() string       { return i.bean.Title }
func (i beanItem) Description() string { return i.bean.ID + " · " + i.bean.Status }
func (i beanItem) FilterValue() string { return i.bean.Title + " " + i.bean.ID }

// itemDelegate handles rendering of list items
type itemDelegate struct {
	cfg        *config.Config
	hasTags    bool
	width      int
	cols       ui.ResponsiveColumns // cached responsive columns
	idColWidth int                  // ID column width (accounts for tree prefix)
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

	// Get colors from config
	colors := d.cfg.GetBeanColors(item.bean.Status, item.bean.Type, item.bean.Priority)

	// Calculate max title width using responsive columns
	idWidth := d.cols.ID
	if d.idColWidth > 0 {
		idWidth = d.idColWidth
	}
	baseWidth := idWidth + d.cols.Status + d.cols.Type + 4 // 4 for cursor + padding
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
			StatusColor:   colors.StatusColor,
			TypeColor:     colors.TypeColor,
			PriorityColor: colors.PriorityColor,
			Priority:      item.bean.Priority,
			IsArchive:     colors.IsArchive,
			MaxTitleWidth: maxTitleWidth,
			ShowCursor:    true,
			IsSelected:    index == m.Index(),
			Tags:          item.bean.Tags,
			ShowTags:      d.cols.ShowTags,
			TagsColWidth:  d.cols.Tags,
			MaxTags:       d.cols.MaxTags,
			TreePrefix:    item.treePrefix,
			Dimmed:        !item.matched,
			IDColWidth:    d.idColWidth,
		},
	)

	fmt.Fprint(w, str)
}

// listModel is the model for the bean list view
type listModel struct {
	list     list.Model
	resolver *graph.Resolver
	config   *config.Config
	width    int
	height   int
	err      error

	// Responsive column state
	hasTags    bool                 // whether any beans have tags
	cols       ui.ResponsiveColumns // calculated responsive columns
	idColWidth int                  // ID column width (accounts for tree depth)

	// Active filters
	tagFilter string // if set, only show beans with this tag
}

func newListModel(resolver *graph.Resolver, cfg *config.Config) listModel {
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
		list:     l,
		resolver: resolver,
		config:   cfg,
	}
}

// beansLoadedMsg is sent when beans are loaded
type beansLoadedMsg struct {
	items      []ui.FlatItem // flattened tree items
	idColWidth int           // calculated ID column width for tree
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
	// Build filter if tag filter is set
	var filter *model.BeanFilter
	if m.tagFilter != "" {
		filter = &model.BeanFilter{Tags: []string{m.tagFilter}}
	}

	// Query filtered beans
	filteredBeans, err := m.resolver.Query().Beans(context.Background(), filter)
	if err != nil {
		return errMsg{err}
	}

	// Query all beans for tree context (ancestors)
	allBeans, err := m.resolver.Query().Beans(context.Background(), nil)
	if err != nil {
		return errMsg{err}
	}

	// Sort function for tree building
	sortFn := func(beans []*bean.Bean) {
		bean.SortByStatusPriorityAndType(beans, m.config.StatusNames(), m.config.PriorityNames(), m.config.TypeNames())
	}

	// Build tree and flatten it
	tree := ui.BuildTree(filteredBeans, allBeans, sortFn)
	items := ui.FlattenTree(tree)

	// Calculate ID column width based on max ID length and tree depth
	maxIDLen := 0
	for _, b := range allBeans {
		if len(b.ID) > maxIDLen {
			maxIDLen = len(b.ID)
		}
	}
	maxDepth := ui.MaxTreeDepth(items)
	// ID column = base ID width + tree indent (2 chars per depth level for depth > 0)
	idColWidth := maxIDLen + 2 // base padding
	if maxDepth > 0 {
		idColWidth += maxDepth * 3 // 3 chars per depth level (├─ + space)
	}

	return beansLoadedMsg{items: items, idColWidth: idColWidth}
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
		items := make([]list.Item, len(msg.items))
		// Check if any beans have tags
		m.hasTags = false
		for i, flatItem := range msg.items {
			items[i] = beanItem{
				bean:       flatItem.Bean,
				cfg:        m.config,
				treePrefix: flatItem.TreePrefix,
				matched:    flatItem.Matched,
			}
			if len(flatItem.Bean.Tags) > 0 {
				m.hasTags = true
			}
		}
		m.list.SetItems(items)
		m.idColWidth = msg.idColWidth
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
			case "p":
				// Open parent picker for selected bean
				if item, ok := m.list.SelectedItem().(beanItem); ok {
					return m, func() tea.Msg {
						return openParentPickerMsg{
							beanID:        item.bean.ID,
							beanTitle:     item.bean.Title,
							beanType:      item.bean.Type,
							currentParent: item.bean.Parent,
						}
					}
				}
			case "s":
				// Open status picker for selected bean
				if item, ok := m.list.SelectedItem().(beanItem); ok {
					return m, func() tea.Msg {
						return openStatusPickerMsg{
							beanID:        item.bean.ID,
							beanTitle:     item.bean.Title,
							currentStatus: item.bean.Status,
						}
					}
				}
			case "t":
				// Open type picker for selected bean
				if item, ok := m.list.SelectedItem().(beanItem); ok {
					return m, func() tea.Msg {
						return openTypePickerMsg{
							beanID:      item.bean.ID,
							beanTitle:   item.bean.Title,
							currentType: item.bean.Type,
						}
					}
				}
			case "b":
				// Open blocking picker for selected bean
				if item, ok := m.list.SelectedItem().(beanItem); ok {
					return m, func() tea.Msg {
						return openBlockingPickerMsg{
							beanID:          item.bean.ID,
							beanTitle:       item.bean.Title,
							currentBlocking: item.bean.Blocking,
						}
					}
				}
			case "c":
				// Open create modal
				return m, func() tea.Msg {
					return openCreateModalMsg{}
				}
			case "e":
				// Open editor for selected bean
				if item, ok := m.list.SelectedItem().(beanItem); ok {
					return m, func() tea.Msg {
						return openEditorMsg{
							beanID:   item.bean.ID,
							beanPath: item.bean.Path,
						}
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
		cfg:        m.config,
		hasTags:    m.hasTags,
		width:      m.width,
		cols:       m.cols,
		idColWidth: m.idColWidth,
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
			helpKeyStyle.Render("c") + " " + helpStyle.Render("create") + "  " +
			helpKeyStyle.Render("e") + " " + helpStyle.Render("edit") + "  " +
			helpKeyStyle.Render("s") + " " + helpStyle.Render("status") + "  " +
			helpKeyStyle.Render("t") + " " + helpStyle.Render("type") + "  " +
			helpKeyStyle.Render("p") + " " + helpStyle.Render("parent") + "  " +
			helpKeyStyle.Render("b") + " " + helpStyle.Render("blocking") + "  " +
			helpKeyStyle.Render("esc") + " " + helpStyle.Render("clear filter") + "  " +
			helpKeyStyle.Render("q") + " " + helpStyle.Render("quit")
	} else {
		help = helpKeyStyle.Render("enter") + " " + helpStyle.Render("view") + "  " +
			helpKeyStyle.Render("c") + " " + helpStyle.Render("create") + "  " +
			helpKeyStyle.Render("e") + " " + helpStyle.Render("edit") + "  " +
			helpKeyStyle.Render("s") + " " + helpStyle.Render("status") + "  " +
			helpKeyStyle.Render("t") + " " + helpStyle.Render("type") + "  " +
			helpKeyStyle.Render("p") + " " + helpStyle.Render("parent") + "  " +
			helpKeyStyle.Render("b") + " " + helpStyle.Render("blocking") + "  " +
			helpKeyStyle.Render("/") + " " + helpStyle.Render("filter") + "  " +
			helpKeyStyle.Render("q") + " " + helpStyle.Render("quit")
	}

	return content + "\n" + help
}

