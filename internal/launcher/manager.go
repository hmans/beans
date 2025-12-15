package launcher

import (
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
	go m.monitorLaunch(launch)

	return nil
}

// monitorLaunch monitors a single launch for completion
func (m *LaunchManager) monitorLaunch(launch *BeanLaunch) {
	err := launch.cmd.Wait()

	m.mu.Lock()
	defer m.mu.Unlock()

	launch.finished = time.Now()

	if err != nil {
		launch.Status = LaunchFailed
		launch.Error = err
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

// GetStatus returns a snapshot of all launch statuses
func (m *LaunchManager) GetStatus() []*BeanLaunch {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Create a copy to avoid race conditions
	snapshot := make([]*BeanLaunch, len(m.launches))
	for i, launch := range m.launches {
		launchCopy := *launch
		snapshot[i] = &launchCopy
	}

	return snapshot
}

// IsComplete returns true if all launches have finished (success or failure)
func (m *LaunchManager) IsComplete() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, launch := range m.launches {
		if launch.Status == LaunchPending || launch.Status == LaunchRunning {
			return false
		}
	}

	return true
}

// AllSuccessful returns true if all launches completed successfully
func (m *LaunchManager) AllSuccessful() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, launch := range m.launches {
		if launch.Status != LaunchSuccess {
			return false
		}
	}

	return true
}

// GetFirstError returns the first error encountered, if any
func (m *LaunchManager) GetFirstError() (*BeanLaunch, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, launch := range m.launches {
		if launch.Status == LaunchFailed && launch.Error != nil {
			return launch, launch.Error
		}
	}

	return nil, nil
}

// GetCounts returns counts of launches by status
func (m *LaunchManager) GetCounts() (pending, running, success, failed int) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, launch := range m.launches {
		switch launch.Status {
		case LaunchPending:
			pending++
		case LaunchRunning:
			running++
		case LaunchSuccess:
			success++
		case LaunchFailed:
			failed++
		}
	}

	return
}
