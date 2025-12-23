package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hmans/beans/internal/config"
	launcherexec "github.com/hmans/beans/internal/launcher"
)

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
			name: "single-line exec",
			cfg: &config.Config{
				Launchers: []config.Launcher{
					{Name: "ls-tool", Exec: "ls", Description: "List files"},
				},
			},
			beansRoot: tmpDir,
			wantCount: 1,
			wantNames: []string{"ls-tool"},
		},
		{
			name: "single-line exec with args",
			cfg: &config.Config{
				Launchers: []config.Launcher{
					{Name: "my-tool", Exec: "echo hello", Description: "My tool"},
				},
			},
			beansRoot: tmpDir,
			wantCount: 1,
			wantNames: []string{"my-tool"},
		},
		{
			name: "multi-line exec with shebang",
			cfg: &config.Config{
				Launchers: []config.Launcher{
					{Name: "script", Exec: "#!/bin/bash\necho hello", Description: "Script"},
				},
			},
			beansRoot: tmpDir,
			wantCount: 1,
			wantNames: []string{"script"},
		},
		{
			name: "empty exec skipped",
			cfg: &config.Config{
				Launchers: []config.Launcher{
					{Name: "missing", Exec: "", Description: "Missing"},
				},
			},
			beansRoot: tmpDir,
			wantCount: 0,
			wantNames: []string{},
		},
		{
			name: "mix of single and multi-line exec",
			cfg: &config.Config{
				Launchers: []config.Launcher{
					{Name: "ls-tool", Exec: "ls", Description: "List"},
					{Name: "my-tool", Exec: "echo hello", Description: "My tool"},
					{Name: "script", Exec: "#!/bin/bash\necho world", Description: "Script"},
				},
			},
			beansRoot: tmpDir,
			wantCount: 3,
			wantNames: []string{"ls-tool", "my-tool", "script"},
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
					{Name: "test", Exec: "test"},
				},
			},
			want: true,
		},
		{
			name: "multiple launchers configured",
			cfg: &config.Config{
				Launchers: []config.Launcher{
					{Name: "test1", Exec: "test1"},
					{Name: "test2", Exec: "test2"},
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

func TestLauncherIntegration_EnvironmentVariables(t *testing.T) {
	// This test verifies that the TUI properly uses the launcher package's CreateExecCommand
	// which includes all required environment variables including BEANS_DIR
	tests := []struct {
		name      string
		script    string
		beansDir  string
		beanID    string
		beanTitle string
		wantEnv   map[string]string
	}{
		{
			name:      "single-line script sets all env vars including BEANS_DIR",
			script:    "echo test",
			beansDir:  "/project/.beans",
			beanID:    "xyz",
			beanTitle: "Test Task",
			wantEnv: map[string]string{
				"BEANS_ROOT": "/project",
				"BEANS_DIR":  "/project/.beans",
				"BEANS_ID":   "xyz",
				"BEANS_TASK": "Test Task",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use the launcher package directly (this is what TUI now uses)
			cmd, result, err := launcherexec.CreateExecCommand(tt.script, tt.beansDir, tt.beanID, tt.beanTitle)
			if err != nil {
				t.Fatalf("CreateExecCommand() error = %v", err)
			}
			defer result.Cleanup()

			if cmd == nil {
				t.Fatal("CreateExecCommand() returned nil cmd")
			}

			// Convert env slice to map for easier checking
			envMap := make(map[string]string)
			for _, env := range cmd.Env {
				parts := strings.SplitN(env, "=", 2)
				if len(parts) == 2 {
					envMap[parts[0]] = parts[1]
				}
			}

			// Check each expected env var
			for key, wantValue := range tt.wantEnv {
				gotValue, exists := envMap[key]
				if !exists {
					t.Errorf("Environment variable %s not set", key)
					continue
				}
				if gotValue != wantValue {
					t.Errorf("Environment variable %s = %v, want %v", key, gotValue, wantValue)
				}
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
		{Name: "opencode", Exec: "opencode -p \"Work on task $BEANS_ID\"", Description: "Open task in OpenCode"},
		{Name: "claude", Exec: "claude \"Work on task $BEANS_ID\"", Description: "Open task in Claude Code"},
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
	if !strings.Contains(resultStr, "exec: opencode -p") {
		t.Error("Result does not contain opencode launcher exec")
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
