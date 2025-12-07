package tui

import (
	"fmt"
	"sort"
	"strings"
	"sync"

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

// detailModel displays a single bean's details
type detailModel struct {
	viewport     viewport.Model
	bean         *bean.Bean
	core         *beancore.Core
	config       *config.Config
	width        int
	height       int
	ready        bool
	links        []resolvedLink       // combined outgoing + incoming links
	selectedLink int                  // -1 = none selected, 0+ = index in links
	linksActive  bool                 // true = links section focused
	cols         ui.ResponsiveColumns // responsive column widths for links
}

func newDetailModel(b *bean.Bean, core *beancore.Core, cfg *config.Config, width, height int) detailModel {
	m := detailModel{
		bean:         b,
		core:         core,
		config:       cfg,
		width:        width,
		height:       height,
		ready:        true,
		selectedLink: -1,
		linksActive:  false,
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

	// If there are links, select first one by default
	if len(m.links) > 0 {
		m.selectedLink = 0
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

func (m detailModel) Init() tea.Cmd {
	return nil
}

func (m detailModel) Update(msg tea.Msg) (detailModel, tea.Cmd) {
	var cmd tea.Cmd

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
		switch msg.String() {
		case "esc", "backspace":
			return m, func() tea.Msg {
				return backToListMsg{}
			}

		case "tab":
			// Toggle focus between links and body
			if len(m.links) > 0 {
				m.linksActive = !m.linksActive
				if m.linksActive && m.selectedLink < 0 {
					m.selectedLink = 0
				}
			}
			return m, nil

		case "enter":
			// Navigate to selected link
			if m.linksActive && m.selectedLink >= 0 && m.selectedLink < len(m.links) {
				targetBean := m.links[m.selectedLink].bean
				return m, func() tea.Msg {
					return selectBeanMsg{bean: targetBean}
				}
			}

		case "up", "k":
			if m.linksActive && len(m.links) > 0 {
				if m.selectedLink > 0 {
					m.selectedLink--
				}
				return m, nil
			}

		case "down", "j":
			if m.linksActive && len(m.links) > 0 {
				if m.selectedLink < len(m.links)-1 {
					m.selectedLink++
				}
				return m, nil
			}
		}
	}

	// Only forward to viewport if links are not active
	if !m.linksActive {
		m.viewport, cmd = m.viewport.Update(msg)
	}
	return m, cmd
}

func (m detailModel) View() string {
	if !m.ready {
		return "Loading..."
	}

	// Header
	header := m.renderHeader()

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
		footer += helpKeyStyle.Render("enter") + " " + helpStyle.Render("go to") + "  "
	}
	footer += helpKeyStyle.Render("j/k") + " " + helpStyle.Render("scroll") + "  " +
		helpKeyStyle.Render("esc") + " " + helpStyle.Render("back") + "  " +
		helpKeyStyle.Render("q") + " " + helpStyle.Render("quit")

	return header + "\n" + body + "\n" + footer
}

func (m detailModel) calculateHeaderHeight() int {
	// Base: title line + ID/status line + borders/padding = ~6
	baseHeight := 6

	// Add lines for links
	if len(m.links) > 0 {
		// Count outgoing and incoming separately
		outgoing := 0
		incoming := 0
		for _, l := range m.links {
			if l.incoming {
				incoming++
			} else {
				outgoing++
			}
		}

		// Add link lines
		baseHeight += outgoing + incoming

		// Add separator if we have both types
		if outgoing > 0 && incoming > 0 {
			baseHeight += 1
		}

		// Add top separator
		baseHeight += 1
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

	// Add relationships section if there are any
	if len(m.links) > 0 {
		headerContent.WriteString("\n")
		headerContent.WriteString(ui.Muted.Render(strings.Repeat("─", m.width-8)))
		headerContent.WriteString("\n")
		headerContent.WriteString(m.renderLinks())
	}

	// Header box - highlight border when links are active
	borderColor := ui.ColorMuted
	if m.linksActive {
		borderColor = ui.ColorPrimary
	}

	headerBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1).
		Width(m.width - 4)

	return headerBox.Render(headerContent.String())
}

func (m detailModel) renderLinks() string {
	if len(m.links) == 0 {
		return ""
	}

	var lines []string
	currentIndex := 0

	// Render outgoing links first
	for _, link := range m.links {
		if link.incoming {
			continue
		}
		lines = append(lines, m.renderLinkLine(link, currentIndex))
		currentIndex++
	}

	// Count incoming for separator
	hasIncoming := false
	for _, link := range m.links {
		if link.incoming {
			hasIncoming = true
			break
		}
	}

	// Add separator between outgoing and incoming
	if len(lines) > 0 && hasIncoming {
		lines = append(lines, ui.Muted.Render(strings.Repeat("─", m.width-8)))
	}

	// Render incoming links
	for _, link := range m.links {
		if !link.incoming {
			continue
		}
		lines = append(lines, m.renderLinkLine(link, currentIndex))
		currentIndex++
	}

	return strings.Join(lines, "\n")
}

func (m detailModel) renderLinkLine(link resolvedLink, index int) string {
	// Cursor indicator
	cursor := "  "
	if m.linksActive && index == m.selectedLink {
		cursor = ui.Primary.Render("▸ ")
	}

	// Format the link type label
	label := m.formatLinkLabel(link.linkType, link.incoming)
	labelCol := lipgloss.NewStyle().Width(12).Render(ui.Muted.Render(label + ":"))

	// Get status and type colors
	statusColor := "gray"
	if statusCfg := m.config.GetStatus(link.bean.Status); statusCfg != nil {
		statusColor = statusCfg.Color
	}
	isArchive := m.config.IsArchiveStatus(link.bean.Status)

	typeColor := ""
	if typeCfg := m.config.GetType(link.bean.Type); typeCfg != nil {
		typeColor = typeCfg.Color
	}

	// Calculate max title width using responsive columns
	baseWidth := m.cols.ID + m.cols.Status + m.cols.Type + 12 + 4 // label + cursor + padding
	if m.cols.ShowTags {
		baseWidth += m.cols.Tags
	}
	maxTitleWidth := max(10, m.width-baseWidth-8) // 8 for border padding

	// Use shared bean row rendering (without cursor, we handle it separately)
	row := ui.RenderBeanRow(
		link.bean.ID,
		link.bean.Status,
		link.bean.Type,
		link.bean.Title,
		ui.BeanRowConfig{
			StatusColor:   statusColor,
			TypeColor:     typeColor,
			IsArchive:     isArchive,
			MaxTitleWidth: maxTitleWidth,
			ShowCursor:    false,
			IsSelected:    false,
			Tags:          link.bean.Tags,
			ShowTags:      m.cols.ShowTags,
			TagsColWidth:  m.cols.Tags,
			MaxTags:       m.cols.MaxTags,
		},
	)

	return cursor + labelCol + row
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
		return lipgloss.NewStyle().Foreground(ui.ColorMuted).Italic(true).Render("No description")
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
