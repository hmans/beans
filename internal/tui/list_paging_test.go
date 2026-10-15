package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/hmans/beans/internal/ui"
	"github.com/hmans/beans/pkg/bean"
	"github.com/hmans/beans/pkg/config"
)

// visibleListIDs returns the IDs of items rendered in the list pane, which
// occupies the top paneRows rows and the left paneWidth columns of the view.
func visibleListIDs(view string, paneWidth, paneRows int, items []ui.FlatItem) []string {
	lines := strings.Split(ansi.Strip(view), "\n")
	pane := make([]string, 0, paneRows)
	for _, line := range lines[:min(paneRows, len(lines))] {
		pane = append(pane, ansi.Truncate(line, paneWidth, ""))
	}
	text := strings.Join(pane, "\n")
	var ids []string
	for _, it := range items {
		if strings.Contains(text, it.Bean.ID) {
			ids = append(ids, it.Bean.ID)
		}
	}
	return ids
}

func TestListPagingMatchesVisiblePane(t *testing.T) {
	var items []ui.FlatItem
	for i := range 100 {
		items = append(items, ui.FlatItem{
			Bean:    &bean.Bean{ID: fmt.Sprintf("beans-%04d", i), Title: "Title", Status: "todo", Type: "task"},
			Matched: true,
		})
	}
	leftWidth, _ := calculatePaneWidths(140)

	tests := []struct {
		name                string
		width, height       int
		paneWidth, paneRows int
		resizeInHelp        bool // resize while the help overlay is open
	}{
		{"single column", 100, 30, 100, 30 - footerHeight, false},
		{"two columns", 140, 30, leftWidth, 30 - footerHeight, false},
		{"stacked", 100, 40, 100, config.DefaultStackedListHeight, false},
		{"stacked after resize in help overlay", 100, 40, 100, config.DefaultStackedListHeight, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := New(nil, config.Default())
			if tt.resizeInHelp {
				a.Update(tea.WindowSizeMsg{Width: 200, Height: 60})
				a.Update(beansLoadedMsg{items: items, idColWidth: 12})
				a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
				a.Update(tea.WindowSizeMsg{Width: tt.width, Height: tt.height})
				_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyEsc})
				a.Update(cmd())
				if a.state != viewList {
					t.Fatalf("state after esc = %v, want list", a.state)
				}
			} else {
				a.Update(tea.WindowSizeMsg{Width: tt.width, Height: tt.height})
				a.Update(beansLoadedMsg{items: items, idColWidth: 12})
			}

			firstPage := visibleListIDs(a.View(), tt.paneWidth, tt.paneRows, items)
			if len(firstPage) == 0 {
				t.Fatalf("no beans visible in the list pane\n%s", a.View())
			}

			a.Update(tea.KeyMsg{Type: tea.KeyRight})
			want := items[len(firstPage)].Bean.ID
			secondPage := visibleListIDs(a.View(), tt.paneWidth, tt.paneRows, items)
			if len(secondPage) == 0 || secondPage[0] != want {
				t.Errorf("after right: first visible = %v, want %s (page 1 showed %d beans)", secondPage, want, len(firstPage))
			}
			if got := a.list.list.Index(); got != len(firstPage) {
				t.Errorf("after right: index = %d, want %d", got, len(firstPage))
			}

			a.Update(tea.KeyMsg{Type: tea.KeyLeft})
			if got := a.list.list.Index(); got != 0 {
				t.Errorf("after left: index = %d, want 0", got)
			}
		})
	}
}
