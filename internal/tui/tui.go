package tui

import (
	"sort"

	tea "github.com/charmbracelet/bubbletea"
	"hmans.dev/beans/internal/beancore"
	"hmans.dev/beans/internal/config"
)

// viewState represents which view is currently active
type viewState int

const (
	viewList viewState = iota
	viewDetail
	viewTagPicker
)

// beansChangedMsg is sent when beans change on disk (via file watcher)
type beansChangedMsg struct{}

// openTagPickerMsg requests opening the tag picker
type openTagPickerMsg struct{}

// tagSelectedMsg is sent when a tag is selected from the picker
type tagSelectedMsg struct {
	tag string
}

// clearFilterMsg is sent to clear any active filter
type clearFilterMsg struct{}

// App is the main TUI application model
type App struct {
	state     viewState
	list      listModel
	detail    detailModel
	tagPicker tagPickerModel
	history   []detailModel // stack of previous detail views for back navigation
	core      *beancore.Core
	config    *config.Config
	width     int
	height    int
	program   *tea.Program // reference to program for sending messages from watcher

	// Key chord state - tracks partial key sequences like "g" waiting for "t"
	pendingKey string
}

// New creates a new TUI application
func New(core *beancore.Core, cfg *config.Config) *App {
	return &App{
		state:  viewList,
		core:   core,
		config: cfg,
		list:   newListModel(core, cfg),
	}
}

// Init initializes the application
func (a *App) Init() tea.Cmd {
	return a.list.Init()
}

// Update handles messages
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height

	case tea.KeyMsg:
		// Handle key chord sequences
		if a.state == viewList && a.list.list.FilterState() != 1 {
			if a.pendingKey == "g" {
				a.pendingKey = ""
				switch msg.String() {
				case "t":
					// "g t" - go to tags
					return a, func() tea.Msg { return openTagPickerMsg{} }
				default:
					// Invalid second key, ignore the chord
				}
				// Don't forward this key since it was part of a chord attempt
				return a, nil
			}

			// Start of potential chord
			if msg.String() == "g" {
				a.pendingKey = "g"
				return a, nil
			}
		}

		// Clear pending key on any other key press
		a.pendingKey = ""

		switch msg.String() {
		case "ctrl+c":
			return a, tea.Quit
		case "q":
			if a.state == viewDetail || a.state == viewTagPicker {
				return a, tea.Quit
			}
			// For list, only quit if not filtering
			if a.state == viewList && a.list.list.FilterState() != 1 {
				return a, tea.Quit
			}
		}

	case beansChangedMsg:
		// Beans changed on disk - refresh
		if a.state == viewDetail {
			// Try to reload the current bean
			if updatedBean, err := a.core.Get(a.detail.bean.ID); err != nil {
				// Bean was deleted - return to list
				a.state = viewList
				a.history = nil
			} else {
				// Recreate detail view with fresh bean data
				a.detail = newDetailModel(updatedBean, a.core, a.config, a.width, a.height)
			}
		}
		// Trigger list refresh
		return a, a.list.loadBeans

	case openTagPickerMsg:
		// Collect all unique tags from beans
		tags := a.collectAllTags()
		if len(tags) == 0 {
			// No tags in system, don't open picker
			return a, nil
		}
		a.tagPicker = newTagPickerModel(tags, a.width, a.height)
		a.state = viewTagPicker
		return a, a.tagPicker.Init()

	case tagSelectedMsg:
		a.state = viewList
		a.list.setTagFilter(msg.tag)
		return a, a.list.loadBeans

	case clearFilterMsg:
		a.list.clearFilter()
		return a, a.list.loadBeans

	case selectBeanMsg:
		// Push current detail view to history if we're already viewing a bean
		if a.state == viewDetail {
			a.history = append(a.history, a.detail)
		}
		a.state = viewDetail
		a.detail = newDetailModel(msg.bean, a.core, a.config, a.width, a.height)
		return a, a.detail.Init()

	case backToListMsg:
		// Pop from history if available, otherwise go to list
		if len(a.history) > 0 {
			a.detail = a.history[len(a.history)-1]
			a.history = a.history[:len(a.history)-1]
			// Stay in viewDetail state
		} else {
			a.state = viewList
			// Force list to pick up any size changes that happened while in detail view
			a.list, cmd = a.list.Update(tea.WindowSizeMsg{Width: a.width, Height: a.height})
			return a, cmd
		}
		return a, nil
	}

	// Forward all messages to the current view
	switch a.state {
	case viewList:
		a.list, cmd = a.list.Update(msg)
	case viewDetail:
		a.detail, cmd = a.detail.Update(msg)
	case viewTagPicker:
		a.tagPicker, cmd = a.tagPicker.Update(msg)
	}

	return a, cmd
}

// collectAllTags returns all unique tags across all beans, sorted alphabetically
func (a *App) collectAllTags() []string {
	tagSet := make(map[string]struct{})
	for _, b := range a.core.All() {
		for _, tag := range b.Tags {
			tagSet[tag] = struct{}{}
		}
	}

	tags := make([]string, 0, len(tagSet))
	for tag := range tagSet {
		tags = append(tags, tag)
	}

	sort.Strings(tags)
	return tags
}

// View renders the current view
func (a *App) View() string {
	switch a.state {
	case viewList:
		return a.list.View()
	case viewDetail:
		return a.detail.View()
	case viewTagPicker:
		return a.tagPicker.View()
	}
	return ""
}

// Run starts the TUI application with file watching
func Run(core *beancore.Core, cfg *config.Config) error {
	app := New(core, cfg)
	p := tea.NewProgram(app, tea.WithAltScreen())

	// Store reference to program for sending messages from watcher
	app.program = p

	// Start file watching
	if err := core.Watch(func() {
		// Send message to TUI when beans change
		if app.program != nil {
			app.program.Send(beansChangedMsg{})
		}
	}); err != nil {
		return err
	}
	defer core.Unwatch()

	_, err := p.Run()
	return err
}
