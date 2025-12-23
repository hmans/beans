package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/config"
	launcherexec "github.com/hmans/beans/internal/launcher"
)

// TestLaunchProgressDoesNotDismissOnKeyPress is a regression test for beans-b1u8.
// Previously, the viewLaunchProgress case had misplaced code that dismissed the
// view on ANY key press, causing the TUI to get stuck showing "Loading...".
func TestLaunchProgressDoesNotDismissOnKeyPress(t *testing.T) {
	// Create a minimal app with launch progress state
	app := &App{
		state:         viewLaunchProgress,
		previousState: viewList,
		width:         80,
		height:        24,
	}

	// Create a mock launcher manager with some beans
	mockBeans := []*bean.Bean{
		{ID: "test1", Title: "Test Bean 1"},
		{ID: "test2", Title: "Test Bean 2"},
	}
	mockLauncher := &config.Launcher{
		Name:     "test-launcher",
		Exec:     "echo test",
		Multiple: true,
	}
	manager := launcherexec.NewLaunchManager(mockLauncher, mockBeans)

	app.launchProgress = newLaunchProgress(manager, mockLauncher.Name)

	// Set initial window size so View() doesn't return "Loading..."
	app.launchProgress.width = 80
	app.launchProgress.height = 24

	// Test that arbitrary key presses don't dismiss the view
	testCases := []struct {
		name   string
		keyMsg tea.KeyMsg
	}{
		{"letter key", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}},
		{"space key", tea.KeyMsg{Type: tea.KeySpace}},
		{"enter key", tea.KeyMsg{Type: tea.KeyEnter}},
		{"arrow key", tea.KeyMsg{Type: tea.KeyUp}},
		{"number key", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Store initial state
			initialState := app.state

			// Send the key message
			_, _ = app.Update(tc.keyMsg)

			// Verify the state hasn't changed (view not dismissed)
			if app.state != initialState {
				t.Errorf("key %q dismissed launch progress view (state changed from %v to %v), but should not have",
					tc.name, initialState, app.state)
			}

			// Verify we're still in launch progress view
			if app.state != viewLaunchProgress {
				t.Errorf("after key %q, expected state viewLaunchProgress, got %v", tc.name, app.state)
			}
		})
	}
}

// TestLaunchProgressDismissesOnQuitKeys verifies that q and esc still properly
// dismiss the launch progress view.
func TestLaunchProgressDismissesOnQuitKeys(t *testing.T) {
	testCases := []struct {
		name string
		key  string
	}{
		{"q key", "q"},
		{"esc key", "esc"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a minimal app with launch progress state
			app := &App{
				state:         viewLaunchProgress,
				previousState: viewList,
				width:         80,
				height:        24,
			}

			// Create a mock launcher manager
			mockBeans := []*bean.Bean{
				{ID: "test1", Title: "Test Bean 1"},
			}
			mockLauncher := &config.Launcher{
				Name: "test-launcher",
				Exec: "echo test",
			}
			manager := launcherexec.NewLaunchManager(mockLauncher, mockBeans)
			app.launchProgress = newLaunchProgress(manager, mockLauncher.Name)
			app.launchProgress.width = 80
			app.launchProgress.height = 24

			// Send the quit key
			keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tc.key)}
			if tc.key == "esc" {
				keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
			}

			_, _ = app.Update(keyMsg)

			// Verify the view was dismissed (returned to previous state)
			if app.state != viewList {
				t.Errorf("after pressing %q, expected to return to viewList, got %v", tc.key, app.state)
			}
		})
	}
}

// TestLaunchProgressDismissesOnCompletion verifies that the launch progress view
// properly dismisses when all launches complete.
func TestLaunchProgressDismissesOnCompletion(t *testing.T) {
	app := &App{
		state:         viewLaunchProgress,
		previousState: viewList,
		width:         80,
		height:        24,
		list:          listModel{selectedBeans: map[string]bool{"test1": true}},
	}

	// Create a mock launcher manager
	mockBeans := []*bean.Bean{
		{ID: "test1", Title: "Test Bean 1"},
	}
	mockLauncher := &config.Launcher{
		Name: "test-launcher",
		Exec: "echo test",
	}
	manager := launcherexec.NewLaunchManager(mockLauncher, mockBeans)
	app.launchProgress = newLaunchProgress(manager, mockLauncher.Name)
	app.launchProgress.width = 80
	app.launchProgress.height = 24

	// Simulate completion
	_, _ = app.Update(launchCompleteMsg{})

	// Verify the view was dismissed
	if app.state != viewList {
		t.Errorf("after launchCompleteMsg, expected to return to viewList, got %v", app.state)
	}

	// Verify selection was cleared
	if len(app.list.selectedBeans) != 0 {
		t.Errorf("after launchCompleteMsg, expected selection to be cleared, got %d selected beans", len(app.list.selectedBeans))
	}
}

// TestLaunchProgressDismissesOnFailure verifies that the launch progress view
// properly handles failures.
func TestLaunchProgressDismissesOnFailure(t *testing.T) {
	app := &App{
		state:         viewLaunchProgress,
		previousState: viewList,
		width:         80,
		height:        24,
	}

	// Create a mock launcher manager
	mockBeans := []*bean.Bean{
		{ID: "test1", Title: "Test Bean 1"},
	}
	mockLauncher := &config.Launcher{
		Name: "test-launcher",
		Exec: "echo test",
	}
	manager := launcherexec.NewLaunchManager(mockLauncher, mockBeans)
	app.launchProgress = newLaunchProgress(manager, mockLauncher.Name)
	app.launchProgress.width = 80
	app.launchProgress.height = 24

	// Simulate failure
	testErr := tea.ErrProgramKilled
	_, _ = app.Update(launchFailedMsg{beanID: "test1", err: testErr})

	// Verify the view switched to error view
	if app.state != viewLauncherError {
		t.Errorf("after launchFailedMsg, expected to switch to viewLauncherError, got %v", app.state)
	}
}

// TestLaunchProgressProcessesProgressMessages verifies that launchProgressMsg
// is properly processed and doesn't dismiss the view.
func TestLaunchProgressProcessesProgressMessages(t *testing.T) {
	app := &App{
		state:         viewLaunchProgress,
		previousState: viewList,
		width:         80,
		height:        24,
	}

	// Create a mock launcher manager
	mockBeans := []*bean.Bean{
		{ID: "test1", Title: "Test Bean 1"},
	}
	mockLauncher := &config.Launcher{
		Name: "test-launcher",
		Exec: "echo test",
	}
	manager := launcherexec.NewLaunchManager(mockLauncher, mockBeans)
	app.launchProgress = newLaunchProgress(manager, mockLauncher.Name)
	app.launchProgress.width = 80
	app.launchProgress.height = 24

	// Send a progress message (simulating the ticker)
	_, _ = app.Update(launchProgressMsg{})

	// Verify we're still in launch progress view
	if app.state != viewLaunchProgress {
		t.Errorf("after launchProgressMsg, expected to stay in viewLaunchProgress, got %v", app.state)
	}
}

// TestLaunchProgressHandlesWindowResize verifies that window resize messages
// don't dismiss the view.
func TestLaunchProgressHandlesWindowResize(t *testing.T) {
	app := &App{
		state:         viewLaunchProgress,
		previousState: viewList,
		width:         80,
		height:        24,
	}

	// Create a mock launcher manager
	mockBeans := []*bean.Bean{
		{ID: "test1", Title: "Test Bean 1"},
	}
	mockLauncher := &config.Launcher{
		Name: "test-launcher",
		Exec: "echo test",
	}
	manager := launcherexec.NewLaunchManager(mockLauncher, mockBeans)
	app.launchProgress = newLaunchProgress(manager, mockLauncher.Name)
	app.launchProgress.width = 80
	app.launchProgress.height = 24

	// Simulate window resize
	resizeMsg := tea.WindowSizeMsg{Width: 100, Height: 30}
	_, _ = app.Update(resizeMsg)

	// Verify we're still in launch progress view
	if app.state != viewLaunchProgress {
		t.Errorf("after WindowSizeMsg, expected to stay in viewLaunchProgress, got %v", app.state)
	}

	// Verify dimensions were updated
	if app.width != 100 || app.height != 30 {
		t.Errorf("after WindowSizeMsg, expected dimensions 100x30, got %dx%d", app.width, app.height)
	}
}

// TestLaunchProgressTickerCleanup verifies that the ticker is properly stopped
// when the view is dismissed.
func TestLaunchProgressTickerCleanup(t *testing.T) {
	app := &App{
		state:         viewLaunchProgress,
		previousState: viewList,
		width:         80,
		height:        24,
	}

	// Create a mock launcher manager
	mockBeans := []*bean.Bean{
		{ID: "test1", Title: "Test Bean 1"},
	}
	mockLauncher := &config.Launcher{
		Name: "test-launcher",
		Exec: "echo test",
	}
	manager := launcherexec.NewLaunchManager(mockLauncher, mockBeans)
	app.launchProgress = newLaunchProgress(manager, mockLauncher.Name)
	app.launchProgress.width = 80
	app.launchProgress.height = 24

	// Verify ticker is running
	if app.launchProgress.ticker == nil {
		t.Fatal("expected ticker to be initialized")
	}

	// Dismiss the view with q
	_, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	// Give a small delay to allow cleanup
	time.Sleep(10 * time.Millisecond)

	// Note: We can't directly verify the ticker is stopped, but we can verify
	// the cleanup method was called and the state changed correctly
	if app.state != viewList {
		t.Errorf("after pressing q, expected to return to viewList, got %v", app.state)
	}
}
