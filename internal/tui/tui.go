package tui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hmans/beans/pkg/beancore"
	"github.com/hmans/beans/pkg/config"
	"github.com/hmans/beans/pkg/safepath"
	"github.com/hmans/beans/pkg/beangraph"
	"github.com/hmans/beans/pkg/beangraph/model"
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
	viewSettingsOverlay
)

// Pane geometry shared by the list and the preview.
const (
	paneBorder        = 1 // border thickness on each side of a pane
	paneBorders       = 2 * paneBorder
	footerHeight      = 1
	paneSeparator     = 1 // column between the two-column panes
	listBottomPadding = 1 // spare row below the single-column list
	previewPaddingX   = 1 // horizontal padding inside the preview border, per side
	// previewChromeX is the preview width not available to body text.
	previewChromeX = paneBorders + 2*previewPaddingX
)

// Layout selection: two columns, preview stacked below the list, or the list alone.
const (
	// PreviewMinTextWidth is the text width the preview aims for, following the
	// 80-column convention for text files.
	PreviewMinTextWidth = 80
	RightPaneMaxWidth   = PreviewMinTextWidth + previewChromeX
	LeftPaneMinWidth    = 40
	// TwoColumnFullWidth is the narrowest terminal whose two-column preview
	// keeps PreviewMinTextWidth; narrower, tall terminals stack instead.
	TwoColumnFullWidth = LeftPaneMinWidth + RightPaneMaxWidth
	// TwoColumnMinWidth is the narrowest terminal for two columns with a
	// shrunk preview, used when the terminal is too short to stack.
	TwoColumnMinWidth = 120

	// StackedBelowListMinHeight is the minimum number of rows below the list,
	// preview and footer together, for the stacked layout.
	StackedBelowListMinHeight = 20
	// StackedPreviewMaxTextWidth caps the stacked preview, since longer lines
	// are hard to read.
	StackedPreviewMaxTextWidth = 120
	StackedPreviewMaxWidth     = StackedPreviewMaxTextWidth + previewChromeX
)

// calculatePaneWidths returns (leftWidth, rightWidth) for two-column layout.
// Right pane is capped at RightPaneMaxWidth, left pane gets remaining space.
func calculatePaneWidths(totalWidth int) (int, int) {
	rightWidth := RightPaneMaxWidth
	if totalWidth-rightWidth < LeftPaneMinWidth {
		rightWidth = totalWidth - LeftPaneMinWidth
	}
	leftWidth := totalWidth - rightWidth - paneSeparator
	return leftWidth, rightWidth
}

// beansChangedMsg is sent when beans change on disk (via file watcher)
type beansChangedMsg struct{}

// cursorChangedMsg is sent when the list cursor moves to a different bean
type cursorChangedMsg struct {
	beanID  string
	blocked bool
}

// openTagPickerMsg requests opening the tag picker
type openTagPickerMsg struct{}

// tagSelectedMsg is sent when a tag is selected from the picker
type tagSelectedMsg struct {
	tag string
}

// clearFilterMsg is sent to clear any active filter
type clearFilterMsg struct{}

// copyBeanIDMsg requests copying bean ID(s) to the clipboard
type copyBeanIDMsg struct {
	ids []string
}

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
	state          viewState
	list           listModel
	detail         detailModel
	preview        previewModel
	tagPicker      tagPickerModel
	parentPicker   parentPickerModel
	statusPicker   statusPickerModel
	typePicker     typePickerModel
	blockingPicker blockingPickerModel
	priorityPicker priorityPickerModel
	createModal    createModalModel
	helpOverlay    helpOverlayModel
	settingsModal  settingsOverlayModel
	history        []detailModel // stack of previous detail views for back navigation
	core           *beancore.Core
	resolver       *beangraph.CoreResolver
	config         *config.Config
	width          int
	height         int
	// stackedListHeight is the configured list pane height in the stacked
	// layout, border included; see stackedListPaneHeight.
	stackedListHeight int
	settings          tuiSettings
	program           *tea.Program // reference to program for sending messages from watcher

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
	resolver := &beangraph.CoreResolver{Core: core}
	return &App{
		state:    viewList,
		core:     core,
		resolver: resolver,
		config:   cfg,
		list:     newListModel(resolver, cfg),
		preview:  newPreviewModel(nil, 0, 0),

		stackedListHeight: cfg.GetStackedListHeight(),
		settings: tuiSettings{
			previewPosition: cfg.GetPreviewPosition(),
			previewPercent:  DefaultPreviewPercent,
		},
	}
}

// stackedListPaneHeight returns the list pane height in the stacked layout,
// border included. Position below splits the height by previewPercent, keeping
// at least MinStackedListHeight rows for both the list and the preview.
func (a *App) stackedListPaneHeight() int {
	if a.settings.previewPosition != config.PreviewPositionBelow {
		return a.stackedListHeight
	}
	panes := a.height - footerHeight
	listHeight := (panes*(100-a.settings.previewPercent) + 50) / 100
	return max(config.MinStackedListHeight, min(listHeight, panes-config.MinStackedListHeight))
}

// stackedMinHeight returns the minimum terminal height for the preview below the list.
func (a *App) stackedMinHeight() int {
	return a.stackedListHeight + StackedBelowListMinHeight
}

// Init initializes the application
func (a *App) Init() tea.Cmd {
	return a.list.Init()
}

// isTwoColumnMode returns true if the list and preview are shown side by side.
// In auto position, a shrunk side preview is used only when the terminal is
// too short to stack. Position below never shows two columns.
func (a *App) isTwoColumnMode() bool {
	fitsTwoColumn := a.width >= TwoColumnMinWidth
	fitsStacked := a.height >= a.stackedMinHeight()
	switch a.settings.previewPosition {
	case config.PreviewPositionRight:
		return fitsTwoColumn
	case config.PreviewPositionBelow:
		return false
	}
	return a.width >= TwoColumnFullWidth || (fitsTwoColumn && !fitsStacked)
}

// togglePreviewPosition moves the preview from the side to below the list, or back.
func (a *App) togglePreviewPosition() {
	if a.isTwoColumnMode() {
		a.settings.previewPosition = config.PreviewPositionBelow
	} else {
		a.settings.previewPosition = config.PreviewPositionRight
	}
	a.applyLayout()
}

// applyLayout resizes the preview and the list to the current layout.
func (a *App) applyLayout() {
	a.preview.width, a.preview.height = a.previewSize()
	a.fitListToPane()
}

// isStackedMode returns true if the preview is shown below the list because
// two columns are not chosen and the terminal is tall enough for both. For
// position below, tall enough means room for both panes at minimum height.
func (a *App) isStackedMode() bool {
	if a.settings.previewPosition == config.PreviewPositionBelow {
		return a.height-footerHeight >= 2*config.MinStackedListHeight
	}
	return !a.isTwoColumnMode() && a.height >= a.stackedMinHeight()
}

// previewSize returns the size the preview pane is rendered at.
func (a *App) previewSize() (int, int) {
	if a.isStackedMode() {
		return min(a.width, StackedPreviewMaxWidth), a.height - a.stackedListPaneHeight() - footerHeight
	}
	_, rightWidth := calculatePaneWidths(a.width)
	return rightWidth, a.height - footerHeight
}

// listPaneSize returns the outer width and the inner height the list pane is
// rendered at.
func (a *App) listPaneSize() (int, int) {
	if a.isStackedMode() {
		return a.width, a.stackedListPaneHeight() - paneBorders
	}
	if a.isTwoColumnMode() {
		leftWidth, _ := calculatePaneWidths(a.width)
		return leftWidth, a.height - footerHeight - paneBorders
	}
	return a.width, a.height - footerHeight - paneBorders - listBottomPadding
}

// fitListToPane sizes the list model to its rendered pane, so that page
// navigation moves by the number of visible rows.
func (a *App) fitListToPane() {
	paneWidth, innerHeight := a.listPaneSize()
	a.list.list.SetSize(paneWidth-paneBorders, innerHeight)
}

// Update handles messages
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd, inactiveCmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height

		if a.isTwoColumnMode() || a.isStackedMode() {
			a.preview.width, a.preview.height = a.previewSize()
		}

		// Only the active view receives the message below, so resize the
		// inactive views too; a modal is drawn over them, and they render
		// again once it closes.
		if a.state != viewList {
			a.list, inactiveCmd = a.list.Update(msg)
			a.fitListToPane()
		}
		if a.state != viewDetail && a.detail.bean != nil {
			var detailCmd tea.Cmd
			a.detail, detailCmd = a.detail.Update(msg)
			inactiveCmd = tea.Batch(inactiveCmd, detailCmd)
		}

	case tea.KeyMsg:
		// Clear status messages on any keypress
		a.list.statusMessage = ""
		a.detail.statusMessage = ""

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
		case ",":
			typingFilter := a.list.list.FilterState() == 1
			if a.state == viewDetail {
				typingFilter = a.detail.linksActive && a.detail.linkList.FilterState() == 1
			}
			if (a.state == viewList || a.state == viewDetail) && !typingFilter {
				a.previousState = a.state
				a.settingsModal = newSettingsOverlayModel(a.settings, a.list.viewMode, a.width, a.height)
				a.state = viewSettingsOverlay
				return a, nil
			}
		case "pgup", "pgdown":
			// With a preview visible these scroll it; otherwise the list pages.
			if a.state == viewList && a.list.list.FilterState() != 1 && (a.isTwoColumnMode() || a.isStackedMode()) {
				// Clamp against the size the preview renders at.
				a.preview.width, a.preview.height = a.previewSize()
				if msg.String() == "pgup" {
					a.preview.scrollBy(-previewScrollStep)
				} else {
					a.preview.scrollBy(previewScrollStep)
				}
				return a, nil
			}
		case "v":
			if a.state == viewList && a.list.list.FilterState() != 1 {
				a.togglePreviewPosition()
				return a, nil
			}
		case "q":
			if a.state == viewDetail || a.state == viewTagPicker || a.state == viewParentPicker || a.state == viewStatusPicker || a.state == viewTypePicker || a.state == viewBlockingPicker || a.state == viewPriorityPicker || a.state == viewHelpOverlay || a.state == viewSettingsOverlay {
				return a, tea.Quit
			}
			// For list, only quit if not filtering
			if a.state == viewList && a.list.list.FilterState() != 1 {
				return a, tea.Quit
			}
		}

	case tea.MouseMsg:
		if a.state == viewList {
			if msg.Action != tea.MouseActionPress {
				return a, nil
			}
			paneWidth, innerHeight := a.listPaneSize()
			overPreview := false
			if a.isStackedMode() {
				overPreview = msg.Y >= a.stackedListPaneHeight()
			} else if a.isTwoColumnMode() {
				overPreview = msg.X >= paneWidth
			}
			if overPreview {
				// Clamp against the size the preview renders at.
				a.preview.width, a.preview.height = a.previewSize()
				switch msg.Button {
				case tea.MouseButtonWheelUp:
					a.preview.scrollBy(-previewScrollStep)
				case tea.MouseButtonWheelDown:
					a.preview.scrollBy(previewScrollStep)
				}
				return a, nil
			}
			if msg.Button != tea.MouseButtonLeft {
				return a, nil
			}
			a.list, cmd = a.list.selectAt(msg.X, msg.Y, paneWidth, innerHeight)
			return a, cmd
		}

	case cursorChangedMsg:
		// Update preview with the newly highlighted bean
		previewWidth, previewHeight := a.previewSize()
		if msg.beanID != "" {
			bean, err := a.resolver.Bean(context.Background(), msg.beanID)
			if err == nil && bean != nil {
				a.preview = newPreviewModel(bean, previewWidth, previewHeight)
				a.preview.blocked = msg.blocked
			}
		} else {
			a.preview = newPreviewModel(nil, previewWidth, previewHeight)
		}
		return a, nil

	case beansLoadedMsg:
		// Forward to list view
		a.list, cmd = a.list.Update(msg)
		// Update preview with current cursor position
		previewWidth, previewHeight := a.previewSize()
		if len(msg.items) == 0 {
			a.preview = newPreviewModel(nil, previewWidth, previewHeight)
		} else if item, ok := a.list.list.SelectedItem().(beanItem); ok {
			scroll := 0
			if a.preview.bean != nil && a.preview.bean.ID == item.bean.ID {
				scroll = a.preview.scroll
			}
			a.preview = newPreviewModel(item.bean, previewWidth, previewHeight)
			a.preview.blocked = item.showsBlocked()
			a.preview.scroll = scroll
		}
		return a, cmd

	case beansChangedMsg:
		// Beans changed on disk - refresh
		if a.state == viewDetail {
			// Try to reload the current bean via GraphQL
			updatedBean, err := a.resolver.Bean(context.Background(), a.detail.bean.ID)
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
			_, err := a.resolver.UpdateBean(context.Background(), beanID, model.UpdateBeanInput{
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
			updatedBean, _ := a.resolver.Bean(context.Background(), msg.beanIDs[0])
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
			_, err := a.resolver.UpdateBean(context.Background(), beanID, model.UpdateBeanInput{
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
			updatedBean, _ := a.resolver.Bean(context.Background(), msg.beanIDs[0])
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
			_, err := a.resolver.UpdateBean(context.Background(), beanID, model.UpdateBeanInput{
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
			updatedBean, _ := a.resolver.Bean(context.Background(), msg.beanIDs[0])
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

	case settingsChangedMsg:
		a.settings = msg.settings
		a.applyLayout()
		if msg.viewMode != a.list.viewMode {
			a.list.viewMode = msg.viewMode
			return a, a.list.loadBeans
		}
		return a, nil

	case closeSettingsMsg:
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
			_, err := a.resolver.AddBlocking(context.Background(), msg.beanID, targetID, nil)
			if err != nil {
				// Continue with other changes even if one fails
				continue
			}
		}
		for _, targetID := range msg.toRemove {
			_, err := a.resolver.RemoveBlocking(context.Background(), msg.beanID, targetID, nil)
			if err != nil {
				// Continue with other changes even if one fails
				continue
			}
		}
		// Return to previous view and refresh
		a.state = a.previousState
		if a.state == viewDetail {
			updatedBean, _ := a.resolver.Bean(context.Background(), msg.beanID)
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
		createdBean, err := a.resolver.CreateBean(context.Background(), model.CreateBeanInput{
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
		fullPath, err := safepath.SafeJoin(a.core.Root(), msg.beanPath)
		if err != nil {
			a.list.statusMessage = fmt.Sprintf("unsafe bean path: %v", err)
			return a, nil
		}

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
				fullPath, pathErr := safepath.SafeJoin(a.core.Root(), b.Path)
				if pathErr != nil {
					break
				}
				if info, err := os.Stat(fullPath); err == nil {
					if info.ModTime().After(a.editingBeanModTime) {
						// File was modified - reload from disk first to get user's changes,
						// then call Update to set updated_at
						_ = a.core.Load()
						if b, err = a.core.Get(a.editingBeanID); err == nil {
							_ = a.core.Update(b.Clone(), b.ETag())
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
			_, err := a.resolver.SetParent(context.Background(), beanID, parentID, nil)
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
			updatedBean, _ := a.resolver.Bean(context.Background(), msg.beanIDs[0])
			if updatedBean != nil {
				a.detail = newDetailModel(updatedBean, a.resolver, a.config, a.width, a.height)
			}
		}
		return a, a.list.loadBeans

	case clearFilterMsg:
		a.list.clearFilter()
		return a, a.list.loadBeans

	case copyBeanIDMsg:
		var statusMsg string
		text := strings.Join(msg.ids, ", ")
		if err := clipboard.WriteAll(text); err != nil {
			statusMsg = fmt.Sprintf("Failed to copy: %v", err)
		} else if len(msg.ids) == 1 {
			statusMsg = fmt.Sprintf("Copied %s to clipboard", msg.ids[0])
		} else {
			statusMsg = fmt.Sprintf("Copied %d bean IDs to clipboard", len(msg.ids))
		}

		// Set status on current view
		if a.state == viewList {
			a.list.statusMessage = statusMsg
		} else if a.state == viewDetail {
			a.detail.statusMessage = statusMsg
		}

		return a, nil

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
		// Fit first: the terminal may have been resized while another view was active.
		a.fitListToPane()
		a.list, cmd = a.list.Update(msg)
		if _, ok := msg.(tea.WindowSizeMsg); ok {
			a.fitListToPane()
		}
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
	case viewSettingsOverlay:
		a.settingsModal, cmd = a.settingsModal.Update(msg)
	}

	return a, tea.Batch(inactiveCmd, cmd)
}

// collectTagsWithCounts returns all tags with their usage counts
func (a *App) collectTagsWithCounts() []tagWithCount {
	beans, _ := a.resolver.Beans(context.Background(), nil)
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

// renderTwoColumnView renders the list and preview side by side with app-global footer
func (a *App) renderTwoColumnView() string {
	leftWidth, rightWidth := calculatePaneWidths(a.width)
	contentHeight := a.height - footerHeight

	// Render left pane (list) with constrained width, no footer
	leftPane := a.list.ViewConstrained(leftWidth, contentHeight)

	// Render right pane (preview) with same height
	a.preview.width = rightWidth
	a.preview.height = contentHeight
	rightPane := a.preview.View()

	// Compose columns
	columns := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)

	// App-global footer spans full width
	footer := a.list.Footer()

	return columns + "\n" + footer
}

// renderStackedView renders the preview below a fixed-height list.
func (a *App) renderStackedView() string {
	listPane := a.list.ViewConstrained(a.width, a.stackedListPaneHeight())
	a.preview.width, a.preview.height = a.previewSize()
	return listPane + "\n" + a.preview.View() + "\n" + a.list.Footer()
}

// View renders the current view
func (a *App) View() string {
	switch a.state {
	case viewList:
		return a.renderListLayout()
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
	case viewSettingsOverlay:
		return a.settingsModal.ModalView(a.getBackgroundView(), a.width, a.height)
	}
	return ""
}

// renderListLayout renders the list view in the layout that fits the terminal.
func (a *App) renderListLayout() string {
	if a.isTwoColumnMode() {
		return a.renderTwoColumnView()
	}
	if a.isStackedMode() {
		return a.renderStackedView()
	}
	return a.list.View()
}

// getBackgroundView returns the view to show behind modal pickers
func (a *App) getBackgroundView() string {
	if a.previousState == viewDetail {
		return a.detail.View()
	}
	return a.renderListLayout()
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
	p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())

	// Store reference to program for sending messages from watcher
	app.program = p

	// Start file watching
	if err := core.StartWatching(); err != nil {
		return err
	}
	defer core.Unwatch()

	// Subscribe to bean events
	eventCh, unsubscribe := core.Subscribe()
	defer unsubscribe()

	// Forward events to TUI in a goroutine
	go func() {
		for range eventCh {
			// Send message to TUI when beans change
			if app.program != nil {
				app.program.Send(beansChangedMsg{})
			}
		}
	}()

	_, err := p.Run()
	return err
}
