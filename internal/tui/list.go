package tui

import (
	"context"
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hmans/beans/pkg/bean"
	"github.com/hmans/beans/pkg/config"
	"github.com/hmans/beans/pkg/beangraph"
	"github.com/hmans/beans/pkg/beangraph/model"
	"github.com/hmans/beans/internal/ui"
)

// beanItem wraps a Bean to implement list.Item, with tree context
type beanItem struct {
	bean            *bean.Bean
	cfg             *config.Config
	treePrefix      string // tree prefix for rendering (e.g., "├─" or "  └─")
	matched         bool   // true if bean matched filter (vs. ancestor shown for context)
	implicitStatus string // implicit terminal status from an ancestor, if any
	blocked        bool   // open bean with an active blocker, directly or via an ancestor
}

func (i beanItem) Title() string       { return i.bean.Title }
func (i beanItem) Description() string { return i.bean.ID + " · " + i.bean.Status }
func (i beanItem) FilterValue() string { return i.bean.Title + " " + i.bean.ID }

// showsBlocked reports whether the bean carries the blocked mark. A closed
// ancestor takes precedence, because its blockers no longer matter.
func (i beanItem) showsBlocked() bool { return i.blocked && i.implicitStatus == "" }

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
			StatusColor:     colors.StatusColor,
			TypeColor:       colors.TypeColor,
			PriorityColor:   colors.PriorityColor,
			Priority:        item.bean.Priority,
			IsArchive:       colors.IsArchive,
			MaxTitleWidth:   maxTitleWidth,
			ShowCursor:      true,
			IsSelected:      index == m.Index(),
			IsMarked:        isMarked,
			Tags:            item.bean.Tags,
			ShowTags:        d.cols.ShowTags,
			TagsColWidth:    d.cols.Tags,
			MaxTags:         d.cols.MaxTags,
			TreePrefix:      item.treePrefix,
			Dimmed:          !item.matched,
			IDColWidth:      d.idColWidth,
			TypeColWidth:    d.cols.Type,
			StatusColWidth:  d.cols.Status,
			ImplicitStatus: item.implicitStatus,
			Blocked:        item.blocked,
		},
	)

	fmt.Fprint(w, str)
}

// listModel is the model for the bean list view
type listModel struct {
	list     list.Model
	resolver *beangraph.CoreResolver
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

	// viewMode is not a filter, so clearFilter leaves it alone.
	viewMode viewMode

	// Multi-select state
	selectedBeans map[string]bool // IDs of beans marked for multi-edit

	// Status message to display in footer
	statusMessage string
}

// viewMode selects which beans the list shows; each mode narrows the previous one.
type viewMode int

const (
	viewAll viewMode = iota
	// viewActive hides beans with an archive status and their descendants.
	viewActive
	// viewUnblocked additionally hides blocked beans.
	viewUnblocked
	// viewReady shows what `beans list --ready` shows.
	viewReady
	viewModeCount
)

func (v viewMode) String() string {
	return [...]string{"all", "active", "unblocked", "ready"}[v]
}

func (v viewMode) next() viewMode {
	return (v + 1) % viewModeCount
}

func newListModel(resolver *beangraph.CoreResolver, cfg *config.Config) listModel {
	selectedBeans := make(map[string]bool)
	delegate := itemDelegate{cfg: cfg, selectedBeans: &selectedBeans}

	l := list.New([]list.Item{}, delegate, 0, 0)
	// The title and the filter input go into the pane's top border.
	l.SetShowTitle(false)
	l.SetShowFilter(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)
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
	items      []ui.FlatItem   // flattened tree items
	idColWidth int             // calculated ID column width for tree
	blocked    map[string]bool // IDs of open beans with an active blocker
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
	if m.viewMode >= viewActive {
		if filter == nil {
			filter = &model.BeanFilter{}
		}
		for _, s := range m.config.StatusNames() {
			if m.config.IsArchiveStatus(s) {
				filter.ExcludeStatus = append(filter.ExcludeStatus, s)
			}
		}
	}
	if m.viewMode >= viewUnblocked {
		isBlocked := false
		filter.IsBlocked = &isBlocked
	}
	if m.viewMode >= viewReady {
		beangraph.AddReadyFilter(filter)
	}

	// Query filtered beans
	filteredBeans, err := m.resolver.Beans(context.Background(), filter)
	if err != nil {
		return errMsg{err}
	}

	// Query all beans for tree context (ancestors)
	allBeans, err := m.resolver.Beans(context.Background(), nil)
	if err != nil {
		return errMsg{err}
	}

	// Sort function for tree building
	sortFn := func(beans []*bean.Bean) {
		bean.SortByStatusPriorityAndType(beans, m.config.StatusNames(), m.config.PriorityNames(), m.config.TypeNames())
	}

	// Pre-compute implicit statuses for all beans
	implicitStatuses := make(map[string]string, len(allBeans))
	for _, b := range allBeans {
		if status, _ := m.resolver.Core.ClosedAncestor(b.ID); status != "" {
			implicitStatuses[b.ID] = status
		}
	}

	blocked := make(map[string]bool)
	for _, b := range allBeans {
		if !m.config.IsArchiveStatus(b.Status) && m.resolver.Core.IsBlocked(b.ID) {
			blocked[b.ID] = true
		}
	}

	if m.viewMode >= viewActive {
		open := filteredBeans[:0]
		for _, b := range filteredBeans {
			if _, closed := implicitStatuses[b.ID]; !closed {
				open = append(open, b)
			}
		}
		filteredBeans = open
	}

	// Build tree and flatten it
	tree := ui.BuildTree(filteredBeans, allBeans, sortFn, implicitStatuses)
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

	return beansLoadedMsg{items: items, idColWidth: idColWidth, blocked: blocked}
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
		m.list.SetSize(msg.Width-paneBorders, msg.Height-footerHeight-paneBorders-listBottomPadding)
		// Recalculate responsive columns
		m.cols = ui.CalculateResponsiveColumns(msg.Width, m.hasTags)
		m.updateDelegate()

	case beansLoadedMsg:
		items := make([]list.Item, len(msg.items))
		// Check if any beans have tags
		m.hasTags = false
		for i, flatItem := range msg.items {
			items[i] = beanItem{
				bean:            flatItem.Bean,
				cfg:             m.config,
				treePrefix:      flatItem.TreePrefix,
				matched:         flatItem.Matched,
				implicitStatus: flatItem.ImplicitStatus,
				blocked:        msg.blocked[flatItem.Bean.ID],
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
			case "h":
				m.viewMode = m.viewMode.next()
				return m, m.loadBeans
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
				return cursorChangedMsg{beanID: item.bean.ID, blocked: item.showsBlocked()}
			})
		}
	}

	return m, tea.Batch(cmds...)
}

// itemIndexAt returns the index of the item rendered at pane-relative
// position (x, y), for a list pane rendered by viewContent with the given
// outer width and inner height.
func (m listModel) itemIndexAt(x, y, paneWidth, innerHeight int) (int, bool) {
	if x < paneBorder || x >= paneWidth-paneBorder {
		return 0, false
	}

	// Paginate a copy at the rendered size, so the page matches the screen.
	l := m.list
	l.SetSize(paneWidth-paneBorders, innerHeight)

	row := y - paneBorder
	visible := len(l.VisibleItems())
	if row < 0 || row >= l.Paginator.ItemsOnPage(visible) {
		return 0, false
	}
	return l.Paginator.Page*l.Paginator.PerPage + row, true
}

// selectAt moves the cursor to the item at pane-relative position (x, y).
func (m listModel) selectAt(x, y, paneWidth, innerHeight int) (listModel, tea.Cmd) {
	index, ok := m.itemIndexAt(x, y, paneWidth, innerHeight)
	if !ok || index == m.list.Index() {
		return m, nil
	}
	m.list.Select(index)
	item, ok := m.list.SelectedItem().(beanItem)
	if !ok {
		return m, nil
	}
	return m, func() tea.Msg { return cursorChangedMsg{beanID: item.bean.ID, blocked: item.showsBlocked()} }
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

	return m.viewContent(m.height-footerHeight-paneBorders-listBottomPadding) + "\n" + m.Footer()
}

func (m listModel) title() string {
	title := "Beans"
	if m.viewMode != viewAll {
		title += " (" + m.viewMode.String() + ")"
	}
	if m.tagFilter != "" {
		title += fmt.Sprintf(" [tag: %s]", m.tagFilter)
	}
	return title
}

// viewContent renders just the bordered list without footer.
// innerHeight is the content height inside the border (not including border lines).
func (m listModel) viewContent(innerHeight int) string {
	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorMuted).
		Width(m.width - paneBorders).
		Height(innerHeight)

	// Clip the content so an over-wide row cannot wrap and grow the pane.
	content := lipgloss.NewStyle().MaxWidth(m.width - paneBorders).MaxHeight(innerHeight).Render(m.list.View())
	return withBorderTitle(border.Render(content), m.borderTitle())
}

// borderTitle is the filter input while the user types a filter, else the title.
func (m listModel) borderTitle() string {
	if m.list.FilterState() == list.Filtering {
		input := m.list.FilterInput
		input.Width = 0 // the list sizes it to the pane, which would pad it with spaces
		return input.View()
	}
	return listTitleStyle.Render(m.title())
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
			helpKeyStyle.Render("h") + " " + helpStyle.Render(m.viewModeHelp()) + "  " +
			helpKeyStyle.Render("v") + " " + helpStyle.Render("layout") + "  " +
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
			helpKeyStyle.Render("h") + " " + helpStyle.Render(m.viewModeHelp()) + "  " +
			helpKeyStyle.Render("/") + " " + helpStyle.Render("filter") + "  " +
			helpKeyStyle.Render("v") + " " + helpStyle.Render("layout") + "  " +
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

func (m listModel) viewModeHelp() string {
	return "show " + m.viewMode.next().String()
}

// ViewConstrained renders the list constrained to the given width and height.
// Used for the left pane in two-column mode. Returns only the content without footer.
// The output will be exactly `height` lines tall.
func (m listModel) ViewConstrained(width, height int) string {
	// Temporarily set constrained dimensions
	m.width = width
	m.height = height

	innerHeight := height - paneBorders
	m.list.SetSize(width-paneBorders, innerHeight)

	// Recalculate columns for constrained width
	m.cols = ui.CalculateResponsiveColumns(width, m.hasTags)
	m.updateDelegate()

	return m.viewContent(innerHeight)
}

