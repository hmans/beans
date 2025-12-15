package tui

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/beancore"
	"github.com/hmans/beans/internal/config"
	"github.com/hmans/beans/internal/graph"
	"github.com/hmans/beans/internal/graph/model"
	launcherexec "github.com/hmans/beans/internal/launcher"
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
	viewNoLaunchersMulti
	viewConfigureLaunchers
	viewConfigWriteError
	viewLaunchConfirm
	viewLaunchProgress
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
	noLaunchersMulti   noLaunchersMultiModel
	configureLaunchers configureLaunchersModel
	configWriteError   configWriteErrorModel
	launchConfirm      launchConfirm
	launchProgress     launchProgress
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
			if a.state == viewDetail || a.state == viewTagPicker || a.state == viewParentPicker || a.state == viewStatusPicker || a.state == viewTypePicker || a.state == viewBlockingPicker || a.state == viewPriorityPicker || a.state == viewHelpOverlay {
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
		// Discover available launchers based on bean count
		var launchers []launcher

		if len(msg.beanIDs) > 1 {
			// Multiple beans - only show launchers with multiple: true
			launchers = discoverLaunchersForMultiple(a.config, a.core.Root())
		} else {
			// Single bean - show all launchers
			launchers = discoverLaunchers(a.config, a.core.Root())
		}

		if len(launchers) == 0 {
			// Check if this is multi-bean with no multiple-capable launchers
			if len(msg.beanIDs) > 1 {
				// Show specific error for multi-bean
				a.previousState = a.state
				a.noLaunchersMulti = newNoLaunchersMultiModel(a.width, a.height)
				a.state = viewNoLaunchersMulti
				return a, nil
			}

			// Check if any launchers are configured at all
			if !hasLaunchersConfigured(a.config) {
				// First time - offer to configure defaults
				// For configure dialog, just show the first bean ID
				firstBeanID := ""
				if len(msg.beanIDs) > 0 {
					firstBeanID = msg.beanIDs[0]
				}
				a.previousState = a.state
				a.configureLaunchers = newConfigureLaunchersModel(firstBeanID, msg.beanTitle, a.width, a.height)
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
		a.launcherPicker = newLauncherPickerModel(launchers, msg.beanIDs, msg.beanTitle, a.width, a.height)
		a.state = viewLauncherPicker
		return a, a.launcherPicker.Init()

	case launcherSelectedMsg:
		// Check if single bean or multiple beans
		if len(msg.beanIDs) == 1 {
			// Single bean - use original ExecProcess approach
			a.state = a.previousState

			// Create command using launcher package (handles both single-line and multi-line exec)
			beanID := msg.beanIDs[0]
			beansDir := filepath.Join(a.core.Root(), ".beans")
			cmd, result, err := launcherexec.CreateExecCommand(msg.launcher.exec, beansDir, beanID, msg.beanTitle)

			if err != nil {
				return a, func() tea.Msg {
					return launcherFinishedMsg{
						err:          err,
						launcherName: msg.launcher.name,
					}
				}
			}

			// Store launcher name and cleanup function for callback
			launcherName := msg.launcher.name
			cleanup := result.Cleanup

			return a, tea.ExecProcess(cmd, func(err error) tea.Msg {
				// Clean up temp file if created
				if cleanup != nil {
					cleanup()
				}
				return launcherFinishedMsg{
					err:          err,
					launcherName: launcherName,
				}
			})
		}

		// Multiple beans - check if we need confirmation
		needsConfirm := len(msg.beanIDs) >= 5 && !a.config.TUI.DisableLauncherWarning

		if needsConfirm {
			// Show confirmation dialog
			a.previousState = a.state

			// Convert launcher to config.Launcher
			configLauncher := &config.Launcher{
				Name:        msg.launcher.name,
				Exec:        msg.launcher.exec,
				Description: msg.launcher.description,
			}

			a.launchConfirm = newLaunchConfirm(configLauncher, msg.beanIDs)
			a.state = viewLaunchConfirm
			return a, a.launchConfirm.Init()
		}

		// No confirmation needed - launch directly
		return a, func() tea.Msg {
			// Convert launcher to config.Launcher
			configLauncher := &config.Launcher{
				Name:        msg.launcher.name,
				Exec:        msg.launcher.exec,
				Description: msg.launcher.description,
			}

			return launchConfirmedMsg{
				launcher: configLauncher,
				beanIDs:  msg.beanIDs,
			}
		}

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

	case launchConfirmedMsg:
		// User confirmed multi-bean launch or skipped confirmation
		// Fetch all beans
		var beans []*bean.Bean
		for _, beanID := range msg.beanIDs {
			b, err := a.core.Get(beanID)
			if err != nil {
				// Skip beans that can't be loaded
				continue
			}
			beans = append(beans, b)
		}

		if len(beans) == 0 {
			// All beans failed to load - return to previous state
			a.state = a.previousState
			return a, nil
		}

		// Create launch manager
		manager := launcherexec.NewLaunchManager(msg.launcher, beans)

		// Start all launches
		if err := manager.Start(a.core.Root()); err != nil {
			// Failed to start - show error and return
			a.previousState = viewDetail
			a.launcherError = newLauncherErrorModel(msg.launcher.Name, err, a.width, a.height)
			a.state = viewLauncherError
			return a, nil
		}

		// Show progress view
		a.previousState = a.state
		a.launchProgress = newLaunchProgress(manager, msg.launcher.Name)
		a.state = viewLaunchProgress
		return a, a.launchProgress.Init()

	case launchCancelledMsg:
		// User cancelled confirmation
		a.state = a.previousState
		return a, nil

	case launchCompleteMsg:
		// All launches completed successfully
		a.state = a.previousState

		// Clear selection if we're in list view
		if a.state == viewList {
			clear(a.list.selectedBeans)
		}

		// Cleanup the progress view
		a.launchProgress.Cleanup()

		return a, nil

	case launchFailedMsg:
		// A launch failed - cleanup and show error
		a.launchProgress.Cleanup()

		// Show error modal
		a.previousState = viewDetail
		a.launcherError = newLauncherErrorModel("launcher", msg.err, a.width, a.height)
		a.state = viewLauncherError

		// Note: We don't clear selection so user can retry
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
	case viewNoLaunchersMulti:
		// Any key dismisses no multi-bean launchers modal
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
	case viewLaunchConfirm:
		a.launchConfirm, cmd = a.launchConfirm.Update(msg)
	case viewLaunchProgress:
		// Handle q/esc to cancel and stop all processes
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.String() {
			case "q", "esc":
				// Stop all processes
				if a.launchProgress.manager != nil {
					a.launchProgress.manager.Stop()
				}
				a.launchProgress.Cleanup()
				a.state = a.previousState
				return a, nil
			}
		}
		a.launchProgress, cmd = a.launchProgress.Update(msg)
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
	case viewNoLaunchersMulti:
		return a.noLaunchersMulti.ModalView(a.getBackgroundView(), a.width, a.height)
	case viewConfigureLaunchers:
		return a.configureLaunchers.ModalView(a.getBackgroundView(), a.width, a.height)
	case viewConfigWriteError:
		return a.configWriteError.ModalView(a.getBackgroundView(), a.width, a.height)
	case viewLaunchConfirm:
		return a.launchConfirm.View()
	case viewLaunchProgress:
		return a.launchProgress.View()
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
