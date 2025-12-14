package tui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hmans/beans/internal/beancore"
	"github.com/hmans/beans/internal/config"
	"github.com/hmans/beans/internal/graph"
	"github.com/hmans/beans/internal/graph/model"
)

// viewState represents which view is currently active
type viewState int

const (
	viewList viewState = iota
	viewDetail
	viewTagPicker
	viewParentPicker
	viewStatusPicker
	viewTypePicker
	viewBlockingPicker
	viewPriorityPicker
	viewCreateModal
	viewHelpOverlay
	viewLauncherPicker
	viewLauncherError
	viewNoLaunchers
	viewConfigureLaunchers
	viewConfigWriteError
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

// openEditorMsg requests opening the editor for a bean
type openEditorMsg struct {
	beanID   string
	beanPath string
}

// editorFinishedMsg is sent when the editor closes
type editorFinishedMsg struct {
	err error
}

// openParentPickerMsg requests opening the parent picker for bean(s)
type openParentPickerMsg struct {
	beanIDs       []string // IDs of beans to update
	beanTitle     string   // Display title (single title or "N selected beans")
	beanTypes     []string // Types of the beans (to filter eligible parents)
	currentParent string   // Only meaningful for single bean
}

// App is the main TUI application model
type App struct {
	state              viewState
	list               listModel
	detail             detailModel
	tagPicker          tagPickerModel
	parentPicker       parentPickerModel
	statusPicker       statusPickerModel
	typePicker         typePickerModel
	blockingPicker     blockingPickerModel
	priorityPicker     priorityPickerModel
	createModal        createModalModel
	helpOverlay        helpOverlayModel
	launcherPicker     launcherPickerModel
	launcherError      launcherErrorModel
	noLaunchers        noLaunchersModel
	configureLaunchers configureLaunchersModel
	configWriteError   configWriteErrorModel
	history            []detailModel // stack of previous detail views for back navigation
	core               *beancore.Core
	resolver           *graph.Resolver
	config             *config.Config
	width              int
	height             int
	program            *tea.Program // reference to program for sending messages from watcher

	// Key chord state - tracks partial key sequences like "g" waiting for "t"
	pendingKey string

	// Modal state - tracks view behind modal pickers
	previousState viewState

	// Editor state - tracks bean being edited to update updated_at on save
	editingBeanID      string
	editingBeanModTime time.Time
}

// New creates a new TUI application
func New(core *beancore.Core, cfg *config.Config) *App {
	resolver := &graph.Resolver{Core: core}
	return &App{
		state:    viewList,
		core:     core,
		resolver: resolver,
		config:   cfg,
		list:     newListModel(resolver, cfg),
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
		case "?":
			// Open help overlay if not already showing it (and not in a picker/modal)
			if a.state == viewList || a.state == viewDetail {
				a.previousState = a.state
				a.helpOverlay = newHelpOverlayModel(a.width, a.height)
				a.state = viewHelpOverlay
				return a, a.helpOverlay.Init()
			}
		case "q":
			if a.state == viewDetail || a.state == viewTagPicker || a.state == viewParentPicker || a.state == viewStatusPicker || a.state == viewTypePicker || a.state == viewBlockingPicker || a.state == viewPriorityPicker || a.state == viewCreateModal || a.state == viewHelpOverlay {
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
			// Try to reload the current bean via GraphQL
			updatedBean, err := a.resolver.Query().Bean(context.Background(), a.detail.bean.ID)
			if err != nil || updatedBean == nil {
				// Bean was deleted - return to list
				a.state = viewList
				a.history = nil
			} else {
				// Recreate detail view with fresh bean data
				a.detail = newDetailModel(updatedBean, a.resolver, a.config, a.width, a.height)
			}
		}
		// Trigger list refresh
		return a, a.list.loadBeans

	case openTagPickerMsg:
		// Collect all tags with their counts
		tags := a.collectTagsWithCounts()
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

	case openParentPickerMsg:
		// Check if all bean types can have parents
		for _, beanType := range msg.beanTypes {
			if beancore.ValidParentTypes(beanType) == nil {
				// At least one bean type (e.g., milestone) cannot have parents - don't open the picker
				return a, nil
			}
		}
		a.previousState = a.state // Remember where we came from for the modal background
		a.parentPicker = newParentPickerModel(msg.beanIDs, msg.beanTitle, msg.beanTypes, msg.currentParent, a.resolver, a.config, a.width, a.height)
		a.state = viewParentPicker
		return a, a.parentPicker.Init()

	case closeParentPickerMsg:
		// Return to previous view and refresh in case beans changed while picker was open
		a.state = a.previousState
		return a, a.list.loadBeans

	case openStatusPickerMsg:
		a.previousState = a.state
		a.statusPicker = newStatusPickerModel(msg.beanIDs, msg.beanTitle, msg.currentStatus, a.config, a.width, a.height)
		a.state = viewStatusPicker
		return a, a.statusPicker.Init()

	case closeStatusPickerMsg:
		// Return to previous view and refresh in case beans changed while picker was open
		a.state = a.previousState
		return a, a.list.loadBeans

	case statusSelectedMsg:
		// Update all beans' status via GraphQL mutations
		for _, beanID := range msg.beanIDs {
			_, err := a.resolver.Mutation().UpdateBean(context.Background(), beanID, model.UpdateBeanInput{
				Status: &msg.status,
			})
			if err != nil {
				// Continue with other beans even if one fails
				continue
			}
		}
		// Return to the previous view and refresh
		a.state = a.previousState
		// Clear selection after batch edit
		clear(a.list.selectedBeans)
		if a.state == viewDetail && len(msg.beanIDs) == 1 {
			updatedBean, _ := a.resolver.Query().Bean(context.Background(), msg.beanIDs[0])
			if updatedBean != nil {
				a.detail = newDetailModel(updatedBean, a.resolver, a.config, a.width, a.height)
			}
		}
		return a, a.list.loadBeans

	case openTypePickerMsg:
		a.previousState = a.state
		a.typePicker = newTypePickerModel(msg.beanIDs, msg.beanTitle, msg.currentType, a.config, a.width, a.height)
		a.state = viewTypePicker
		return a, a.typePicker.Init()

	case closeTypePickerMsg:
		// Return to previous view and refresh in case beans changed while picker was open
		a.state = a.previousState
		return a, a.list.loadBeans

	case typeSelectedMsg:
		// Update all beans' type via GraphQL mutations
		for _, beanID := range msg.beanIDs {
			_, err := a.resolver.Mutation().UpdateBean(context.Background(), beanID, model.UpdateBeanInput{
				Type: &msg.beanType,
			})
			if err != nil {
				// Continue with other beans even if one fails
				continue
			}
		}
		// Return to the previous view and refresh
		a.state = a.previousState
		// Clear selection after batch edit
		clear(a.list.selectedBeans)
		if a.state == viewDetail && len(msg.beanIDs) == 1 {
			updatedBean, _ := a.resolver.Query().Bean(context.Background(), msg.beanIDs[0])
			if updatedBean != nil {
				a.detail = newDetailModel(updatedBean, a.resolver, a.config, a.width, a.height)
			}
		}
		return a, a.list.loadBeans

	case openPriorityPickerMsg:
		a.previousState = a.state
		a.priorityPicker = newPriorityPickerModel(msg.beanIDs, msg.beanTitle, msg.currentPriority, a.config, a.width, a.height)
		a.state = viewPriorityPicker
		return a, a.priorityPicker.Init()

	case closePriorityPickerMsg:
		// Return to previous view and refresh in case beans changed while picker was open
		a.state = a.previousState
		return a, a.list.loadBeans

	case prioritySelectedMsg:
		// Update all beans' priority via GraphQL mutations
		for _, beanID := range msg.beanIDs {
			_, err := a.resolver.Mutation().UpdateBean(context.Background(), beanID, model.UpdateBeanInput{
				Priority: &msg.priority,
			})
			if err != nil {
				// Continue with other beans even if one fails
				continue
			}
		}
		// Return to the previous view and refresh
		a.state = a.previousState
		// Clear selection after batch edit
		clear(a.list.selectedBeans)
		if a.state == viewDetail && len(msg.beanIDs) == 1 {
			updatedBean, _ := a.resolver.Query().Bean(context.Background(), msg.beanIDs[0])
			if updatedBean != nil {
				a.detail = newDetailModel(updatedBean, a.resolver, a.config, a.width, a.height)
			}
		}
		return a, a.list.loadBeans

	case openHelpMsg:
		a.previousState = a.state
		a.helpOverlay = newHelpOverlayModel(a.width, a.height)
		a.state = viewHelpOverlay
		return a, a.helpOverlay.Init()

	case closeHelpMsg:
		a.state = a.previousState
		return a, nil

	case openBlockingPickerMsg:
		a.previousState = a.state
		a.blockingPicker = newBlockingPickerModel(msg.beanID, msg.beanTitle, msg.currentBlocking, a.resolver, a.config, a.width, a.height)
		a.state = viewBlockingPicker
		return a, a.blockingPicker.Init()

	case closeBlockingPickerMsg:
		// Return to previous view and refresh in case beans changed while picker was open
		a.state = a.previousState
		return a, a.list.loadBeans

	case blockingConfirmedMsg:
		// Apply all blocking changes via GraphQL mutations
		for _, targetID := range msg.toAdd {
			_, err := a.resolver.Mutation().AddBlocking(context.Background(), msg.beanID, targetID)
			if err != nil {
				// Continue with other changes even if one fails
				continue
			}
		}
		for _, targetID := range msg.toRemove {
			_, err := a.resolver.Mutation().RemoveBlocking(context.Background(), msg.beanID, targetID)
			if err != nil {
				// Continue with other changes even if one fails
				continue
			}
		}
		// Return to previous view and refresh
		a.state = a.previousState
		if a.state == viewDetail {
			updatedBean, _ := a.resolver.Query().Bean(context.Background(), msg.beanID)
			if updatedBean != nil {
				a.detail = newDetailModel(updatedBean, a.resolver, a.config, a.width, a.height)
			}
		}
		return a, a.list.loadBeans

	case openCreateModalMsg:
		a.previousState = a.state
		a.createModal = newCreateModalModel(a.width, a.height)
		a.state = viewCreateModal
		return a, a.createModal.Init()

	case closeCreateModalMsg:
		a.state = a.previousState
		return a, nil

	case beanCreatedMsg:
		// Create the bean via GraphQL mutation with draft status
		draftStatus := "draft"
		createdBean, err := a.resolver.Mutation().CreateBean(context.Background(), model.CreateBeanInput{
			Title:  msg.title,
			Status: &draftStatus,
		})
		if err != nil {
			// TODO: Show error to user
			a.state = a.previousState
			return a, nil
		}
		// Return to list and open the new bean in editor
		a.state = viewList
		return a, tea.Batch(
			a.list.loadBeans,
			func() tea.Msg {
				return openEditorMsg{beanID: createdBean.ID, beanPath: createdBean.Path}
			},
		)

	case openEditorMsg:
		// Launch editor for the bean file
		editor := getEditor()
		fullPath := filepath.Join(a.core.Root(), msg.beanPath)

		// Record the bean ID and file mod time before editing
		a.editingBeanID = msg.beanID
		if info, err := os.Stat(fullPath); err == nil {
			a.editingBeanModTime = info.ModTime()
		}

		c := exec.Command(editor, fullPath)
		return a, tea.ExecProcess(c, func(err error) tea.Msg {
			return editorFinishedMsg{err: err}
		})

	case editorFinishedMsg:
		// Editor closed - check if file was modified and update updated_at if so
		if a.editingBeanID != "" {
			if b, err := a.core.Get(a.editingBeanID); err == nil {
				fullPath := filepath.Join(a.core.Root(), b.Path)
				if info, err := os.Stat(fullPath); err == nil {
					if info.ModTime().After(a.editingBeanModTime) {
						// File was modified - reload from disk first to get user's changes,
						// then call Update to set updated_at
						_ = a.core.Load()
						if b, err = a.core.Get(a.editingBeanID); err == nil {
							_ = a.core.Update(b)
						}
					}
				}
			}
			// Clear editing state
			a.editingBeanID = ""
			a.editingBeanModTime = time.Time{}
		}
		return a, nil

	case parentSelectedMsg:
		// Set the new parent via GraphQL mutation for all beans
		var parentID *string
		if msg.parentID != "" {
			parentID = &msg.parentID
		}
		for _, beanID := range msg.beanIDs {
			_, err := a.resolver.Mutation().SetParent(context.Background(), beanID, parentID)
			if err != nil {
				// Continue with other beans even if one fails
				continue
			}
		}
		// Return to the previous view and refresh
		a.state = a.previousState
		// Clear selection after batch edit
		clear(a.list.selectedBeans)
		if a.state == viewDetail && len(msg.beanIDs) == 1 {
			// Refresh the bean to show updated parent
			updatedBean, _ := a.resolver.Query().Bean(context.Background(), msg.beanIDs[0])
			if updatedBean != nil {
				a.detail = newDetailModel(updatedBean, a.resolver, a.config, a.width, a.height)
			}
		}
		return a, a.list.loadBeans

	case openLauncherPickerMsg:
		// Discover available launchers
		launchers := discoverLaunchers(a.config, a.core.Root())

		if len(launchers) == 0 {
			// Check if any launchers are configured at all
			if !hasLaunchersConfigured(a.config) {
				// First time - offer to configure defaults
				a.previousState = a.state
				a.configureLaunchers = newConfigureLaunchersModel(msg.beanID, msg.beanTitle, a.width, a.height)
				a.state = viewConfigureLaunchers
				return a, a.configureLaunchers.Init()
			}

			// Launchers configured but none available
			a.previousState = a.state
			a.noLaunchers = newNoLaunchersModel(a.width, a.height)
			a.state = viewNoLaunchers
			return a, nil
		}

		// Open launcher picker
		a.previousState = a.state
		a.launcherPicker = newLauncherPickerModel(launchers, msg.beanID, msg.beanTitle, a.width, a.height)
		a.state = viewLauncherPicker
		return a, a.launcherPicker.Init()

	case launcherSelectedMsg:
		// Close the picker before launching
		a.state = a.previousState

		// Create command (handles both single-line and multi-line exec)
		beanPath := filepath.Join(a.core.Root(), ".beans", "beans-"+msg.beanID+".md")
		result := createExecCommand(msg.launcher.exec, msg.beanID, a.core.Root(), beanPath)

		// Store launcher name and cleanup function for callback
		launcherName := msg.launcher.name
		cleanup := result.cleanup

		return a, tea.ExecProcess(result.cmd, func(err error) tea.Msg {
			// Clean up temp file if created
			if cleanup != nil {
				cleanup()
			}
			return launcherFinishedMsg{
				err:          err,
				launcherName: launcherName,
			}
		})

	case launcherFinishedMsg:
		if msg.err != nil {
			// Show error modal
			a.previousState = viewDetail
			a.launcherError = newLauncherErrorModel(msg.launcherName, msg.err, a.width, a.height)
			a.state = viewLauncherError
			return a, nil
		}
		// Success - already back in detail view
		return a, nil

	case closeLauncherPickerMsg:
		a.state = a.previousState
		return a, nil

	case launchersConfiguredMsg:
		// Append selected launchers to .beans.yml
		projectRoot := a.config.ConfigDir()
		if projectRoot == "" {
			// Fallback: try to get parent of beans directory
			projectRoot = filepath.Dir(a.core.Root())
		}

		if err := appendLaunchersToConfig(projectRoot, msg.launchers); err != nil {
			// Show error modal
			a.previousState = viewDetail
			a.configWriteError = newConfigWriteErrorModel(err, a.width, a.height)
			a.state = viewConfigWriteError
			return a, nil
		}

		// Reload config to pick up the new launchers
		if newCfg, err := config.LoadFromDirectory(projectRoot); err == nil {
			a.config = newCfg
		}

		// Return to previous state (detail view)
		a.state = a.previousState
		return a, nil

	case closeConfigureLaunchersMsg:
		// User cancelled - return to previous state
		a.state = a.previousState
		return a, nil

	case clearFilterMsg:
		a.list.clearFilter()
		return a, a.list.loadBeans

	case selectBeanMsg:
		// Push current detail view to history if we're already viewing a bean
		if a.state == viewDetail {
			a.history = append(a.history, a.detail)
		}
		a.state = viewDetail
		a.detail = newDetailModel(msg.bean, a.resolver, a.config, a.width, a.height)
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
	case viewParentPicker:
		a.parentPicker, cmd = a.parentPicker.Update(msg)
	case viewStatusPicker:
		a.statusPicker, cmd = a.statusPicker.Update(msg)
	case viewTypePicker:
		a.typePicker, cmd = a.typePicker.Update(msg)
	case viewPriorityPicker:
		a.priorityPicker, cmd = a.priorityPicker.Update(msg)
	case viewBlockingPicker:
		a.blockingPicker, cmd = a.blockingPicker.Update(msg)
	case viewCreateModal:
		a.createModal, cmd = a.createModal.Update(msg)
	case viewHelpOverlay:
		a.helpOverlay, cmd = a.helpOverlay.Update(msg)
	case viewLauncherPicker:
		a.launcherPicker, cmd = a.launcherPicker.Update(msg)
	case viewLauncherError:
		// Any key dismisses error modal
		if _, ok := msg.(tea.KeyMsg); ok {
			a.state = a.previousState
			return a, nil
		}
	case viewNoLaunchers:
		// Any key dismisses no-launchers modal
		if _, ok := msg.(tea.KeyMsg); ok {
			a.state = a.previousState
			return a, nil
		}
	case viewConfigureLaunchers:
		a.configureLaunchers, cmd = a.configureLaunchers.Update(msg)
	case viewConfigWriteError:
		// Any key dismisses error modal
		if _, ok := msg.(tea.KeyMsg); ok {
			a.state = a.previousState
			return a, nil
		}
	}

	return a, cmd
}

// collectTagsWithCounts returns all tags with their usage counts
func (a *App) collectTagsWithCounts() []tagWithCount {
	beans, _ := a.resolver.Query().Beans(context.Background(), nil)
	tagCounts := make(map[string]int)
	for _, b := range beans {
		for _, tag := range b.Tags {
			tagCounts[tag]++
		}
	}

	tags := make([]tagWithCount, 0, len(tagCounts))
	for tag, count := range tagCounts {
		tags = append(tags, tagWithCount{tag: tag, count: count})
	}

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
	case viewParentPicker:
		return a.parentPicker.ModalView(a.getBackgroundView(), a.width, a.height)
	case viewStatusPicker:
		return a.statusPicker.ModalView(a.getBackgroundView(), a.width, a.height)
	case viewTypePicker:
		return a.typePicker.ModalView(a.getBackgroundView(), a.width, a.height)
	case viewPriorityPicker:
		return a.priorityPicker.ModalView(a.getBackgroundView(), a.width, a.height)
	case viewBlockingPicker:
		return a.blockingPicker.ModalView(a.getBackgroundView(), a.width, a.height)
	case viewCreateModal:
		return a.createModal.ModalView(a.getBackgroundView(), a.width, a.height)
	case viewHelpOverlay:
		return a.helpOverlay.ModalView(a.getBackgroundView(), a.width, a.height)
	case viewLauncherPicker:
		return a.launcherPicker.ModalView(a.getBackgroundView(), a.width, a.height)
	case viewLauncherError:
		return a.launcherError.ModalView(a.getBackgroundView(), a.width, a.height)
	case viewNoLaunchers:
		return a.noLaunchers.ModalView(a.getBackgroundView(), a.width, a.height)
	case viewConfigureLaunchers:
		return a.configureLaunchers.ModalView(a.getBackgroundView(), a.width, a.height)
	case viewConfigWriteError:
		return a.configWriteError.ModalView(a.getBackgroundView(), a.width, a.height)
	}
	return ""
}

// getBackgroundView returns the view to show behind modal pickers
func (a *App) getBackgroundView() string {
	switch a.previousState {
	case viewList:
		return a.list.View()
	case viewDetail:
		return a.detail.View()
	default:
		return a.list.View()
	}
}

// execResult holds a command and its cleanup function
type execResult struct {
	cmd     *exec.Cmd
	cleanup func()
}

// createExecCommand creates a command for executing an exec script.
// For multi-line scripts, it creates a temp file and returns cleanup function.
// For single-line scripts, it executes via sh -c.
func createExecCommand(execScript, beanID, beansRoot, beanPath string) execResult {
	// Check if this is multi-line (contains newline)
	if !strings.Contains(execScript, "\n") {
		// Single-line: execute via sh -c
		cmd := exec.Command("sh", "-c", execScript)
		cmd.Env = append(os.Environ(),
			fmt.Sprintf("BEANS_ROOT=%s", beansRoot),
			fmt.Sprintf("BEANS_ID=%s", beanID),
			fmt.Sprintf("BEANS_TASK=%s", beanPath),
		)
		cmd.Dir = beansRoot
		return execResult{cmd: cmd, cleanup: func() {}}
	}

	// Multi-line: create temp file
	tmpFile, err := os.CreateTemp("", "beans-launcher-*.sh")
	if err != nil {
		// Return command that will fail with good error message
		cmd := exec.Command("sh", "-c", fmt.Sprintf("echo 'Error: failed to create temp script: %v' >&2 && exit 1", err))
		return execResult{cmd: cmd, cleanup: func() {}}
	}

	tmpPath := tmpFile.Name()

	// Write script content
	if _, err := tmpFile.WriteString(execScript); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		cmd := exec.Command("sh", "-c", fmt.Sprintf("echo 'Error: failed to write script: %v' >&2 && exit 1", err))
		return execResult{cmd: cmd, cleanup: func() {}}
	}
	tmpFile.Close()

	// Make executable
	if err := os.Chmod(tmpPath, 0755); err != nil {
		os.Remove(tmpPath)
		cmd := exec.Command("sh", "-c", fmt.Sprintf("echo 'Error: failed to make script executable: %v' >&2 && exit 1", err))
		return execResult{cmd: cmd, cleanup: func() {}}
	}

	// Create command to execute temp file
	cmd := exec.Command(tmpPath)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("BEANS_ROOT=%s", beansRoot),
		fmt.Sprintf("BEANS_ID=%s", beanID),
		fmt.Sprintf("BEANS_TASK=%s", beanPath),
	)
	cmd.Dir = beansRoot

	// Return with cleanup function
	cleanup := func() {
		os.Remove(tmpPath)
	}

	return execResult{cmd: cmd, cleanup: cleanup}
}

// getEditor returns the user's preferred editor using the fallback chain:
// $VISUAL -> $EDITOR -> vi -> nano
func getEditor() string {
	if editor := os.Getenv("VISUAL"); editor != "" {
		return editor
	}
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}
	// Fallback chain: vi is more universal, nano as last resort
	if _, err := exec.LookPath("vi"); err == nil {
		return "vi"
	}
	return "nano"
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
