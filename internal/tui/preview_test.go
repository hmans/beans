package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/hmans/beans/internal/ui"
	"github.com/hmans/beans/pkg/bean"
	"github.com/hmans/beans/pkg/config"
)

func TestPreviewView(t *testing.T) {
	b := &bean.Bean{
		ID:       "beans-test",
		Title:    "Test Bean",
		Status:   "todo",
		Type:     "feature",
		Priority: "high",
		Tags:     []string{"frontend", "design"},
		Body:     "## Summary\n\nThis is the body.",
	}

	preview := newPreviewModel(b, 60, 20)
	view := preview.View()

	// Should contain the title
	if !strings.Contains(view, "Test Bean") {
		t.Error("preview should contain bean title")
	}

	lines := strings.Split(ansi.Strip(view), "\n")
	if !strings.HasPrefix(lines[0], "╭─ beans-test ─") {
		t.Errorf("top border = %q, want the bean ID in it", lines[0])
	}
	if !strings.Contains(lines[1], "Test Bean") {
		t.Errorf("first row = %q, want the bean title", lines[1])
	}

	// Should contain status
	if !strings.Contains(view, "todo") {
		t.Error("preview should contain status")
	}

	// Should contain type
	if !strings.Contains(view, "feature") {
		t.Error("preview should contain type")
	}

	// Should contain body content
	if !strings.Contains(view, "Summary") {
		t.Error("preview should contain body")
	}
}

func TestPreviewViewEmpty(t *testing.T) {
	preview := newPreviewModel(nil, 60, 20)
	view := preview.View()

	if !strings.Contains(view, "No bean selected") {
		t.Error("empty preview should show 'No bean selected'")
	}
}

func TestPreviewViewWithTags(t *testing.T) {
	b := &bean.Bean{
		ID:     "beans-test",
		Title:  "Bean with Tags",
		Status: "in-progress",
		Type:   "bug",
		Tags:   []string{"urgent", "backend"},
		Body:   "Test body",
	}

	preview := newPreviewModel(b, 60, 20)
	view := preview.View()

	// Should show tags
	if !strings.Contains(view, "urgent") || !strings.Contains(view, "backend") {
		t.Error("preview should display tags")
	}
}

func TestPreviewViewWithPriority(t *testing.T) {
	b := &bean.Bean{
		ID:       "beans-test",
		Title:    "High Priority Bean",
		Status:   "todo",
		Type:     "task",
		Priority: "critical",
		Body:     "Important work",
	}

	preview := newPreviewModel(b, 60, 20)
	view := preview.View()

	// Should show priority
	if !strings.Contains(view, "critical") {
		t.Error("preview should display priority when not normal")
	}
}

func TestPreviewViewBlocked(t *testing.T) {
	b := &bean.Bean{ID: "beans-test", Title: "Bean", Status: "todo", Type: "task"}
	for _, blocked := range []bool{false, true} {
		preview := newPreviewModel(b, 60, 20)
		preview.blocked = blocked
		view := ansi.Strip(preview.View())
		if got := strings.Contains(view, "Status: todo ⊘ blocked  Type: task"); got != blocked {
			t.Errorf("blocked=%v: blocked mark shown = %v\n%s", blocked, got, view)
		}
	}
}

// The preview takes the blocked mark from the list item, with the list's
// precedence of a closed ancestor over a blocker.
func TestPreviewBlockedFromList(t *testing.T) {
	tests := []struct {
		name           string
		blocked        bool
		implicitStatus string
		want           bool
	}{
		{"unblocked", false, "", false},
		{"blocked", true, "", true},
		{"blocked below closed ancestor", true, "completed", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &bean.Bean{ID: "beans-0001", Title: "A", Status: "todo", Type: "task"}
			items := []ui.FlatItem{
				{Bean: b, Matched: true, ImplicitStatus: tt.implicitStatus},
				{Bean: &bean.Bean{ID: "beans-0002", Title: "B", Status: "todo", Type: "task"}, Matched: true},
			}
			a := New(nil, config.Default())
			a.Update(tea.WindowSizeMsg{Width: TwoColumnFullWidth, Height: 30})
			a.Update(beansLoadedMsg{items: items, idColWidth: 12, blocked: map[string]bool{b.ID: tt.blocked}})
			if a.preview.blocked != tt.want {
				t.Errorf("after load: preview blocked = %v, want %v", a.preview.blocked, tt.want)
			}

			// Move away and back; the cursor message carries the mark.
			a.list, _ = a.list.Update(tea.KeyMsg{Type: tea.KeyDown})
			_, cmd := a.list.Update(tea.KeyMsg{Type: tea.KeyUp})
			msg, ok := findMsg[cursorChangedMsg](cmd)
			if !ok {
				t.Fatal("cursor move emitted no cursorChangedMsg")
			}
			if msg.beanID != b.ID || msg.blocked != tt.want {
				t.Errorf("cursorChangedMsg = %+v, want beanID %s, blocked %v", msg, b.ID, tt.want)
			}
		})
	}
}

// findMsg runs cmd, descending into batches, and returns the first T.
func findMsg[T tea.Msg](cmd tea.Cmd) (T, bool) {
	var zero T
	if cmd == nil {
		return zero, false
	}
	switch msg := cmd().(type) {
	case T:
		return msg, true
	case tea.BatchMsg:
		for _, c := range msg {
			if m, ok := findMsg[T](c); ok {
				return m, true
			}
		}
	}
	return zero, false
}

func TestPreviewViewEmptyBody(t *testing.T) {
	b := &bean.Bean{
		ID:     "beans-test",
		Title:  "Bean without body",
		Status: "todo",
		Type:   "task",
		Body:   "",
	}

	preview := newPreviewModel(b, 60, 20)
	view := preview.View()

	// Should show placeholder for empty body
	if !strings.Contains(view, "No description") {
		t.Error("preview should show 'No description' for empty body")
	}
}

func TestPreviewWrapsBodyOnce(t *testing.T) {
	line80 := strings.Repeat("abcdefghi ", 7) + "abcdefghij"
	b := &bean.Bean{ID: "beans-wrap", Title: "Wrap", Status: "todo", Type: "task",
		Body: line80 + "\n" + strings.Repeat("word ", 60)}

	for _, width := range []int{RightPaneMaxWidth, 60} {
		t.Run(fmt.Sprint(width), func(t *testing.T) {
			view := ansi.Strip(newPreviewModel(b, width, 40).View())
			for _, l := range strings.Split(view, "\n") {
				if w := lipgloss.Width(l); w > width {
					t.Errorf("line width %d exceeds pane width %d: %q", w, width, l)
				}
				if trimmed := strings.TrimSpace(strings.Trim(l, "│")); trimmed == "word" {
					t.Errorf("stray single-word line, body was wrapped twice\n%s", view)
				}
			}
			if width == RightPaneMaxWidth && !strings.Contains(view, line80) {
				t.Errorf("80-column line was broken\n%s", view)
			}
		})
	}
}

func longBodyBean(lines int) *bean.Bean {
	var body strings.Builder
	for i := range lines {
		fmt.Fprintf(&body, "- item %02d\n", i)
	}
	return &bean.Bean{ID: "beans-long", Title: "Long", Status: "todo", Type: "task", Body: body.String()}
}

func TestPreviewScroll(t *testing.T) {
	const height = 20

	tests := []struct {
		name        string
		scrolls     []int
		wantVisible []string
		wantHidden  []string
	}{
		{"initial", nil, []string{"item 00"}, []string{"item 30", "item 49"}},
		{"scrolled down", []int{10}, []string{"item 12"}, []string{"item 05", "item 49"}},
		{"clamped at end", []int{1000}, []string{"item 49"}, []string{"item 00", "..."}},
		{"one notch up from end", []int{1000, -previewScrollStep}, []string{"item 34", "..."}, []string{"item 49"}},
		{"clamped at start", []int{1000, -2000}, []string{"item 00", "..."}, []string{"item 49"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newPreviewModel(longBodyBean(50), RightPaneMaxWidth, height)
			for _, d := range tt.scrolls {
				p.scrollBy(d)
			}
			view := ansi.Strip(p.View())

			if got := lipgloss.Height(view); got != height {
				t.Errorf("height = %d, want %d", got, height)
			}
			if !strings.Contains(view, "beans-long") {
				t.Error("header scrolled away, want it fixed")
			}
			for _, s := range tt.wantVisible {
				if !strings.Contains(view, s) {
					t.Errorf("%q not visible\n%s", s, view)
				}
			}
			for _, s := range tt.wantHidden {
				if strings.Contains(view, s) {
					t.Errorf("%q visible, want hidden\n%s", s, view)
				}
			}
		})
	}
}

func TestStackedLayout(t *testing.T) {
	items := []ui.FlatItem{
		{Bean: longBodyBean(50), Matched: true},
		{Bean: &bean.Bean{ID: "beans-0002", Title: "B", Status: "todo", Type: "task"}, Matched: true},
	}
	defaultMin := config.DefaultStackedListHeight + StackedBelowListMinHeight

	tests := []struct {
		name          string
		listHeight    int // configured tui.stacked_list_height; 0 keeps the default
		width, height int
		wantPreview   bool
	}{
		{"narrow and tall", 0, 100, defaultMin, true},
		{"narrow and short", 0, 100, defaultMin - 1, false},
		{"just below two-column width", 0, TwoColumnMinWidth - 1, 50, true},
		{"two-column preview would shrink", 0, TwoColumnFullWidth - 1, defaultMin, true},
		{"configured taller list", 25, 100, 25 + StackedBelowListMinHeight, true},
		{"configured taller list, too short", 25, 100, 25 + StackedBelowListMinHeight - 1, false},
		{"configured shorter list", 8, 100, 8 + StackedBelowListMinHeight, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.Default()
			cfg.TUI.StackedListHeight = tt.listHeight
			listHeight := cfg.GetStackedListHeight()
			a := New(nil, cfg)
			a.Update(tea.WindowSizeMsg{Width: tt.width, Height: tt.height})
			a.Update(beansLoadedMsg{items: items, idColWidth: 12})
			view := ansi.Strip(a.View())
			lines := strings.Split(view, "\n")

			gotPreview := strings.Contains(strings.Join(lines[min(listHeight, len(lines)):], "\n"), "item 00")
			if gotPreview != tt.wantPreview {
				t.Fatalf("preview below list = %v, want %v\n%s", gotPreview, tt.wantPreview, view)
			}
			if !tt.wantPreview {
				return
			}
			if got := len(lines); got != tt.height {
				t.Errorf("view height = %d, want %d", got, tt.height)
			}
			// The footer is excluded: its help line is not truncated in any layout.
			if got := lipgloss.Width(strings.Join(lines[:len(lines)-1], "\n")); got > tt.width {
				t.Errorf("panes width = %d, want <= %d", got, tt.width)
			}
			if got := lipgloss.Width(lines[listHeight]); got != min(tt.width, StackedPreviewMaxWidth) {
				t.Errorf("preview width = %d, want %d", got, min(tt.width, StackedPreviewMaxWidth))
			}

			wheel := func(y int) {
				a.Update(tea.MouseMsg{X: 5, Y: y, Action: tea.MouseActionPress, Button: tea.MouseButtonWheelDown})
			}
			wheel(listHeight - 1)
			if a.preview.scroll != 0 {
				t.Errorf("wheel over list: preview scroll = %d, want 0", a.preview.scroll)
			}
			wheel(listHeight)
			if a.preview.scroll != previewScrollStep {
				t.Errorf("wheel over preview: scroll = %d, want %d", a.preview.scroll, previewScrollStep)
			}
		})
	}
}

func TestLayoutSelection(t *testing.T) {
	stackedMin := config.DefaultStackedListHeight + StackedBelowListMinHeight
	auto, right, below := config.PreviewPositionAuto, config.PreviewPositionRight, config.PreviewPositionBelow
	tests := []struct {
		name          string
		position      config.PreviewPosition
		width, height int
		wantTwoColumn bool
		wantStacked   bool
	}{
		{"wide", auto, TwoColumnFullWidth, stackedMin, true, false},
		{"preview would shrink, tall", auto, TwoColumnFullWidth - 1, stackedMin, false, true},
		{"preview would shrink, short", auto, TwoColumnFullWidth - 1, stackedMin - 1, true, false},
		{"narrow, tall", auto, TwoColumnMinWidth - 1, stackedMin, false, true},
		{"narrow, short", auto, TwoColumnMinWidth - 1, stackedMin - 1, false, false},
		{"right, preview would shrink, tall", right, TwoColumnFullWidth - 1, stackedMin, true, false},
		{"right, narrow, tall falls back to below", right, TwoColumnMinWidth - 1, stackedMin, false, true},
		{"right, narrow, short", right, TwoColumnMinWidth - 1, stackedMin - 1, false, false},
		{"below, wide", below, TwoColumnFullWidth, stackedMin, false, true},
		{"below, wide, short stays below", below, TwoColumnFullWidth, stackedMin - 1, false, true},
		{"below, narrow, room for both panes", below, TwoColumnMinWidth - 1, footerHeight + 2*config.MinStackedListHeight, false, true},
		{"below, narrow, no room for the preview", below, TwoColumnMinWidth - 1, footerHeight + 2*config.MinStackedListHeight - 1, false, false},
		{"below, wide, no room for the preview", below, TwoColumnFullWidth, footerHeight + 2*config.MinStackedListHeight - 1, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.Default()
			cfg.TUI.PreviewPosition = tt.position
			a := New(nil, cfg)
			a.width, a.height = tt.width, tt.height
			if got := a.isTwoColumnMode(); got != tt.wantTwoColumn {
				t.Errorf("isTwoColumnMode() = %v, want %v", got, tt.wantTwoColumn)
			}
			if got := a.isStackedMode(); got != tt.wantStacked {
				t.Errorf("isStackedMode() = %v, want %v", got, tt.wantStacked)
			}
			if tt.wantTwoColumn && tt.width >= TwoColumnFullWidth {
				if w, _ := a.previewSize(); w-previewChromeX < PreviewMinTextWidth {
					t.Errorf("preview text width = %d, want >= %d", w-previewChromeX, PreviewMinTextWidth)
				}
			}
		})
	}
}

func TestPreviewRatio(t *testing.T) {
	items := []ui.FlatItem{{Bean: longBodyBean(50), Matched: true}}
	auto, right, below := config.PreviewPositionAuto, config.PreviewPositionRight, config.PreviewPositionBelow
	tests := []struct {
		name          string
		position      config.PreviewPosition
		width, height int
		percent       int
		wantStacked   bool
		wantList      int // list pane height, border included
	}{
		{"below, even split", below, 100, 40, 50, true, 20},
		{"below, small preview", below, 100, 40, 20, true, 31},
		{"below, large preview", below, 100, 40, 90, true, 5},
		{"below, list clamped to minimum", below, 100, 20, 90, true, config.MinStackedListHeight},
		{"below, preview clamped to minimum", below, 100, 40, 10, true, 40 - footerHeight - config.MinStackedListHeight},
		{"below, wide terminal stays stacked", below, 200, 40, 50, true, 20},
		{"below, too short for both panes", below, 100, 10, 50, false, 0},
		{"auto ignores the ratio", auto, 100, 40, 90, true, config.DefaultStackedListHeight},
		{"right fallback ignores the ratio", right, TwoColumnMinWidth - 1, 40, 90, true, config.DefaultStackedListHeight},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := New(nil, config.Default())
			a.settings = tuiSettings{previewPosition: tt.position, previewPercent: tt.percent}
			a.Update(tea.WindowSizeMsg{Width: tt.width, Height: tt.height})
			a.Update(beansLoadedMsg{items: items, idColWidth: 12})

			if got := a.isStackedMode(); got != tt.wantStacked {
				t.Fatalf("isStackedMode() = %v, want %v", got, tt.wantStacked)
			}
			// The single-column layout keeps a spare row below the list.
			wantHeight := tt.height
			if !tt.wantStacked {
				wantHeight -= listBottomPadding
			}
			if lines := strings.Split(ansi.Strip(a.View()), "\n"); len(lines) != wantHeight {
				t.Errorf("view height = %d, want %d", len(lines), wantHeight)
			}
			if !tt.wantStacked {
				return
			}
			if got := a.stackedListPaneHeight(); got != tt.wantList {
				t.Errorf("list pane height = %d, want %d", got, tt.wantList)
			}
			wantPreview := tt.height - footerHeight - tt.wantList
			if _, got := a.previewSize(); got != wantPreview {
				t.Errorf("preview height = %d, want %d", got, wantPreview)
			}
		})
	}
}

func TestPreviewRatioKeptOnResize(t *testing.T) {
	items := []ui.FlatItem{{Bean: longBodyBean(50), Matched: true}}
	a := New(nil, config.Default())
	a.settings = tuiSettings{previewPosition: config.PreviewPositionBelow, previewPercent: 30}
	a.Update(tea.WindowSizeMsg{Width: 100, Height: 41})
	a.Update(beansLoadedMsg{items: items, idColWidth: 12})
	if got, want := a.stackedListPaneHeight(), 28; got != want {
		t.Fatalf("list pane height at 41 rows = %d, want %d", got, want)
	}

	a.Update(tea.WindowSizeMsg{Width: 100, Height: 21})
	if a.settings.previewPercent != 30 {
		t.Errorf("preview percent after resize = %d, want 30", a.settings.previewPercent)
	}
	if got, want := a.stackedListPaneHeight(), 14; got != want {
		t.Errorf("list pane height at 21 rows = %d, want %d", got, want)
	}
	if _, h := a.previewSize(); h != 21-footerHeight-14 {
		t.Errorf("preview height at 21 rows = %d, want %d", h, 21-footerHeight-14)
	}
	if lines := strings.Split(ansi.Strip(a.View()), "\n"); len(lines) != 21 {
		t.Errorf("view height = %d, want 21", len(lines))
	}
}

func TestTogglePreviewPosition(t *testing.T) {
	items := []ui.FlatItem{{Bean: longBodyBean(50), Matched: true}}
	stackedMin := config.DefaultStackedListHeight + StackedBelowListMinHeight
	v := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")}

	a := New(nil, config.Default())
	a.Update(tea.WindowSizeMsg{Width: TwoColumnFullWidth, Height: stackedMin})
	a.Update(beansLoadedMsg{items: items, idColWidth: 12})
	if !a.isTwoColumnMode() {
		t.Fatal("wide terminal should start with the preview on the right")
	}

	a.Update(v)
	if !a.isStackedMode() {
		t.Fatal("after v: preview should be below the list")
	}
	lines := strings.Split(ansi.Strip(a.View()), "\n")
	if got := len(lines); got != stackedMin {
		t.Errorf("view height = %d, want %d", got, stackedMin)
	}
	if !strings.Contains(strings.Join(lines[a.stackedListPaneHeight():], "\n"), "item 00") {
		t.Errorf("after v: preview body not below the list\n%s", strings.Join(lines, "\n"))
	}
	if _, h := a.listPaneSize(); a.list.list.Height() != h {
		t.Errorf("list height = %d, want %d fitted to the stacked pane", a.list.list.Height(), h)
	}

	a.Update(v)
	if !a.isTwoColumnMode() {
		t.Error("after second v: preview should be on the right again")
	}
}

func TestPreviewWheel(t *testing.T) {
	items := []ui.FlatItem{
		{Bean: longBodyBean(50), Matched: true},
		{Bean: &bean.Bean{ID: "beans-0002", Title: "B", Status: "todo", Type: "task"}, Matched: true},
	}
	wheelDown := func(x int) tea.MouseMsg {
		return tea.MouseMsg{X: x, Y: 5, Action: tea.MouseActionPress, Button: tea.MouseButtonWheelDown}
	}

	a := New(nil, config.Default())
	a.Update(tea.WindowSizeMsg{Width: 140, Height: 30})
	a.Update(beansLoadedMsg{items: items, idColWidth: 12})
	leftWidth, _ := calculatePaneWidths(140)

	a.Update(wheelDown(5))
	if a.preview.scroll != 0 || a.list.list.Index() != 0 {
		t.Errorf("wheel over list: preview scroll = %d, list index = %d, want both 0", a.preview.scroll, a.list.list.Index())
	}

	a.Update(wheelDown(leftWidth + 5))
	if a.preview.scroll != previewScrollStep {
		t.Errorf("wheel over preview: scroll = %d, want %d", a.preview.scroll, previewScrollStep)
	}

	a.Update(beansLoadedMsg{items: items, idColWidth: 12})
	if a.preview.scroll != previewScrollStep {
		t.Errorf("reload of the same bean: scroll = %d, want %d kept", a.preview.scroll, previewScrollStep)
	}
}

func TestPreviewPageKeys(t *testing.T) {
	var items []ui.FlatItem
	for i := range 100 {
		b := longBodyBean(50)
		b.ID = fmt.Sprintf("beans-%04d", i)
		items = append(items, ui.FlatItem{Bean: b, Matched: true})
	}
	pgDown := tea.KeyMsg{Type: tea.KeyPgDown}
	pgUp := tea.KeyMsg{Type: tea.KeyPgUp}

	tests := []struct {
		name          string
		width, height int
		wantScroll    bool // false: the keys page the list instead
	}{
		{"two columns", 140, 30, true},
		{"stacked", 100, 40, true},
		{"list only", 100, 30, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := New(nil, config.Default())
			a.Update(tea.WindowSizeMsg{Width: tt.width, Height: tt.height})
			a.Update(beansLoadedMsg{items: items, idColWidth: 12})

			a.Update(pgDown)
			a.Update(pgDown)
			if !tt.wantScroll {
				if a.list.list.Index() == 0 {
					t.Error("pgdown without preview: list index = 0, want a later page")
				}
				return
			}
			if a.list.list.Index() != 0 {
				t.Errorf("pgdown with preview: list index = %d, want 0", a.list.list.Index())
			}
			if a.preview.scroll != 2*previewScrollStep {
				t.Errorf("after 2x pgdown: scroll = %d, want %d", a.preview.scroll, 2*previewScrollStep)
			}
			a.Update(pgUp)
			if a.preview.scroll != previewScrollStep {
				t.Errorf("after pgup: scroll = %d, want %d", a.preview.scroll, previewScrollStep)
			}
			a.Update(pgUp)
			a.Update(pgUp)
			if a.preview.scroll != 0 {
				t.Errorf("pgup past the top: scroll = %d, want 0", a.preview.scroll)
			}
		})
	}
}

func TestPreviewViewWithProperties(t *testing.T) {
	b := &bean.Bean{
		ID:         "beans-test",
		Title:      "Bean with Properties",
		Status:     "todo",
		Type:       "task",
		Properties: map[string]string{"zeta": "z", "github_issue": "#135"},
		Body:       "Test body",
	}

	view := ansi.Strip(newPreviewModel(b, 60, 20).View())

	issue := strings.Index(view, "github_issue: #135")
	zeta := strings.Index(view, "zeta: z")
	if issue < 0 || zeta < 0 {
		t.Fatalf("preview should list both properties:\n%s", view)
	}
	if issue > zeta {
		t.Errorf("properties should be sorted by key:\n%s", view)
	}
}

func TestPropertyLines(t *testing.T) {
	got := propertyLines(map[string]string{"b": "multi\nline", "a": "1"})
	for i := range got {
		got[i] = ansi.Strip(got[i])
	}
	want := []string{"a: 1", "b: multi line"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("propertyLines() = %q, want %q", got, want)
	}
}
