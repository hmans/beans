package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	launcherexec "github.com/hmans/beans/internal/launcher"
	"github.com/hmans/beans/internal/ui"
)

// launchProgressMsg is sent periodically to update progress display
type launchProgressMsg struct{}

// launchCompleteMsg is sent when all launches complete successfully
type launchCompleteMsg struct{}

// launchFailedMsg is sent when a launch fails
type launchFailedMsg struct {
	beanID string
	err    error
}

// launchProgress displays real-time progress of parallel launcher execution
type launchProgress struct {
	manager      *launcherexec.LaunchManager
	launcherName string
	width        int
	height       int
	ticker       *time.Ticker
}

func newLaunchProgress(manager *launcherexec.LaunchManager, launcherName string) launchProgress {
	return launchProgress{
		manager:      manager,
		launcherName: launcherName,
		ticker:       time.NewTicker(100 * time.Millisecond),
	}
}

func (m launchProgress) Init() tea.Cmd {
	return m.tick()
}

func (m launchProgress) Update(msg tea.Msg) (launchProgress, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case launchProgressMsg:
		// Check if complete
		if m.manager.IsComplete() {
			m.ticker.Stop()

			// Check for failures
			if failedLaunch, err := m.manager.GetFirstError(); err != nil {
				// Stop all other processes
				m.manager.Stop()

				return m, func() tea.Msg {
					return launchFailedMsg{
						beanID: failedLaunch.Bean.ID,
						err:    err,
					}
				}
			}

			// All successful
			return m, func() tea.Msg {
				return launchCompleteMsg{}
			}
		}

		// Continue ticking
		return m, m.tick()
	}

	return m, nil
}

func (m launchProgress) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var b strings.Builder

	// Title
	title := fmt.Sprintf("Launching %s", m.launcherName)
	b.WriteString(launchTitleStyle.Render(title))
	b.WriteString("\n\n")

	// Get status counts
	pending, running, success, failed := m.manager.GetCounts()
	total := pending + running + success + failed

	// Progress bar (simple text-based)
	percent := 0.0
	if total > 0 {
		percent = float64(success+failed) / float64(total)
	}
	barWidth := min(m.width-10, 60)
	bar := renderProgressBar(percent, barWidth)
	b.WriteString(bar)
	b.WriteString("\n\n")

	// Status summary
	summary := fmt.Sprintf("Total: %d | Running: %d | Success: %d | Failed: %d",
		total, running, success, failed)
	b.WriteString(summaryStyle.Render(summary))
	b.WriteString("\n\n")

	// List all beans with status indicators
	launches := m.manager.GetStatus()
	maxLines := m.height - 12 // Reserve space for header, progress, summary, help

	for i, launch := range launches {
		if i >= maxLines {
			remaining := len(launches) - i
			b.WriteString(dimStyle.Render(fmt.Sprintf("  ... and %d more", remaining)))
			break
		}

		// Status indicator
		var indicator string
		var style lipgloss.Style

		switch launch.Status {
		case launcherexec.LaunchPending:
			indicator = "⋯"
			style = dimStyle
		case launcherexec.LaunchRunning:
			indicator = "◐"
			style = runningStyle
		case launcherexec.LaunchSuccess:
			indicator = "✓"
			style = successStyle
		case launcherexec.LaunchFailed:
			indicator = "✗"
			style = errorStyle
		}

		// Bean info
		line := fmt.Sprintf("  %s %s", indicator, launch.Bean.ID)

		// Show duration for running/complete
		duration := launch.Duration()
		if duration > 0 {
			line += fmt.Sprintf(" (%s)", duration.Round(time.Millisecond))
		}

		// Show error if failed
		if launch.Status == launcherexec.LaunchFailed && launch.Error != nil {
			line += fmt.Sprintf(": %s", launch.Error)
		}

		b.WriteString(style.Render(line))
		b.WriteString("\n")
	}

	// Help
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("Press q or esc to cancel and stop all processes"))

	return b.String()
}

// tick returns a command that waits for the next tick
func (m launchProgress) tick() tea.Cmd {
	return func() tea.Msg {
		<-m.ticker.C
		return launchProgressMsg{}
	}
}

// Cleanup stops the ticker
func (m *launchProgress) Cleanup() {
	if m.ticker != nil {
		m.ticker.Stop()
	}
}

// renderProgressBar creates a simple text-based progress bar
func renderProgressBar(percent float64, width int) string {
	filled := int(percent * float64(width))
	empty := width - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	label := fmt.Sprintf(" %.0f%%", percent*100)

	return progressBarStyle.Render(bar) + label
}

// Styles for launch progress view
var (
	launchTitleStyle = lipgloss.NewStyle().
				Foreground(ui.ColorPrimary).
				Bold(true)

	progressBarStyle = lipgloss.NewStyle().
				Foreground(ui.ColorPrimary)

	summaryStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	runningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")) // Orange

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")) // Green

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")) // Gray

	errorStyle = lipgloss.NewStyle().
			Foreground(ui.ColorDanger)
)
