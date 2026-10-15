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

// screenRowOf returns the screen row on which id is rendered, or -1.
func screenRowOf(view, id string) int {
	for y, line := range strings.Split(view, "\n") {
		if strings.Contains(ansi.Strip(line), id) {
			return y
		}
	}
	return -1
}

func TestListClickSelectsItem(t *testing.T) {
	var items []ui.FlatItem
	for i := range 60 {
		items = append(items, ui.FlatItem{
			Bean:    &bean.Bean{ID: fmt.Sprintf("beans-%04d", i), Title: "Title", Status: "todo", Type: "task"},
			Matched: true,
		})
	}

	tests := []struct {
		name          string
		width, height int
		startIndex    int // cursor before the click, which determines the page
		target        int
	}{
		{"single column, first page", 100, 30, 0, 5},
		{"single column, later page", 100, 30, 40, 42},
		{"two columns, first page", 140, 30, 0, 7},
		{"two columns, later page", 140, 30, 40, 44},
		{"stacked, first page", 100, 40, 0, 5},
		{"stacked, later page", 100, 40, 40, 42},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := New(nil, config.Default())
			a.Update(tea.WindowSizeMsg{Width: tt.width, Height: tt.height})
			a.Update(beansLoadedMsg{items: items, idColWidth: 12})
			a.list.list.Select(tt.startIndex)

			targetID := items[tt.target].Bean.ID
			y := screenRowOf(a.View(), targetID)
			if y < 0 {
				t.Fatalf("%s not rendered on screen", targetID)
			}

			_, cmd := a.Update(tea.MouseMsg{X: 5, Y: y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})

			if got := a.list.list.Index(); got != tt.target {
				t.Errorf("index after click = %d, want %d", got, tt.target)
			}
			if cmd == nil {
				t.Fatal("click returned no command, want cursorChangedMsg")
			}
			if msg, ok := cmd().(cursorChangedMsg); !ok || msg.beanID != targetID {
				t.Errorf("cmd() = %#v, want cursorChangedMsg{%q}", msg, targetID)
			}
		})
	}
}

func TestListClickOutsideItemsKeepsSelection(t *testing.T) {
	items := []ui.FlatItem{
		{Bean: &bean.Bean{ID: "beans-0001", Title: "A", Status: "todo", Type: "task"}, Matched: true},
		{Bean: &bean.Bean{ID: "beans-0002", Title: "B", Status: "todo", Type: "task"}, Matched: true},
	}

	tests := []struct {
		name string
		msg  tea.MouseMsg
	}{
		{"top border", tea.MouseMsg{X: 5, Y: 0, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}},
		{"below last item", tea.MouseMsg{X: 5, Y: 10, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}},
		{"preview pane", tea.MouseMsg{X: 130, Y: 4, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}},
		{"wheel", tea.MouseMsg{X: 5, Y: 4, Action: tea.MouseActionPress, Button: tea.MouseButtonWheelDown}},
		{"release", tea.MouseMsg{X: 5, Y: 4, Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := New(nil, config.Default())
			a.Update(tea.WindowSizeMsg{Width: 140, Height: 30})
			a.Update(beansLoadedMsg{items: items, idColWidth: 12})

			a.Update(tt.msg)

			if got := a.list.list.Index(); got != 0 {
				t.Errorf("index after %s = %d, want 0", tt.name, got)
			}
		})
	}
}
