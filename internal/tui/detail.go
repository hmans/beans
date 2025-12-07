package tui

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"hmans.dev/beans/internal/bean"
	"hmans.dev/beans/internal/beancore"
	"hmans.dev/beans/internal/config"
	"hmans.dev/beans/internal/ui"
)

// Cached glamour renderer - initialized once per width
var (
	glamourRenderer     *glamour.TermRenderer
	glamourRendererOnce sync.Once
)

func getGlamourRenderer() *glamour.TermRenderer {
	glamourRendererOnce.Do(func() {
		var err error
		glamourRenderer, err = glamour.NewTermRenderer(glamour.WithAutoStyle())
		if err != nil {
			glamourRenderer = nil
		}
	})
	return glamourRenderer
}

// backToListMsg signals navigation back to the list
type backToListMsg struct{}

// resolvedLink represents a link with the target bean resolved
type resolvedLink struct {
	linkType string
	bean     *bean.Bean
	incoming bool // true if another bean links TO this one
}

// linkItem wraps a resolvedLink to implement list.Item
type linkItem struct {
	link   resolvedLink
	cfg    *config.Config
	width  int
	cols   ui.ResponsiveColumns
	label  string // pre-computed label like "Blocks:" or "Blocked by:"
}

func (i linkItem) Title() string       { return i.link.bean.Title }
func (i linkItem) Description() string { return i.link.bean.ID }
func (i linkItem) FilterValue() string { return i.link.bean.Title + " " + i.link.bean.ID + " " + i.label }

// linkDelegate handles rendering of link list items
type linkDelegate struct {
	cfg   *config.Config
	width int
	cols  ui.ResponsiveColumns
}

func (d linkDelegate) Height() int                             { return 1 }
func (d linkDelegate) Spacing() int                            { return 0 }
func (d linkDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d linkDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(linkItem)
	if !ok {
		return
	}

	link := item.link

	// Cursor indicator
	cursor := "  "
	if index == m.Index() {
		cursor = ui.Primary.Render("▸ ")
	}

	// Format the link type label
	labelCol := lipgloss.NewStyle().Width(12).Render(ui.Muted.Render(item.label + ":"))

	// Get colors from config
	colors := d.cfg.GetBeanColors(link.bean.Status, link.bean.Type)

	// Calculate max title width using responsive columns
	baseWidth := d.cols.ID + d.cols.Status + d.cols.Type + 12 + 4 // label + cursor + padding
	if d.cols.ShowTags {
		baseWidth += d.cols.Tags
	}
	maxTitleWidth := max(10, d.width-baseWidth-8) // 8 for border padding

	// Use shared bean row rendering (without cursor, we handle it separately)
	row := ui.RenderBeanRow(
		link.bean.ID,
		link.bean.Status,
		link.bean.Type,
		link.bean.Title,
		ui.BeanRowConfig{
			StatusColor:   colors.StatusColor,
			TypeColor:     colors.TypeColor,
			IsArchive:     colors.IsArchive,
			MaxTitleWidth: maxTitleWidth,
			ShowCursor:    false,
			IsSelected:    false,
			Tags:          link.bean.Tags,
			ShowTags:      d.cols.ShowTags,
			TagsColWidth:  d.cols.Tags,
			MaxTags:       d.cols.MaxTags,
		},
	)

	fmt.Fprint(w, cursor+labelCol+row)
}

// detailModel displays a single bean's details
type detailModel struct {
	viewport    viewport.Model
	bean        *bean.Bean
	core        *beancore.Core
	config      *config.Config
	width       int
	height      int
	ready       bool
	links       []resolvedLink       // combined outgoing + incoming links
	linkList    list.Model           // list component for links (supports filtering)
	linksActive bool                 // true = links section focused
	cols        ui.ResponsiveColumns // responsive column widths for links
}

func newDetailModel(b *bean.Bean, core *beancore.Core, cfg *config.Config, width, height int) detailModel {
	m := detailModel{
		bean:        b,
		core:        core,
		config:      cfg,
		width:       width,
		height:      height,
		ready:       true,
		linksActive: false,
	}

	// Resolve all links
	m.links = m.resolveAllLinks()

	// Check if any linked beans have tags
	hasTags := false
	for _, link := range m.links {
		if len(link.bean.Tags) > 0 {
			hasTags = true
			break
		}
	}

	// Calculate responsive columns for links section
	// Account for the label column (12 chars) + cursor (2 chars) + border padding
	linkAreaWidth := width - 12 - 2 - 8
	m.cols = ui.CalculateResponsiveColumns(linkAreaWidth, hasTags)

	// Initialize link list with items
	m.linkList = m.createLinkList()

	// If there are links, select first one and focus links by default
	if len(m.links) > 0 {
		m.linksActive = true
	}

	// Calculate header height dynamically
	headerHeight := m.calculateHeaderHeight()
	footerHeight := 2
	vpWidth := width - 4
	vpHeight := height - headerHeight - footerHeight

	m.viewport = viewport.New(vpWidth, vpHeight)
	m.viewport.SetContent(m.renderBody(vpWidth))

	return m
}

// createLinkList creates a new list.Model for the links
func (m detailModel) createLinkList() list.Model {
	delegate := linkDelegate{
		cfg:   m.config,
		width: m.width,
		cols:  m.cols,
	}

	// Convert links to list items
	items := make([]list.Item, len(m.links))
	for i, link := range m.links {
		items[i] = linkItem{
			link:  link,
			cfg:   m.config,
			width: m.width,
			cols:  m.cols,
			label: m.formatLinkLabel(link.linkType, link.incoming),
		}
	}

	// Calculate list height: show all links up to 1/3 of screen height
	// Add 2 for the title row and padding
	maxHeight := max(3, m.height/3)
	listHeight := min(len(m.links), maxHeight) + 2

	l := list.New(items, delegate, m.width-8, listHeight)
	l.Title = "Linked Beans"
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.SetFilteringEnabled(true)

	// Style the title bar similar to the detail header title (badge style) but with different color
	l.Styles.Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#fff")).
		Background(ui.ColorBlue).
		Padding(0, 1)
	l.Styles.TitleBar = lipgloss.NewStyle().Padding(0, 0, 0, 1) // Left padding to align with header title
	l.Styles.FilterPrompt = lipgloss.NewStyle().Foreground(ui.ColorPrimary)
	l.Styles.FilterCursor = lipgloss.NewStyle().Foreground(ui.ColorPrimary)
	l.Styles.NoItems = lipgloss.NewStyle()

	return l
}

func (m detailModel) Init() tea.Cmd {
	return nil
}

func (m detailModel) Update(msg tea.Msg) (detailModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Recalculate responsive columns for links
		hasTags := false
		for _, link := range m.links {
			if len(link.bean.Tags) > 0 {
				hasTags = true
				break
			}
		}
		linkAreaWidth := msg.Width - 12 - 2 - 8
		m.cols = ui.CalculateResponsiveColumns(linkAreaWidth, hasTags)

		// Update link list delegate with new dimensions
		m.updateLinkListDelegate()

		// Update link list size: show all links up to 1/3 of screen height
		// Add 2 for the title row and padding
		maxHeight := max(3, msg.Height/3)
		listHeight := min(len(m.links), maxHeight) + 2
		m.linkList.SetSize(msg.Width-8, listHeight)

		headerHeight := m.calculateHeaderHeight()
		footerHeight := 2
		vpWidth := msg.Width - 4
		vpHeight := msg.Height - headerHeight - footerHeight

		// Ensure vpHeight doesn't go negative
		if vpHeight < 1 {
			vpHeight = 1
		}

		if !m.ready {
			m.viewport = viewport.New(vpWidth, vpHeight)
			m.viewport.SetContent(m.renderBody(vpWidth))
			m.ready = true
		} else {
			m.viewport.Width = vpWidth
			m.viewport.Height = vpHeight
			m.viewport.SetContent(m.renderBody(vpWidth))
		}

	case tea.KeyMsg:
		// If links list is filtering, let it handle all keys except quit
		if m.linksActive && m.linkList.FilterState() == list.Filtering {
			m.linkList, cmd = m.linkList.Update(msg)
			return m, cmd
		}

		switch msg.String() {
		case "esc", "backspace":
			return m, func() tea.Msg {
				return backToListMsg{}
			}

		case "tab":
			// Toggle focus between links and body
			if len(m.links) > 0 {
				m.linksActive = !m.linksActive
			}
			return m, nil

		case "enter":
			// Navigate to selected link
			if m.linksActive {
				if item, ok := m.linkList.SelectedItem().(linkItem); ok {
					targetBean := item.link.bean
					return m, func() tea.Msg {
						return selectBeanMsg{bean: targetBean}
					}
				}
			}
		}
	}

	// Forward updates to the appropriate component
	if m.linksActive && len(m.links) > 0 {
		m.linkList, cmd = m.linkList.Update(msg)
		cmds = append(cmds, cmd)
	} else {
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// updateLinkListDelegate updates the link list delegate with current dimensions
func (m *detailModel) updateLinkListDelegate() {
	delegate := linkDelegate{
		cfg:   m.config,
		width: m.width,
		cols:  m.cols,
	}
	m.linkList.SetDelegate(delegate)
}

func (m detailModel) View() string {
	if !m.ready {
		return "Loading..."
	}

	// Header (bean info only, no links)
	header := m.renderHeader()

	// Links section (if any)
	var linksSection string
	if len(m.links) > 0 {
		linksBorderColor := ui.ColorMuted
		if m.linksActive {
			linksBorderColor = ui.ColorPrimary
		}
		linksBorder := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(linksBorderColor).
			Width(m.width - 4)
		linksSection = linksBorder.Render(m.linkList.View()) + "\n"
	}

	// Body
	bodyBorderColor := ui.ColorMuted
	if !m.linksActive {
		bodyBorderColor = ui.ColorPrimary
	}
	bodyBorder := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(bodyBorderColor).
		Width(m.width - 4)
	body := bodyBorder.Render(m.viewport.View())

	// Footer
	scrollPct := int(m.viewport.ScrollPercent() * 100)
	footer := helpStyle.Render(fmt.Sprintf("%d%%", scrollPct)) + "  "
	if len(m.links) > 0 {
		footer += helpKeyStyle.Render("tab") + " " + helpStyle.Render("switch") + "  "
		if m.linksActive {
			footer += helpKeyStyle.Render("/") + " " + helpStyle.Render("filter") + "  "
		}
		footer += helpKeyStyle.Render("enter") + " " + helpStyle.Render("go to") + "  "
	}
	footer += helpKeyStyle.Render("j/k") + " " + helpStyle.Render("scroll") + "  " +
		helpKeyStyle.Render("esc") + " " + helpStyle.Render("back") + "  " +
		helpKeyStyle.Render("q") + " " + helpStyle.Render("quit")

	return header + "\n" + linksSection + body + "\n" + footer
}

func (m detailModel) calculateHeaderHeight() int {
	// Base: title line + ID/status line + borders/padding = ~6
	baseHeight := 6

	// Add height for links section (separate bordered box)
	if len(m.links) > 0 {
		// Links list height + borders (matches createLinkList calculation)
		// +2 for title row and padding, +3 for borders and spacing
		maxHeight := max(3, m.height/3)
		listHeight := min(len(m.links), maxHeight) + 2
		baseHeight += listHeight + 3
	}

	return baseHeight
}

func (m detailModel) renderHeader() string {
	// Title
	title := detailTitleStyle.Render(m.bean.Title)

	// ID
	id := ui.ID.Render(m.bean.ID)

	// Status badge
	statusCfg := m.config.GetStatus(m.bean.Status)
	statusColor := "gray"
	if statusCfg != nil {
		statusColor = statusCfg.Color
	}
	isArchive := m.config.IsArchiveStatus(m.bean.Status)
	status := ui.RenderStatusWithColor(m.bean.Status, statusColor, isArchive)

	// Build header content
	var headerContent strings.Builder
	headerContent.WriteString(title)
	headerContent.WriteString("\n")
	headerContent.WriteString(id + "  " + status)

	// Add tags if present
	if len(m.bean.Tags) > 0 {
		headerContent.WriteString("  ")
		headerContent.WriteString(ui.RenderTags(m.bean.Tags))
	}

	// Header box style - always muted border (not focused, links section is separate)
	headerBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorMuted).
		Padding(0, 1).
		Width(m.width - 4)

	return headerBox.Render(headerContent.String())
}

// formatLinkLabel returns a human-readable label for the link type
func (m detailModel) formatLinkLabel(linkType string, incoming bool) string {
	if incoming {
		switch linkType {
		case "blocks":
			return "Blocked by"
		case "parent":
			return "Child"
		case "duplicates":
			return "Duplicated by"
		case "related":
			return "Related"
		default:
			return linkType + " (incoming)"
		}
	}

	// Outgoing labels - capitalize first letter
	switch linkType {
	case "blocks":
		return "Blocks"
	case "parent":
		return "Parent"
	case "duplicates":
		return "Duplicates"
	case "related":
		return "Related"
	default:
		return linkType
	}
}

func (m detailModel) resolveAllLinks() []resolvedLink {
	var links []resolvedLink

	// Get all beans from core (already in memory)
	allBeans := m.core.All()

	// Build a lookup map by ID for fast resolution
	beansByID := make(map[string]*bean.Bean)
	for _, b := range allBeans {
		beansByID[b.ID] = b
	}

	// Resolve outgoing links (this bean links to others)
	outgoing := m.resolveOutgoingLinks(beansByID)
	links = append(links, outgoing...)

	// Resolve incoming links (other beans link to this one)
	incoming := m.resolveIncomingLinks(allBeans)
	links = append(links, incoming...)

	return links
}

func (m detailModel) resolveOutgoingLinks(beansByID map[string]*bean.Bean) []resolvedLink {
	var links []resolvedLink

	// Sort by link type for consistent ordering
	sortedLinks := make([]bean.Link, len(m.bean.Links))
	copy(sortedLinks, m.bean.Links)
	sort.Slice(sortedLinks, func(i, j int) bool {
		if sortedLinks[i].Type != sortedLinks[j].Type {
			return sortedLinks[i].Type < sortedLinks[j].Type
		}
		return sortedLinks[i].Target < sortedLinks[j].Target
	})

	for _, link := range sortedLinks {
		targetBean, ok := beansByID[link.Target]
		if !ok {
			// Skip missing beans per user preference
			continue
		}
		links = append(links, resolvedLink{
			linkType: link.Type,
			bean:     targetBean,
			incoming: false,
		})
	}

	return links
}

func (m detailModel) resolveIncomingLinks(allBeans []*bean.Bean) []resolvedLink {
	var links []resolvedLink

	// Collect incoming links
	type incomingLink struct {
		linkType string
		bean     *bean.Bean
	}
	var incoming []incomingLink

	for _, other := range allBeans {
		if other.ID == m.bean.ID {
			continue
		}

		for _, link := range other.Links {
			if link.Target == m.bean.ID {
				incoming = append(incoming, incomingLink{
					linkType: link.Type,
					bean:     other,
				})
			}
		}
	}

	// Sort by link type then by bean ID
	sort.Slice(incoming, func(i, j int) bool {
		if incoming[i].linkType != incoming[j].linkType {
			return incoming[i].linkType < incoming[j].linkType
		}
		return incoming[i].bean.ID < incoming[j].bean.ID
	})

	for _, inc := range incoming {
		links = append(links, resolvedLink{
			linkType: inc.linkType,
			bean:     inc.bean,
			incoming: true,
		})
	}

	return links
}

func (m detailModel) renderBody(_ int) string {
	if m.bean.Body == "" {
		return lipgloss.NewStyle().
			Foreground(ui.ColorMuted).
			Padding(0, 1).
			Render("No description")
	}

	renderer := getGlamourRenderer()
	if renderer == nil {
		return m.bean.Body
	}

	rendered, err := renderer.Render(m.bean.Body)
	if err != nil {
		return m.bean.Body
	}

	return strings.TrimSpace(rendered)
}
