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
	cfg           *config.Config
	hasTags       bool
	width         int
	cols          ui.ResponsiveColumns // cached responsive columns
	idColWidth    int                  // ID column width (accounts for tree prefix)
	selectedBeans *map[string]bool     // pointer to marked beans for multi-select
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

	// Check if bean is marked for multi-select
	var isMarked bool
	if d.selectedBeans != nil {
		isMarked = (*d.selectedBeans)[item.bean.ID]
	}

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
			IsMarked:      isMarked,
			Tags:          item.bean.Tags,
			ShowTags:      d.cols.ShowTags,
			TagsColWidth:  d.cols.Tags,
			MaxTags:       d.cols.MaxTags,
			TreePrefix:    item.treePrefix,
			Dimmed:        !item.matched,
			IDColWidth:    d.idColWidth,
			UseFullNames:  d.cols.UseFullTypeStatus,
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

	// Multi-select state
	selectedBeans map[string]bool // IDs of beans marked for multi-edit

	// Status message to display in footer
	statusMessage string
}

func newListModel(resolver *graph.Resolver, cfg *config.Config) listModel {
	selectedBeans := make(map[string]bool)
	delegate := itemDelegate{cfg: cfg, selectedBeans: &selectedBeans}

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.KeyMap.Quit.SetEnabled(false)      // Only q quits, not escape
	l.KeyMap.ForceQuit.SetEnabled(false) // Disable force quit too
	l.Title = "Beans"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)
	l.Styles.Title = listTitleStyle
	l.Styles.TitleBar = lipgloss.NewStyle().Padding(0, 0, 1, 1)
	l.Styles.FilterPrompt = lipgloss.NewStyle().Foreground(ui.ColorPrimary)
	l.Styles.FilterCursor = lipgloss.NewStyle().Foreground(ui.ColorPrimary)

	return listModel{
		list:          l,
		resolver:      resolver,
		config:        cfg,
		selectedBeans: selectedBeans,
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
	// ID column = base ID width + tree indent (3 chars per depth level)
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
	var cmds []tea.Cmd

	// Track cursor position before update
	prevIndex := m.list.Index()

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
			case " ":
				// Toggle selection for multi-select, then move to next item
				if item, ok := m.list.SelectedItem().(beanItem); ok {
					if m.selectedBeans[item.bean.ID] {
						delete(m.selectedBeans, item.bean.ID)
					} else {
						m.selectedBeans[item.bean.ID] = true
					}
					m.list.CursorDown()
				}
				return m, nil
			case "enter":
				if item, ok := m.list.SelectedItem().(beanItem); ok {
					return m, func() tea.Msg {
						return selectBeanMsg{bean: item.bean}
					}
				}
			case "p":
				// Open parent picker for selected bean(s)
				if len(m.selectedBeans) > 0 {
					// Multi-select mode
					ids := make([]string, 0, len(m.selectedBeans))
					types := make([]string, 0, len(m.selectedBeans))
					for id := range m.selectedBeans {
						ids = append(ids, id)
						// Find the bean to get its type
						for _, item := range m.list.Items() {
							if bi, ok := item.(beanItem); ok && bi.bean.ID == id {
								types = append(types, bi.bean.Type)
								break
							}
						}
					}
					return m, func() tea.Msg {
						return openParentPickerMsg{
							beanIDs:   ids,
							beanTitle: fmt.Sprintf("%d selected beans", len(ids)),
							beanTypes: types,
						}
					}
				} else if item, ok := m.list.SelectedItem().(beanItem); ok {
					return m, func() tea.Msg {
						return openParentPickerMsg{
							beanIDs:       []string{item.bean.ID},
							beanTitle:     item.bean.Title,
							beanTypes:     []string{item.bean.Type},
							currentParent: item.bean.Parent,
						}
					}
				}
			case "s":
				// Open status picker for selected bean(s)
				if len(m.selectedBeans) > 0 {
					// Multi-select mode
					ids := make([]string, 0, len(m.selectedBeans))
					for id := range m.selectedBeans {
						ids = append(ids, id)
					}
					return m, func() tea.Msg {
						return openStatusPickerMsg{
							beanIDs:   ids,
							beanTitle: fmt.Sprintf("%d selected beans", len(ids)),
						}
					}
				} else if item, ok := m.list.SelectedItem().(beanItem); ok {
					return m, func() tea.Msg {
						return openStatusPickerMsg{
							beanIDs:       []string{item.bean.ID},
							beanTitle:     item.bean.Title,
							currentStatus: item.bean.Status,
						}
					}
				}
			case "t":
				// Open type picker for selected bean(s)
				if len(m.selectedBeans) > 0 {
					// Multi-select mode
					ids := make([]string, 0, len(m.selectedBeans))
					for id := range m.selectedBeans {
						ids = append(ids, id)
					}
					return m, func() tea.Msg {
						return openTypePickerMsg{
							beanIDs:   ids,
							beanTitle: fmt.Sprintf("%d selected beans", len(ids)),
						}
					}
				} else if item, ok := m.list.SelectedItem().(beanItem); ok {
					return m, func() tea.Msg {
						return openTypePickerMsg{
							beanIDs:     []string{item.bean.ID},
							beanTitle:   item.bean.Title,
							currentType: item.bean.Type,
						}
					}
				}
			case "P":
				// Open priority picker for selected bean(s)
				if len(m.selectedBeans) > 0 {
					// Multi-select mode
					ids := make([]string, 0, len(m.selectedBeans))
					for id := range m.selectedBeans {
						ids = append(ids, id)
					}
					return m, func() tea.Msg {
						return openPriorityPickerMsg{
							beanIDs:   ids,
							beanTitle: fmt.Sprintf("%d selected beans", len(ids)),
						}
					}
				} else if item, ok := m.list.SelectedItem().(beanItem); ok {
					return m, func() tea.Msg {
						return openPriorityPickerMsg{
							beanIDs:         []string{item.bean.ID},
							beanTitle:       item.bean.Title,
							currentPriority: item.bean.Priority,
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
			case "y":
				// Copy bean ID(s) to clipboard
				if len(m.selectedBeans) > 0 {
					// Multi-select mode: copy all selected IDs
					ids := make([]string, 0, len(m.selectedBeans))
					for id := range m.selectedBeans {
						ids = append(ids, id)
					}
					return m, func() tea.Msg {
						return copyBeanIDMsg{ids: ids}
					}
				} else if item, ok := m.list.SelectedItem().(beanItem); ok {
					// Single bean mode
					return m, func() tea.Msg {
						return copyBeanIDMsg{ids: []string{item.bean.ID}}
					}
				}
			case "esc", "backspace":
				// First clear selection if any beans are selected
				if len(m.selectedBeans) > 0 {
					clear(m.selectedBeans)
					return m, nil
				}
				// Then clear active filter if any
				if m.hasActiveFilter() {
					return m, func() tea.Msg {
						return clearFilterMsg{}
					}
				}
				// Don't forward to bubbles list (would quit)
				return m, nil
			}
		}
	}

	// Always forward to the list component
	m.list, cmd = m.list.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	// Check if cursor moved and emit message
	if m.list.Index() != prevIndex {
		if item, ok := m.list.SelectedItem().(beanItem); ok {
			cmds = append(cmds, func() tea.Msg {
				return cursorChangedMsg{beanID: item.bean.ID}
			})
		}
	}

	return m, tea.Batch(cmds...)
}

// updateDelegate updates the list delegate with current responsive columns
func (m *listModel) updateDelegate() {
	delegate := itemDelegate{
		cfg:           m.config,
		hasTags:       m.hasTags,
		width:         m.width,
		cols:          m.cols,
		idColWidth:    m.idColWidth,
		selectedBeans: &m.selectedBeans,
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

	// Inner height: total height minus border (2) minus footer (1) minus padding (1)
	return m.viewContent(m.height-4, true) + "\n" + m.Footer()
}

// viewContent renders just the bordered list without footer.
// innerHeight is the content height inside the border (not including border lines).
// focused determines the border color (primary when focused, muted when not).
func (m listModel) viewContent(innerHeight int, focused bool) string {
	borderColor := ui.ColorMuted
	if focused {
		borderColor = ui.ColorPrimary
	}
	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(m.width - 2).
		Height(innerHeight)

	return border.Render(m.list.View())
}

// Footer renders the help/status footer for the list view.
func (m listModel) Footer() string {
	var help string

	// Show selection count if any beans are selected
	var selectionPrefix string
	if len(m.selectedBeans) > 0 {
		selectionStyle := lipgloss.NewStyle().Foreground(ui.ColorWarning).Bold(true)
		selectionPrefix = selectionStyle.Render(fmt.Sprintf("(%d selected) ", len(m.selectedBeans)))
	}

	if len(m.selectedBeans) > 0 {
		// When beans are selected, show esc to clear selection
		help = helpKeyStyle.Render("space") + " " + helpStyle.Render("toggle") + "  " +
			helpKeyStyle.Render("P") + " " + helpStyle.Render("priority") + "  " +
			helpKeyStyle.Render("s") + " " + helpStyle.Render("status") + "  " +
			helpKeyStyle.Render("t") + " " + helpStyle.Render("type") + "  " +
			helpKeyStyle.Render("y") + " " + helpStyle.Render("copy id") + "  " +
			helpKeyStyle.Render("esc") + " " + helpStyle.Render("clear selection") + "  " +
			helpKeyStyle.Render("?") + " " + helpStyle.Render("help") + "  " +
			helpKeyStyle.Render("q") + " " + helpStyle.Render("quit")
	} else if m.hasActiveFilter() {
		help = helpKeyStyle.Render("space") + " " + helpStyle.Render("select") + "  " +
			helpKeyStyle.Render("enter") + " " + helpStyle.Render("view") + "  " +
			helpKeyStyle.Render("b") + " " + helpStyle.Render("blocking") + "  " +
			helpKeyStyle.Render("c") + " " + helpStyle.Render("create") + "  " +
			helpKeyStyle.Render("e") + " " + helpStyle.Render("edit") + "  " +
			helpKeyStyle.Render("p") + " " + helpStyle.Render("parent") + "  " +
			helpKeyStyle.Render("P") + " " + helpStyle.Render("priority") + "  " +
			helpKeyStyle.Render("s") + " " + helpStyle.Render("status") + "  " +
			helpKeyStyle.Render("t") + " " + helpStyle.Render("type") + "  " +
			helpKeyStyle.Render("y") + " " + helpStyle.Render("copy id") + "  " +
			helpKeyStyle.Render("esc") + " " + helpStyle.Render("clear filter") + "  " +
			helpKeyStyle.Render("?") + " " + helpStyle.Render("help") + "  " +
			helpKeyStyle.Render("q") + " " + helpStyle.Render("quit")
	} else {
		help = helpKeyStyle.Render("space") + " " + helpStyle.Render("select") + "  " +
			helpKeyStyle.Render("enter") + " " + helpStyle.Render("view") + "  " +
			helpKeyStyle.Render("b") + " " + helpStyle.Render("blocking") + "  " +
			helpKeyStyle.Render("c") + " " + helpStyle.Render("create") + "  " +
			helpKeyStyle.Render("e") + " " + helpStyle.Render("edit") + "  " +
			helpKeyStyle.Render("p") + " " + helpStyle.Render("parent") + "  " +
			helpKeyStyle.Render("P") + " " + helpStyle.Render("priority") + "  " +
			helpKeyStyle.Render("s") + " " + helpStyle.Render("status") + "  " +
			helpKeyStyle.Render("t") + " " + helpStyle.Render("type") + "  " +
			helpKeyStyle.Render("y") + " " + helpStyle.Render("copy id") + "  " +
			helpKeyStyle.Render("/") + " " + helpStyle.Render("filter") + "  " +
			helpKeyStyle.Render("?") + " " + helpStyle.Render("help") + "  " +
			helpKeyStyle.Render("q") + " " + helpStyle.Render("quit")
	}

	// Show status message if present, otherwise show help
	footer := selectionPrefix
	if m.statusMessage != "" {
		statusStyle := lipgloss.NewStyle().Foreground(ui.ColorSuccess).Bold(true)
		footer += statusStyle.Render(m.statusMessage)
	} else {
		footer += help
	}

	return footer
}

// ViewConstrained renders the list constrained to the given width and height.
// Used for the left pane in two-column mode. Returns only the content without footer.
// The output will be exactly `height` lines tall.
func (m listModel) ViewConstrained(width, height int, focused bool) string {
	// Temporarily set constrained dimensions
	m.width = width
	m.height = height

	// Inner height for border content (height minus 2 for top/bottom border)
	innerHeight := height - 2
	m.list.SetSize(width-2, innerHeight)

	// Recalculate columns for constrained width
	m.cols = ui.CalculateResponsiveColumns(width, m.hasTags)
	m.updateDelegate()

	// Update title based on active filter
	if m.tagFilter != "" {
		m.list.Title = fmt.Sprintf("Beans [tag: %s]", m.tagFilter)
	} else {
		m.list.Title = "Beans"
	}

	return m.viewContent(innerHeight, focused)
}

