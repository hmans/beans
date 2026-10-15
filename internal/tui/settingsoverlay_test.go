package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/hmans/beans/internal/ui"
	"github.com/hmans/beans/pkg/bean"
	"github.com/hmans/beans/pkg/beancore"
	"github.com/hmans/beans/pkg/config"
)

// sendKey feeds a key to the app and then the message its command produces,
// unless that message is a list reload, which needs a bean core.
func sendKey(a *App, key tea.KeyMsg) tea.Cmd {
	_, cmd := a.Update(key)
	if cmd == nil {
		return nil
	}
	switch msg := cmd().(type) {
	case settingsChangedMsg, closeSettingsMsg, closeHelpMsg:
		_, cmd = a.Update(msg)
		return cmd
	}
	return cmd
}

func runes(s string) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)} }

func TestSettingsOverlay(t *testing.T) {
	items := []ui.FlatItem{{Bean: longBodyBean(50), Matched: true}}
	cfg := config.Default()
	cfg.TUI.PreviewPosition = config.PreviewPositionBelow
	a := New(nil, cfg)
	a.Update(tea.WindowSizeMsg{Width: 100, Height: 41})
	a.Update(beansLoadedMsg{items: items, idColWidth: 12})

	sendKey(a, runes(","))
	if a.state != viewSettingsOverlay {
		t.Fatalf("after ',': state = %v, want settings overlay", a.state)
	}
	view := ansi.Strip(a.View())
	if !strings.Contains(view, "preview ratio") || !strings.Contains(view, "item 00") {
		t.Errorf("overlay should show above the stacked layout:\n%s", view)
	}

	sendKey(a, tea.KeyMsg{Type: tea.KeyDown})
	sendKey(a, tea.KeyMsg{Type: tea.KeyRight})
	if a.settings.previewPercent != DefaultPreviewPercent+previewPercentStep {
		t.Errorf("after ↓ →: preview percent = %d, want %d", a.settings.previewPercent, DefaultPreviewPercent+previewPercentStep)
	}
	if got, want := a.stackedListPaneHeight(), 16; got != want {
		t.Errorf("list pane height = %d, want %d", got, want)
	}
	if _, h := a.listPaneSize(); a.list.list.Height() != h {
		t.Errorf("list height = %d, want %d fitted to the pane", a.list.list.Height(), h)
	}

	sendKey(a, tea.KeyMsg{Type: tea.KeyDown})
	if cmd := sendKey(a, runes("l")); cmd == nil {
		t.Error("view mode change should reload the list")
	}
	if a.list.viewMode != viewActive {
		t.Errorf("after l on view mode: viewMode = %v, want %v", a.list.viewMode, viewActive)
	}

	sendKey(a, tea.KeyMsg{Type: tea.KeyEsc})
	if a.state != viewList {
		t.Fatalf("after esc: state = %v, want list", a.state)
	}
	if lines := strings.Split(ansi.Strip(a.View()), "\n"); len(lines) != 41 {
		t.Errorf("view height = %d, want 41", len(lines))
	}
}

func TestSettingsOverlayValues(t *testing.T) {
	m := newSettingsOverlayModel(tuiSettings{previewPosition: config.PreviewPositionAuto, previewPercent: MaxPreviewPercent}, viewAll, 100, 40)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if m.settings.previewPosition != config.PreviewPositionBelow {
		t.Errorf("← on auto: position = %q, want below (wraps)", m.settings.previewPosition)
	}

	m, _ = m.Update(runes("j"))
	m, _ = m.Update(runes("l"))
	if m.settings.previewPercent != MaxPreviewPercent {
		t.Errorf("→ at the maximum: percent = %d, want %d", m.settings.previewPercent, MaxPreviewPercent)
	}
	for range 10 {
		m, _ = m.Update(runes("h"))
	}
	if m.settings.previewPercent != MinPreviewPercent {
		t.Errorf("← past the minimum: percent = %d, want %d", m.settings.previewPercent, MinPreviewPercent)
	}

	m, _ = m.Update(runes("k"))
	m, _ = m.Update(runes("k"))
	if m.cursor != settingViewMode {
		t.Fatalf("k twice from the ratio row: cursor = %d, want view mode row (wraps)", m.cursor)
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if m.viewMode != viewReady {
		t.Errorf("← on all: viewMode = %v, want ready (wraps)", m.viewMode)
	}
}

func TestSettingsOverlayOpening(t *testing.T) {
	a := New(nil, config.Default())
	a.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	a.Update(beansLoadedMsg{items: []ui.FlatItem{{Bean: longBodyBean(5), Matched: true}}, idColWidth: 12})

	sendKey(a, runes("/"))
	sendKey(a, runes(","))
	if a.state != viewList {
		t.Errorf("',' while typing a filter: state = %v, want list", a.state)
	}
	sendKey(a, tea.KeyMsg{Type: tea.KeyEsc})

	sendKey(a, runes("?"))
	sendKey(a, runes(","))
	if a.state != viewHelpOverlay {
		t.Errorf("',' in the help overlay: state = %v, want help overlay", a.state)
	}

	sendKey(a, tea.KeyMsg{Type: tea.KeyEsc})
	sendKey(a, runes(","))
	sendKey(a, runes(","))
	if a.state != viewList {
		t.Errorf("',' twice: state = %v, want list", a.state)
	}

	sendKey(a, runes(","))
	if _, cmd := a.Update(runes("q")); cmd == nil || cmd() != tea.Quit() {
		t.Error("q in the settings overlay should quit")
	}
}

func TestSettingsOverlayFromDetail(t *testing.T) {
	beansDir := filepath.Join(t.TempDir(), ".beans")
	if err := os.MkdirAll(beansDir, 0755); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	core := beancore.New(beansDir, cfg)
	if err := core.Load(); err != nil {
		t.Fatal(err)
	}
	b := &bean.Bean{ID: "one", Slug: "one", Title: "One", Status: "todo", Type: "task"}
	if err := core.Create(b); err != nil {
		t.Fatal(err)
	}
	a := New(core, cfg)
	a.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	a.Update(selectBeanMsg{bean: b})

	sendKey(a, runes(","))
	if a.state != viewSettingsOverlay {
		t.Fatalf("',' in the detail view: state = %v, want settings overlay", a.state)
	}
	a.Update(tea.WindowSizeMsg{Width: 100, Height: 20})
	sendKey(a, tea.KeyMsg{Type: tea.KeyEsc})
	if a.state != viewDetail {
		t.Fatalf("esc: state = %v, want back in the detail view", a.state)
	}
	if lines := strings.Split(ansi.Strip(a.View()), "\n"); len(lines) > 20 {
		t.Errorf("detail view height after resize behind the overlay = %d, want <= 20", len(lines))
	}
}

func TestResizeBehindModal(t *testing.T) {
	items := []ui.FlatItem{{Bean: longBodyBean(50), Matched: true}}
	for _, open := range []string{",", "?"} {
		t.Run(open, func(t *testing.T) {
			a := New(nil, config.Default())
			a.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
			a.Update(beansLoadedMsg{items: items, idColWidth: 12})

			sendKey(a, runes(open))
			a.Update(tea.WindowSizeMsg{Width: 100, Height: 20})
			if lines := strings.Split(ansi.Strip(a.View()), "\n"); len(lines) != 20 {
				t.Errorf("modal view height = %d, want 20", len(lines))
			}
			sendKey(a, tea.KeyMsg{Type: tea.KeyEsc})
			if a.state != viewList {
				t.Fatalf("after esc: state = %v, want list", a.state)
			}
			if lines := strings.Split(ansi.Strip(a.View()), "\n"); len(lines) != 20-listBottomPadding {
				t.Errorf("list view height after resize behind the modal = %d, want %d", len(lines), 20-listBottomPadding)
			}
		})
	}
}

func TestModalBackgroundShowsLayout(t *testing.T) {
	items := []ui.FlatItem{{Bean: longBodyBean(50), Matched: true}}
	a := New(nil, config.Default())
	a.Update(tea.WindowSizeMsg{Width: TwoColumnFullWidth, Height: 40})
	a.Update(beansLoadedMsg{items: items, idColWidth: 12})
	if !a.isTwoColumnMode() {
		t.Fatal("wide terminal should show two columns")
	}

	sendKey(a, runes("?"))
	if view := ansi.Strip(a.View()); !strings.Contains(view, "item 00") {
		t.Errorf("help overlay should show the preview behind it:\n%s", view)
	}
}
