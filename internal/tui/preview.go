package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/ui"
)

// previewModel is a read-only detail preview for the two-column layout.
// It has no focus, no interaction - just renders bean details.
type previewModel struct {
	bean   *bean.Bean
	width  int
	height int
}

func newPreviewModel(b *bean.Bean, width, height int) previewModel {
	return previewModel{
		bean:   b,
		width:  width,
		height: height,
	}
}

func (m previewModel) View() string {
	if m.bean == nil {
		return m.renderEmpty()
	}
	return m.renderBean()
}

func (m previewModel) renderEmpty() string {
	style := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center).
		Foreground(ui.ColorMuted)

	return style.Render("No bean selected")
}

func (m previewModel) renderBean() string {
	// Header: ID and Title
	idStyle := lipgloss.NewStyle().Foreground(ui.ColorPrimary).Bold(true)
	titleStyle := lipgloss.NewStyle().Bold(true)

	header := idStyle.Render(m.bean.ID) + "\n" + titleStyle.Render(m.bean.Title)

	// Metadata: Status, Type, Priority
	metaStyle := lipgloss.NewStyle().Foreground(ui.ColorMuted)
	meta := metaStyle.Render("Status: " + m.bean.Status + "  Type: " + m.bean.Type)
	if m.bean.Priority != "" && m.bean.Priority != "normal" {
		meta += metaStyle.Render("  Priority: " + m.bean.Priority)
	}

	// Tags
	var tagsLine string
	if len(m.bean.Tags) > 0 {
		tagsLine = ui.RenderTags(m.bean.Tags)
	}

	// Body (truncated to fit)
	body := m.renderBody()

	// Compose
	var parts []string
	parts = append(parts, header)
	parts = append(parts, "")
	parts = append(parts, meta)
	if tagsLine != "" {
		parts = append(parts, tagsLine)
	}
	parts = append(parts, "")
	parts = append(parts, body)

	content := lipgloss.JoinVertical(lipgloss.Left, parts...)

	// Border
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorMuted).
		Padding(0, 1).
		Width(m.width - 2).
		Height(m.height - 2)

	return borderStyle.Render(content)
}

func (m previewModel) renderBody() string {
	if m.bean.Body == "" {
		return lipgloss.NewStyle().Foreground(ui.ColorMuted).Render("No description")
	}

	// Render markdown (reuse existing glamour renderer from detail.go)
	renderer := getGlamourRenderer()
	if renderer == nil {
		return m.bean.Body
	}

	rendered, err := renderer.Render(m.bean.Body)
	if err != nil {
		return m.bean.Body
	}

	// Truncate to available height
	lines := strings.Split(rendered, "\n")
	// Account for header (2 lines), blank line, meta (1 line), tags (0-1 line), blank line, borders/padding
	// Estimate ~8 lines for header/meta
	availableLines := m.height - 8
	if availableLines < 1 {
		availableLines = 1
	}

	if len(lines) > availableLines {
		lines = lines[:availableLines]
		lines = append(lines, lipgloss.NewStyle().Foreground(ui.ColorMuted).Render("..."))
	}

	return strings.TrimSpace(strings.Join(lines, "\n"))
}
