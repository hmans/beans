package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hmans/beans/internal/config"
)

func TestResolveCommand(t *testing.T) {
	tests := []struct {
		name      string
		cmd       string
		beansRoot string
		want      string
	}{
		{
			name:      "absolute path unchanged",
			cmd:       "/usr/bin/foo",
			beansRoot: "/project",
			want:      "/usr/bin/foo",
		},
		{
			name:      "relative path resolved from beansRoot",
			cmd:       ".beans/scripts/tool.sh",
			beansRoot: "/project",
			want:      "/project/.beans/scripts/tool.sh",
		},
		{
			name:      "relative path with dot prefix",
			cmd:       "./scripts/tool.sh",
			beansRoot: "/project",
			want:      "/project/scripts/tool.sh", // filepath.Join cleans the path
		},
		{
			name:      "command name in PATH returns path",
			cmd:       "ls",
			beansRoot: "/project",
			want:      "/bin/ls", // This will vary by system but should be an absolute path
		},
		{
			name:      "command name not in PATH returns original",
			cmd:       "nonexistent-command-xyz",
			beansRoot: "/project",
			want:      "nonexistent-command-xyz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveCommand(tt.cmd, tt.beansRoot)

			// Special handling for PATH commands - we can't predict exact path
			if tt.cmd == "ls" {
				// Just verify it's an absolute path and ends with ls
				if !filepath.IsAbs(got) {
					t.Errorf("resolveCommand() for PATH command = %v, want absolute path", got)
				}
				if filepath.Base(got) != "ls" {
					t.Errorf("resolveCommand() for PATH command = %v, want path ending in 'ls'", got)
				}
			} else {
				if got != tt.want {
					t.Errorf("resolveCommand() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestIsCommandAvailable(t *testing.T) {
	// Create a temp directory for test files
	tmpDir := t.TempDir()

	// Create a test file (doesn't need to be executable since we use shell)
	testFilePath := filepath.Join(tmpDir, "test-script.sh")
	if err := os.WriteFile(testFilePath, []byte("#!/bin/sh\necho test"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		command string
		want    bool
	}{
		{
			name:    "file exists returns true",
			command: testFilePath,
			want:    true,
		},
		{
			name:    "nonexistent file returns false",
			command: filepath.Join(tmpDir, "nonexistent"),
			want:    false,
		},
		{
			name:    "PATH command found",
			command: "ls",
			want:    true,
		},
		{
			name:    "PATH command not found",
			command: "nonexistent-command-xyz-123",
			want:    false,
		},
		{
			name:    "command with args extracts main command",
			command: "ls -la /tmp",
			want:    true,
		},
		{
			name:    "relative path with args",
			command: testFilePath + " arg1 arg2",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isCommandAvailable(tt.command)
			if got != tt.want {
				t.Errorf("isCommandAvailable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDiscoverLaunchers(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()

	// Create an executable script
	executablePath := filepath.Join(tmpDir, "scripts", "my-tool.sh")
	if err := os.MkdirAll(filepath.Dir(executablePath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(executablePath, []byte("#!/bin/sh\necho test"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create a non-executable script (should be skipped)
	nonExecutablePath := filepath.Join(tmpDir, "scripts", "broken.sh")
	if err := os.WriteFile(nonExecutablePath, []byte("#!/bin/sh\necho test"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		cfg       *config.Config
		beansRoot string
		wantCount int
		wantNames []string
	}{
		{
			name: "empty config returns no launchers",
			cfg: &config.Config{
				Launchers: []config.Launcher{},
			},
			beansRoot: tmpDir,
			wantCount: 0,
			wantNames: []string{},
		},
		{
			name: "path command in PATH",
			cfg: &config.Config{
				Launchers: []config.Launcher{
					{Name: "ls-tool", Command: "ls", Description: "List files"},
				},
			},
			beansRoot: tmpDir,
			wantCount: 1,
			wantNames: []string{"ls-tool"},
		},
		{
			name: "executable relative path",
			cfg: &config.Config{
				Launchers: []config.Launcher{
					{Name: "my-tool", Command: "scripts/my-tool.sh", Description: "My tool"},
				},
			},
			beansRoot: tmpDir,
			wantCount: 1,
			wantNames: []string{"my-tool"},
		},
		{
			name: "non-executable relative path available (shell will execute)",
			cfg: &config.Config{
				Launchers: []config.Launcher{
					{Name: "broken", Command: "scripts/broken.sh", Description: "Broken"},
				},
			},
			beansRoot: tmpDir,
			wantCount: 1,
			wantNames: []string{"broken"},
		},
		{
			name: "nonexistent path skipped",
			cfg: &config.Config{
				Launchers: []config.Launcher{
					{Name: "missing", Command: "scripts/nonexistent.sh", Description: "Missing"},
				},
			},
			beansRoot: tmpDir,
			wantCount: 0,
			wantNames: []string{},
		},
		{
			name: "mix of available and unavailable",
			cfg: &config.Config{
				Launchers: []config.Launcher{
					{Name: "ls-tool", Command: "ls", Description: "List"},
					{Name: "my-tool", Command: "scripts/my-tool.sh", Description: "My tool"},
					{Name: "broken", Command: "scripts/broken.sh", Description: "Broken"},
					{Name: "missing", Command: "scripts/nonexistent.sh", Description: "Missing"},
				},
			},
			beansRoot: tmpDir,
			wantCount: 3,
			wantNames: []string{"ls-tool", "my-tool", "broken"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			launchers := discoverLaunchers(tt.cfg, tt.beansRoot)

			if len(launchers) != tt.wantCount {
				t.Errorf("discoverLaunchers() returned %d launchers, want %d", len(launchers), tt.wantCount)
			}

			// Check launcher names match
			gotNames := make([]string, len(launchers))
			for i, l := range launchers {
				gotNames[i] = l.name
			}

			if len(gotNames) != len(tt.wantNames) {
				t.Errorf("discoverLaunchers() names = %v, want %v", gotNames, tt.wantNames)
				return
			}

			for i, name := range tt.wantNames {
				if gotNames[i] != name {
					t.Errorf("discoverLaunchers() names[%d] = %v, want %v", i, gotNames[i], name)
				}
			}
		})
	}
}

func TestExtractMainCommand(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    string
	}{
		{
			name:    "simple command",
			command: "opencode",
			want:    "opencode",
		},
		{
			name:    "command with args",
			command: "opencode run \"Work on task $BEANS_ID\"",
			want:    "opencode",
		},
		{
			name:    "absolute path",
			command: "/usr/bin/tool --flag",
			want:    "/usr/bin/tool",
		},
		{
			name:    "relative path with args",
			command: ".beans/scripts/tool.sh arg1 arg2",
			want:    ".beans/scripts/tool.sh",
		},
		{
			name:    "command with quotes",
			command: "claude \"Work on task $BEANS_ID\"",
			want:    "claude",
		},
		{
			name:    "empty command",
			command: "",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractMainCommand(tt.command)
			if got != tt.want {
				t.Errorf("extractMainCommand() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasLaunchersConfigured(t *testing.T) {
	tests := []struct {
		name string
		cfg  *config.Config
		want bool
	}{
		{
			name: "no launchers configured",
			cfg: &config.Config{
				Launchers: []config.Launcher{},
			},
			want: false,
		},
		{
			name: "one launcher configured",
			cfg: &config.Config{
				Launchers: []config.Launcher{
					{Name: "test", Command: "test"},
				},
			},
			want: true,
		},
		{
			name: "multiple launchers configured",
			cfg: &config.Config{
				Launchers: []config.Launcher{
					{Name: "test1", Command: "test1"},
					{Name: "test2", Command: "test2"},
				},
			},
			want: true,
		},
		{
			name: "nil launchers slice",
			cfg: &config.Config{
				Launchers: nil,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasLaunchersConfigured(tt.cfg)
			if got != tt.want {
				t.Errorf("hasLaunchersConfigured() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAppendLaunchersToConfig(t *testing.T) {
	// Create temp directory with .beans.yml
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".beans.yml")

	// Write initial config
	initialConfig := `beans:
  path: .beans
  prefix: beans-
`
	if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
		t.Fatal(err)
	}

	// Launchers to append
	launchers := []config.Launcher{
		{Name: "opencode", Command: "opencode run \"Work on task $BEANS_ID\"", Description: "Open task in OpenCode"},
		{Name: "claude", Command: "claude \"Work on task $BEANS_ID\"", Description: "Open task in Claude Code"},
	}

	// Append launchers
	if err := appendLaunchersToConfig(tmpDir, launchers); err != nil {
		t.Fatalf("appendLaunchersToConfig() error = %v", err)
	}

	// Read result
	result, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}

	resultStr := string(result)

	// Verify launchers section exists
	if !strings.Contains(resultStr, "launchers:") {
		t.Error("Result does not contain launchers section")
	}

	// Verify first launcher
	if !strings.Contains(resultStr, "name: opencode") {
		t.Error("Result does not contain opencode launcher name")
	}
	if !strings.Contains(resultStr, "command: opencode run") {
		t.Error("Result does not contain opencode launcher command")
	}
	if !strings.Contains(resultStr, "description: \"Open task in OpenCode\"") {
		t.Error("Result does not contain opencode launcher description")
	}

	// Verify second launcher
	if !strings.Contains(resultStr, "name: claude") {
		t.Error("Result does not contain claude launcher name")
	}

	// Verify original content is preserved
	if !strings.Contains(resultStr, "beans:") {
		t.Error("Original config content was lost")
	}
	if !strings.Contains(resultStr, "path: .beans") {
		t.Error("Original beans path was lost")
	}
}
