package launcher

import (
	"bytes"
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/config"
)

// LaunchStatus represents the status of a bean launch
type LaunchStatus int

const (
	LaunchPending LaunchStatus = iota
	LaunchRunning
	LaunchSuccess
	LaunchFailed
)

func (s LaunchStatus) String() string {
	switch s {
	case LaunchPending:
		return "pending"
	case LaunchRunning:
		return "running"
	case LaunchSuccess:
		return "success"
	case LaunchFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// BeanLaunch represents the launch state of a single bean
type BeanLaunch struct {
	Bean     *bean.Bean
	Status   LaunchStatus
	Error    error
	Output   string // Captured stderr output
	cmd      *exec.Cmd
	result   *ExecutionResult
	started  time.Time
	finished time.Time
}

// Duration returns how long the launch took (or is taking)
func (bl *BeanLaunch) Duration() time.Duration {
	if bl.finished.IsZero() {
		if bl.started.IsZero() {
			return 0
		}
		return time.Since(bl.started)
	}
	return bl.finished.Sub(bl.started)
}

// LaunchCounts represents counts of launches by status
type LaunchCounts struct {
	Pending int
	Running int
	Success int
	Failed  int
	Total   int
}

// LaunchSummary provides a complete snapshot of launch manager state
type LaunchSummary struct {
	Launches      []*BeanLaunch // Deep copy of all launches
	Complete      bool          // True if all launches finished
	AllSuccessful bool          // True if all launches succeeded
	FirstError    *BeanLaunch   // First failed launch (nil if none)
	Counts        LaunchCounts  // Status counts
}

// LaunchManager manages parallel execution of launchers for multiple beans
type LaunchManager struct {
	launcher *config.Launcher
	launches []*BeanLaunch
	mu       sync.RWMutex
	stopOnce sync.Once
	stopped  bool
}

// NewLaunchManager creates a new launch manager
func NewLaunchManager(launcher *config.Launcher, beans []*bean.Bean) *LaunchManager {
	launches := make([]*BeanLaunch, len(beans))
	for i, b := range beans {
		launches[i] = &BeanLaunch{
			Bean:   b,
			Status: LaunchPending,
		}
	}

	return &LaunchManager{
		launcher: launcher,
		launches: launches,
	}
}

// Start begins execution of all launches in parallel
func (m *LaunchManager) Start(beansRoot string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.stopped {
		return fmt.Errorf("manager has been stopped")
	}

	for _, launch := range m.launches {
		if err := m.startLaunch(launch, beansRoot); err != nil {
			return err
		}
	}

	return nil
}

// startLaunch starts a single bean launch (must be called with lock held)
func (m *LaunchManager) startLaunch(launch *BeanLaunch, beansRoot string) error {
	// Create the command
	cmd, result, err := CreateExecCommand(m.launcher.Exec, beansRoot, launch.Bean.ID, launch.Bean.Title)
	if err != nil {
		launch.Status = LaunchFailed
		launch.Error = fmt.Errorf("failed to create command: %w", err)
		return launch.Error
	}

	launch.cmd = cmd
	launch.result = result

	// Capture stderr to provide better error messages
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	// Start the command in background
	if err := cmd.Start(); err != nil {
		result.Cleanup()
		launch.Status = LaunchFailed
		launch.Error = fmt.Errorf("failed to start command: %w", err)
		return launch.Error
	}

	launch.Status = LaunchRunning
	launch.started = time.Now()

	// Monitor completion in background
	go m.monitorLaunch(launch, &stderrBuf)

	return nil
}

// monitorLaunch monitors a single launch for completion
func (m *LaunchManager) monitorLaunch(launch *BeanLaunch, stderrBuf *bytes.Buffer) {
	err := launch.cmd.Wait()

	m.mu.Lock()
	defer m.mu.Unlock()

	launch.finished = time.Now()

	if err != nil {
		launch.Status = LaunchFailed
		// Include stderr output in error if available
		stderr := stderrBuf.String()
		if stderr != "" {
			launch.Output = stderr
			launch.Error = fmt.Errorf("%w\n\nOutput:\n%s", err, stderr)
		} else {
			launch.Error = err
		}
	} else {
		launch.Status = LaunchSuccess
	}

	// Cleanup temp files
	if launch.result != nil {
		launch.result.Cleanup()
	}
}

// Stop kills all running processes
func (m *LaunchManager) Stop() {
	m.stopOnce.Do(func() {
		m.mu.Lock()
		defer m.mu.Unlock()

		m.stopped = true

		for _, launch := range m.launches {
			if launch.Status == LaunchRunning && launch.cmd != nil && launch.cmd.Process != nil {
				// Kill the process
				_ = launch.cmd.Process.Kill()

				// Cleanup temp files
				if launch.result != nil {
					launch.result.Cleanup()
				}

				launch.Status = LaunchFailed
				launch.Error = fmt.Errorf("stopped by user")
				launch.finished = time.Now()
			}
		}
	})
}

// GetSummary returns a comprehensive snapshot of the launch manager state
func (m *LaunchManager) GetSummary() LaunchSummary {
	m.mu.RLock()
	defer m.mu.RUnlock()

	summary := LaunchSummary{
		Complete:      true,
		AllSuccessful: true,
		Counts: LaunchCounts{
			Total: len(m.launches),
		},
	}

	// Deep copy launches and compute all fields in single pass
	summary.Launches = make([]*BeanLaunch, len(m.launches))
	for i, launch := range m.launches {
		// Deep copy
		launchCopy := *launch
		summary.Launches[i] = &launchCopy

		// Update counts and flags based on status
		switch launch.Status {
		case LaunchPending:
			summary.Counts.Pending++
			summary.Complete = false
			summary.AllSuccessful = false
		case LaunchRunning:
			summary.Counts.Running++
			summary.Complete = false
			summary.AllSuccessful = false
		case LaunchSuccess:
			summary.Counts.Success++
		case LaunchFailed:
			summary.Counts.Failed++
			summary.AllSuccessful = false
			// Capture first error
			if summary.FirstError == nil && launch.Error != nil {
				summary.FirstError = &launchCopy
			}
		}
	}

	return summary
}
