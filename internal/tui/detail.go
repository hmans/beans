package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"hmans.dev/beans/internal/bean"
	"hmans.dev/beans/internal/config"
	"hmans.dev/beans/internal/ui"
)

// backToListMsg signals navigation back to the list
type backToListMsg struct{}

// detailModel displays a single bean's details
type detailModel struct {
	viewport viewport.Model
	bean     *bean.Bean
	config   *config.Config
	width    int
	height   int
	ready    bool
}

func newDetailModel(b *bean.Bean, cfg *config.Config, width, height int) detailModel {
	headerHeight := 6
	footerHeight := 2
	vpWidth := width - 4
	vpHeight := height - headerHeight - footerHeight

	m := detailModel{
		bean:   b,
		config: cfg,
		width:  width,
		height: height,
		ready:  true,
	}

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

		headerHeight := 6
		footerHeight := 2
		vpWidth := msg.Width - 4
		vpHeight := msg.Height - headerHeight - footerHeight

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
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m detailModel) View() string {
	if !m.ready {
		return "Loading..."
	}

	// Header
	header := m.renderHeader()

	// Body
	bodyBorder := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorMuted).
		Width(m.width - 4)
	body := bodyBorder.Render(m.viewport.View())

	// Footer
	scrollPct := int(m.viewport.ScrollPercent() * 100)
	footer := helpStyle.Render(fmt.Sprintf("%d%%", scrollPct)) + "  " +
		helpKeyStyle.Render("j/k") + " " + helpStyle.Render("scroll") + "  " +
		helpKeyStyle.Render("esc") + " " + helpStyle.Render("back") + "  " +
		helpKeyStyle.Render("q") + " " + helpStyle.Render("quit")

	return header + "\n" + body + "\n" + footer
}

func (m detailModel) renderHeader() string {
	// Title badge
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

	// Header box
	headerContent := title + "\n" + id + "  " + status
	headerBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorPrimary).
		Padding(0, 1).
		Width(m.width - 4)

	return headerBox.Render(headerContent)
}

func (m detailModel) renderBody(width int) string {
	if m.bean.Description == "" {
		return lipgloss.NewStyle().Foreground(ui.ColorMuted).Italic(true).Render("No description")
	}

	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width-4),
	)
	if err != nil {
		return m.bean.Description
	}

	rendered, err := renderer.Render(m.bean.Description)
	if err != nil {
		return m.bean.Description
	}

	return strings.TrimSpace(rendered)
}
