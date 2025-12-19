package tui

import (
	"fmt"
	"io"
	"sort"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hmans/beans/internal/ui"
)

// tagWithCount holds a tag and its usage count
type tagWithCount struct {
	tag   string
	count int
}

// tagItem wraps a tag with count to implement list.Item
type tagItem struct {
	tag   string
	count int
}

func (i tagItem) Title() string       { return i.tag }
func (i tagItem) Description() string { return "" }
func (i tagItem) FilterValue() string { return i.tag }

// tagItemDelegate handles rendering of tag items
type tagItemDelegate struct{}

func (d tagItemDelegate) Height() int                             { return 1 }
func (d tagItemDelegate) Spacing() int                            { return 0 }
func (d tagItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d tagItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(tagItem)
	if !ok {
		return
	}

	var cursor string
	if index == m.Index() {
		cursor = lipgloss.NewStyle().Foreground(ui.ColorPrimary).Bold(true).Render("▌") + " "
	} else {
		cursor = "  "
	}

	tagBadge := ui.RenderTag(item.tag)
	count := ui.Muted.Render(fmt.Sprintf(" (%d)", item.count))

	fmt.Fprint(w, cursor+tagBadge+count)
}

// tagPickerModel is the model for the tag picker view
type tagPickerModel struct {
	list   list.Model
	tags   []tagWithCount
	width  int
	height int
}

func newTagPickerModel(tags []tagWithCount, width, height int) tagPickerModel {
	// Sort by count descending, then alphabetically
	sort.Slice(tags, func(i, j int) bool {
		if tags[i].count != tags[j].count {
			return tags[i].count > tags[j].count
		}
		return tags[i].tag < tags[j].tag
	})

	delegate := tagItemDelegate{}

	items := make([]list.Item, len(tags))
	for i, t := range tags {
		items[i] = tagItem{tag: t.tag, count: t.count}
	}

	l := list.New(items, delegate, width-4, height-6)
	l.Title = "Select a Tag"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)
	l.Styles.Title = listTitleStyle
	l.Styles.TitleBar = lipgloss.NewStyle().Padding(0, 0, 1, 1)
	l.Styles.FilterPrompt = lipgloss.NewStyle().Foreground(ui.ColorPrimary)
	l.Styles.FilterCursor = lipgloss.NewStyle().Foreground(ui.ColorPrimary)

	return tagPickerModel{
		list:   l,
		tags:   tags,
		width:  width,
		height: height,
	}
}

func (m tagPickerModel) Init() tea.Cmd {
	return nil
}

func (m tagPickerModel) Update(msg tea.Msg) (tagPickerModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width-4, msg.Height-6)

	case tea.KeyMsg:
		if m.list.FilterState() != list.Filtering {
			switch msg.String() {
			case "enter":
				if item, ok := m.list.SelectedItem().(tagItem); ok {
					return m, func() tea.Msg {
						return tagSelectedMsg{tag: item.tag}
					}
				}
			case "esc", "backspace":
				// Return to list without selecting a tag
				return m, func() tea.Msg {
					return backToListMsg{}
				}
			}
		}
	}

	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m tagPickerModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	// Simple bordered container
	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorPrimary).
		Width(m.width - 2).
		Height(m.height - 4)

	content := border.Render(m.list.View())

	// Footer
	help := helpKeyStyle.Render("enter") + " " + helpStyle.Render("select") + "  " +
		helpKeyStyle.Render("/") + " " + helpStyle.Render("filter") + "  " +
		helpKeyStyle.Render("esc") + " " + helpStyle.Render("cancel") + "  " +
		helpKeyStyle.Render("q") + " " + helpStyle.Render("quit")

	return content + "\n" + help
}
