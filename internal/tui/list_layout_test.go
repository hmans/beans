package tui

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/hmans/beans/internal/ui"
	"github.com/hmans/beans/pkg/bean"
	"github.com/hmans/beans/pkg/config"
)

func TestListRowsFitPane(t *testing.T) {
	var items []ui.FlatItem
	for i := range 5 {
		items = append(items, ui.FlatItem{
			Bean: &bean.Bean{ID: fmt.Sprintf("anvl-%04d", i), Status: "completed", Type: "task", Priority: "critical",
				Title: strings.Repeat("a long title that must be truncated ", 6), Tags: []string{"idea", "frontend"}},
			Depth:      1,
			Matched:    true,
			TreePrefix: "├─",
		})
	}

	// The widths cover short, cut and full type/status names, with and without tags.
	for _, width := range []int{80, 120, 124, 139, 140, 150, 160, 190, 220, 250} {
		t.Run(fmt.Sprint(width), func(t *testing.T) {
			m := newListModel(nil, config.Default())
			m, _ = m.Update(tea.WindowSizeMsg{Width: width, Height: 40})
			m, _ = m.Update(beansLoadedMsg{items: items, idColWidth: 9 + 2 + 2})
			out := ansi.Strip(m.ViewConstrained(width, config.DefaultStackedListHeight))

			for _, line := range strings.Split(out, "\n") {
				if got := lipgloss.Width(line); got != width {
					t.Errorf("line width = %d, want %d: %q", got, width, line)
				}
				if strings.TrimSpace(strings.Trim(line, "│")) == "..." {
					t.Errorf("truncation marker wrapped onto its own line\n%s", out)
				}
			}
		})
	}
}

func TestListViewConstrained_ExactSize(t *testing.T) {
	// Deep tree prefixes shrink the title budget to a few columns.
	items := []ui.FlatItem{{
		Bean:    &bean.Bean{ID: "beans-root", Title: "Completed milestone", Status: "completed", Type: "milestone"},
		Matched: true,
	}}
	for i := range 30 {
		items = append(items, ui.FlatItem{
			Bean:           &bean.Bean{ID: fmt.Sprintf("beans-%04d", i), Title: "A child title long enough to be truncated", Status: "completed", Type: "task"},
			Depth:          1 + i%3,
			Matched:        true,
			TreePrefix:     strings.Repeat("│  ", i%3) + "├─",
			ImplicitStatus: "completed",
		})
	}
	const idColWidth = 10 + 2 + 3*3

	for _, sz := range [][2]int{{120, 27}, {120, 28}, {140, 40}} {
		width, height := sz[0], sz[1]
		t.Run(fmt.Sprintf("%dx%d", width, height), func(t *testing.T) {
			m := newListModel(nil, config.Default())
			m, _ = m.Update(tea.WindowSizeMsg{Width: width, Height: height})
			m, _ = m.Update(beansLoadedMsg{items: items, idColWidth: idColWidth})

			leftWidth, _ := calculatePaneWidths(width)
			contentHeight := height - 1
			out := m.ViewConstrained(leftWidth, contentHeight)

			if got := lipgloss.Height(out); got != contentHeight {
				t.Errorf("pane height = %d, want %d\n%s", got, contentHeight, out)
			}
			if got := lipgloss.Width(out); got != leftWidth {
				t.Errorf("pane width = %d, want %d", got, leftWidth)
			}
		})
	}
}

// wideDelegate renders every row twice as wide as the list.
type wideDelegate struct{}

func (wideDelegate) Height() int                             { return 1 }
func (wideDelegate) Spacing() int                            { return 0 }
func (wideDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (wideDelegate) Render(w io.Writer, m list.Model, _ int, _ list.Item) {
	fmt.Fprint(w, strings.Repeat("x", 2*m.Width()))
}

func TestListViewContentClipsWideRows(t *testing.T) {
	items := make([]ui.FlatItem, 30)
	for i := range items {
		items[i] = ui.FlatItem{Bean: &bean.Bean{ID: fmt.Sprintf("beans-%04d", i), Status: "todo", Type: "task"}, Matched: true}
	}

	for _, sz := range [][2]int{{40, 5}, {80, 15}, {120, 26}} {
		width, innerHeight := sz[0], sz[1]
		t.Run(fmt.Sprintf("%dx%d", width, innerHeight), func(t *testing.T) {
			m := newListModel(nil, config.Default())
			m, _ = m.Update(tea.WindowSizeMsg{Width: width, Height: innerHeight + 10})
			m, _ = m.Update(beansLoadedMsg{items: items, idColWidth: 12})
			m.width = width
			m.list.SetSize(width-paneBorders, innerHeight)
			m.list.SetDelegate(wideDelegate{})

			out := m.viewContent(innerHeight)
			if got := lipgloss.Height(out); got != innerHeight+paneBorders {
				t.Errorf("pane height = %d, want %d", got, innerHeight+paneBorders)
			}
			if got := lipgloss.Width(out); got != width {
				t.Errorf("pane width = %d, want %d", got, width)
			}
		})
	}
}

func TestListTitleInBorder(t *testing.T) {
	items := []ui.FlatItem{{Bean: &bean.Bean{ID: "beans-0001", Title: "A", Status: "todo", Type: "task"}, Matched: true}}
	m := newListModel(nil, config.Default())
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	m, _ = m.Update(beansLoadedMsg{items: items, idColWidth: 12})

	lines := strings.Split(ansi.Strip(m.ViewConstrained(80, 15)), "\n")
	if !strings.HasPrefix(lines[0], "╭─ Beans ─") {
		t.Errorf("top border = %q, want the title in it", lines[0])
	}
	if !strings.Contains(lines[1], "beans-0001") {
		t.Errorf("first row = %q, want the first bean", lines[1])
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("ab")})
	lines = strings.Split(ansi.Strip(m.ViewConstrained(80, 15)), "\n")
	if !strings.HasPrefix(lines[0], "╭─ Filter: ab") {
		t.Errorf("top border while filtering = %q, want the filter input in it", lines[0])
	}
	if w := lipgloss.Width(lines[0]); w != 80 {
		t.Errorf("top border width while filtering = %d, want 80", w)
	}
}

func TestWithBorderTitleTruncates(t *testing.T) {
	pane := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Width(18).Render("x")
	top, _, _ := strings.Cut(ansi.Strip(withBorderTitle(pane, strings.Repeat("t", 40))), "\n")
	if w := lipgloss.Width(top); w != 20 {
		t.Errorf("top border width = %d, want 20: %q", w, top)
	}
	if !strings.HasSuffix(top, "… ─╮") {
		t.Errorf("top border = %q, want truncated title followed by border", top)
	}
}
