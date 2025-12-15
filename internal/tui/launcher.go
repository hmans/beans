package tui

import (
	"fmt"
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

// isCommandAvailable checks if a launcher command is available.
// Since we execute via shell, we extract the main command and check if it exists.
// For file paths, checks if file exists. For command names, checks if in PATH.
func isCommandAvailable(command string) bool {
	// Extract the main executable from the command string
	mainCmd := extractMainCommand(command)

	if mainCmd == "" {
		return false
	}

	// If it looks like a local file path, check if it exists
	if filepath.IsAbs(mainCmd) || strings.Contains(mainCmd, string(filepath.Separator)) {
		_, err := os.Stat(mainCmd)
		return err == nil
	}

	// For command names, check if in PATH
	_, err := exec.LookPath(mainCmd)
	return err == nil
}

// launcher represents a discovered and available launcher
type launcher struct {
	name        string
	exec        string // exec script (single-line or multi-line)
	description string
}

// discoverLaunchers discovers all available launchers from config.
// All configured launchers are considered available (validation happens at execution time).
func discoverLaunchers(cfg *config.Config, beansRoot string) []launcher {
	var launchers []launcher

	for _, lc := range cfg.Launchers {
		// Skip launchers with missing required fields
		if lc.Name == "" || lc.Exec == "" {
			continue
		}

		// All exec launchers are shown - single-line and multi-line
		// Availability is checked at execution time
		launchers = append(launchers, launcher{
			name:        lc.Name,
			exec:        lc.Exec,
			description: lc.Description,
		})
	}

	return launchers
}

// discoverLaunchersForMultiple discovers launchers that support multiple beans in parallel.
// Returns only launchers configured with multiple: true.
func discoverLaunchersForMultiple(cfg *config.Config, beansRoot string) []launcher {
	allLaunchers := discoverLaunchers(cfg, beansRoot)

	var multiLaunchers []launcher
	for _, l := range allLaunchers {
		// Find the config launcher to check Multiple flag
		for _, cfgLauncher := range cfg.Launchers {
			if cfgLauncher.Name == l.name && cfgLauncher.Multiple {
				multiLaunchers = append(multiLaunchers, l)
				break
			}
		}
	}

	return multiLaunchers
}

// extractMainCommand extracts the primary executable from a launcher command string.
// For shell commands, this returns the first space-separated token.
// Examples:
//
//	"opencode run ..." -> "opencode"
//	"/usr/bin/tool --flag" -> "/usr/bin/tool"
//	".beans/scripts/tool.sh arg" -> ".beans/scripts/tool.sh"
func extractMainCommand(command string) string {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return command
	}
	return parts[0]
}

// hasLaunchersConfigured returns true if the config has any launchers defined.
func hasLaunchersConfigured(cfg *config.Config) bool {
	return len(cfg.Launchers) > 0
}

// appendLaunchersToConfig appends launcher configurations to .beans.yml.
// This is a simple append operation that doesn't preserve comments or formatting.
// projectRoot should be the directory containing .beans.yml (not the .beans directory).
func appendLaunchersToConfig(projectRoot string, launchers []config.Launcher) error {
	configPath := filepath.Join(projectRoot, config.ConfigFileName)

	// Read current file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	// Build launchers YAML section
	var launchersYAML strings.Builder
	launchersYAML.WriteString("\nlaunchers:\n")
	for _, l := range launchers {
		launchersYAML.WriteString(fmt.Sprintf("  - name: %s\n", l.Name))
		launchersYAML.WriteString(fmt.Sprintf("    exec: %s\n", l.Exec))
		if l.Description != "" {
			launchersYAML.WriteString(fmt.Sprintf("    description: \"%s\"\n", l.Description))
		}
	}

	// Append to file
	newData := append(data, []byte(launchersYAML.String())...)
	if err := os.WriteFile(configPath, newData, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}
