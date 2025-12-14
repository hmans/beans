package tui

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hmans/beans/internal/config"
)

// resolveCommand resolves a command string to an absolute path.
// - Absolute paths are returned as-is
// - Relative paths (containing "/" or starting with ".") are resolved from beansRoot
// - Command names are looked up in PATH
func resolveCommand(cmd string, beansRoot string) string {
	// Absolute path - return as-is
	if filepath.IsAbs(cmd) {
		return cmd
	}

	// Relative path - resolve from beansRoot
	if strings.Contains(cmd, string(filepath.Separator)) || strings.HasPrefix(cmd, ".") {
		return filepath.Join(beansRoot, cmd)
	}

	// Command name - check in PATH
	if path, err := exec.LookPath(cmd); err == nil {
		return path
	}

	// Not found in PATH, return as-is (will fail at execution with good error)
	return cmd
}

// isCommandAvailable checks if a resolved command path is available.
// For local file paths, checks if file exists and is executable.
// For PATH commands (no path separator), assumes available.
func isCommandAvailable(resolvedPath string) bool {
	// If it looks like a local file path, check if it exists and is executable
	if filepath.IsAbs(resolvedPath) || strings.Contains(resolvedPath, string(filepath.Separator)) {
		info, err := os.Stat(resolvedPath)
		if err != nil {
			return false
		}
		// Check if executable (any execute bit set)
		mode := info.Mode()
		return mode&0111 != 0
	}

	// For PATH commands, assume available (let exec fail with proper error)
	return true
}

// launcher represents a discovered and available launcher
type launcher struct {
	name        string
	command     string // resolved command path
	description string
}

// discoverLaunchers discovers all available launchers from config.
// It resolves command paths and filters out unavailable launchers.
func discoverLaunchers(cfg *config.Config, beansRoot string) []launcher {
	var launchers []launcher

	for _, lc := range cfg.Launchers {
		// Skip launchers with missing required fields
		if lc.Name == "" || lc.Command == "" {
			continue
		}

		// Resolve command path
		cmdPath := resolveCommand(lc.Command, beansRoot)

		// Check if available
		if !isCommandAvailable(cmdPath) {
			continue
		}

		launchers = append(launchers, launcher{
			name:        lc.Name,
			command:     cmdPath,
			description: lc.Description,
		})
	}

	return launchers
}
