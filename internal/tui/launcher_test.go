package tui

import (
	"os"
	"path/filepath"
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

	// Create an executable script
	executablePath := filepath.Join(tmpDir, "executable.sh")
	if err := os.WriteFile(executablePath, []byte("#!/bin/sh\necho test"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create a non-executable file
	nonExecutablePath := filepath.Join(tmpDir, "non-executable.sh")
	if err := os.WriteFile(nonExecutablePath, []byte("#!/bin/sh\necho test"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name         string
		resolvedPath string
		want         bool
	}{
		{
			name:         "executable file returns true",
			resolvedPath: executablePath,
			want:         true,
		},
		{
			name:         "non-executable file returns false",
			resolvedPath: nonExecutablePath,
			want:         false,
		},
		{
			name:         "nonexistent file returns false",
			resolvedPath: filepath.Join(tmpDir, "nonexistent"),
			want:         false,
		},
		{
			name:         "PATH command assumed available",
			resolvedPath: "ls",
			want:         true,
		},
		{
			name:         "command without path separator assumed available",
			resolvedPath: "my-command",
			want:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isCommandAvailable(tt.resolvedPath)
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
			name: "non-executable relative path skipped",
			cfg: &config.Config{
				Launchers: []config.Launcher{
					{Name: "broken", Command: "scripts/broken.sh", Description: "Broken"},
				},
			},
			beansRoot: tmpDir,
			wantCount: 0,
			wantNames: []string{},
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
			wantCount: 2,
			wantNames: []string{"ls-tool", "my-tool"},
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
