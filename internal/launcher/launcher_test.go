package launcher

import (
	"errors"
	"os/exec"
	"strings"
	"testing"

	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/config"
)

func TestCreateExecCommand_EnvironmentVariables(t *testing.T) {
	tests := []struct {
		name        string
		execScript  string
		beansDir    string
		beanID      string
		beanTitle   string
		wantEnvVars map[string]string
	}{
		{
			name:       "single-line script has all env vars",
			execScript: "echo $BEANS_ROOT $BEANS_DIR $BEANS_ID $BEANS_TASK",
			beansDir:   "/project/.beans",
			beanID:     "beans-xyz",
			beanTitle:  "Test Task",
			wantEnvVars: map[string]string{
				"BEANS_ROOT": "/project",
				"BEANS_DIR":  "/project/.beans",
				"BEANS_ID":   "beans-xyz",
				"BEANS_TASK": "Test Task",
			},
		},
		{
			name:       "multi-line script has all env vars",
			execScript: "#!/bin/bash\necho $BEANS_ROOT $BEANS_DIR $BEANS_ID $BEANS_TASK",
			beansDir:   "/home/user/project/.beans",
			beanID:     "beans-abc",
			beanTitle:  "Another Task",
			wantEnvVars: map[string]string{
				"BEANS_ROOT": "/home/user/project",
				"BEANS_DIR":  "/home/user/project/.beans",
				"BEANS_ID":   "beans-abc",
				"BEANS_TASK": "Another Task",
			},
		},
		{
			name:       "nested beansDir path",
			execScript: "env",
			beansDir:   "/home/user/workspace/.beans",
			beanID:     "beans-123",
			beanTitle:  "Task Title",
			wantEnvVars: map[string]string{
				"BEANS_ROOT": "/home/user/workspace",
				"BEANS_DIR":  "/home/user/workspace/.beans",
				"BEANS_ID":   "beans-123",
				"BEANS_TASK": "Task Title",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, result, err := CreateExecCommand(tt.execScript, tt.beansDir, tt.beanID, tt.beanTitle)
			if err != nil {
				t.Fatalf("CreateExecCommand() error = %v", err)
			}
			defer result.Cleanup()

			if cmd == nil {
				t.Fatal("CreateExecCommand() returned nil cmd")
			}

			// Check that all expected env vars are set
			envMap := envSliceToMap(cmd.Env)
			for key, wantValue := range tt.wantEnvVars {
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

func TestCreateExecCommand_SingleLine(t *testing.T) {
	cmd, result, err := CreateExecCommand("echo hello", "/project/.beans", "beans-123", "Test")
	if err != nil {
		t.Fatalf("CreateExecCommand() error = %v", err)
	}
	defer result.Cleanup()

	if cmd == nil {
		t.Fatal("CreateExecCommand() returned nil cmd")
	}

	// Check it's using sh -c
	if cmd.Path == "" {
		t.Error("Command path is empty")
	}
	if !strings.Contains(cmd.Path, "sh") {
		t.Errorf("Expected sh command, got %s", cmd.Path)
	}

	// Check args
	if len(cmd.Args) < 3 {
		t.Fatalf("Expected at least 3 args [sh, -c, script], got %v", cmd.Args)
	}
	if cmd.Args[1] != "-c" {
		t.Errorf("Expected second arg to be -c, got %s", cmd.Args[1])
	}
	if cmd.Args[2] != "echo hello" {
		t.Errorf("Expected third arg to be 'echo hello', got %s", cmd.Args[2])
	}

	// Check working directory
	if cmd.Dir != "/project" {
		t.Errorf("Working directory = %v, want /project", cmd.Dir)
	}
}

func TestCreateExecCommand_MultiLine(t *testing.T) {
	script := "#!/bin/bash\necho hello\necho world"
	cmd, result, err := CreateExecCommand(script, "/project/.beans", "beans-456", "Multi Test")
	if err != nil {
		t.Fatalf("CreateExecCommand() error = %v", err)
	}
	defer result.Cleanup()

	if cmd == nil {
		t.Fatal("CreateExecCommand() returned nil cmd")
	}

	// Check stdin is set
	if cmd.Stdin == nil {
		t.Error("Stdin should be set for multi-line script")
	}

	// Check working directory
	if cmd.Dir != "/project" {
		t.Errorf("Working directory = %v, want /project", cmd.Dir)
	}
}

func TestCreateExecCommand_MultiLineNoShebang(t *testing.T) {
	script := "echo hello\necho world"
	_, result, err := CreateExecCommand(script, "/project/.beans", "beans-789", "No Shebang")
	if result != nil {
		defer result.Cleanup()
	}

	if err == nil {
		t.Error("Expected error for multi-line script without shebang")
	}
	if err != nil && !strings.Contains(err.Error(), "shebang") {
		t.Errorf("Expected shebang error, got: %v", err)
	}
}

func TestCreateExecCommand_InvalidShebang(t *testing.T) {
	script := "#!\necho hello"
	_, result, err := CreateExecCommand(script, "/project/.beans", "beans-999", "Invalid Shebang")
	if result != nil {
		defer result.Cleanup()
	}

	if err == nil {
		t.Error("Expected error for invalid shebang")
	}
	if err != nil && !strings.Contains(err.Error(), "shebang") {
		t.Errorf("Expected shebang error, got: %v", err)
	}
}

// envSliceToMap converts os.Environ() style slice to map
func envSliceToMap(envSlice []string) map[string]string {
	result := make(map[string]string)
	for _, entry := range envSlice {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}
	return result
}

// mockCmd wraps exec.Cmd for testing without executing
func mockCmd(name string, arg ...string) *exec.Cmd {
	return exec.Command(name, arg...)
}

func TestGetSummary_AllFields(t *testing.T) {
	// Create test beans
	beans := []*bean.Bean{
		{ID: "beans-1", Title: "Pending Task"},
		{ID: "beans-2", Title: "Running Task"},
		{ID: "beans-3", Title: "Success Task"},
		{ID: "beans-4", Title: "Failed Task"},
		{ID: "beans-5", Title: "Another Success"},
	}

	launcher := &config.Launcher{Name: "test"}
	manager := NewLaunchManager(launcher, beans)

	// Manually set states to test different scenarios
	manager.launches[0].Status = LaunchPending
	manager.launches[1].Status = LaunchRunning
	manager.launches[2].Status = LaunchSuccess
	manager.launches[3].Status = LaunchFailed
	manager.launches[3].Error = errors.New("test error")
	manager.launches[4].Status = LaunchSuccess

	summary := manager.GetSummary()

	// Verify Counts
	if summary.Counts.Total != 5 {
		t.Errorf("Counts.Total = %d, want 5", summary.Counts.Total)
	}
	if summary.Counts.Pending != 1 {
		t.Errorf("Counts.Pending = %d, want 1", summary.Counts.Pending)
	}
	if summary.Counts.Running != 1 {
		t.Errorf("Counts.Running = %d, want 1", summary.Counts.Running)
	}
	if summary.Counts.Success != 2 {
		t.Errorf("Counts.Success = %d, want 2", summary.Counts.Success)
	}
	if summary.Counts.Failed != 1 {
		t.Errorf("Counts.Failed = %d, want 1", summary.Counts.Failed)
	}

	// Verify Complete flag (false because pending and running exist)
	if summary.Complete {
		t.Error("Complete = true, want false (has pending/running)")
	}

	// Verify AllSuccessful flag (false because has failed/pending/running)
	if summary.AllSuccessful {
		t.Error("AllSuccessful = true, want false (has failures)")
	}

	// Verify FirstError
	if summary.FirstError == nil {
		t.Fatal("FirstError = nil, want failed launch")
	}
	if summary.FirstError.Bean.ID != "beans-4" {
		t.Errorf("FirstError.Bean.ID = %s, want beans-4", summary.FirstError.Bean.ID)
	}

	// Verify Launches is a deep copy (same length, different addresses)
	if len(summary.Launches) != 5 {
		t.Errorf("len(Launches) = %d, want 5", len(summary.Launches))
	}
	if &summary.Launches[0] == &manager.launches[0] {
		t.Error("Launches should be deep copy, got same pointer")
	}
}

func TestGetSummary_AllSuccessful(t *testing.T) {
	beans := []*bean.Bean{
		{ID: "beans-1", Title: "Task 1"},
		{ID: "beans-2", Title: "Task 2"},
	}

	launcher := &config.Launcher{Name: "test"}
	manager := NewLaunchManager(launcher, beans)

	// Set all to success
	manager.launches[0].Status = LaunchSuccess
	manager.launches[1].Status = LaunchSuccess

	summary := manager.GetSummary()

	// Verify Complete and AllSuccessful both true
	if !summary.Complete {
		t.Error("Complete = false, want true (all finished)")
	}
	if !summary.AllSuccessful {
		t.Error("AllSuccessful = false, want true (all succeeded)")
	}

	// Verify FirstError is nil
	if summary.FirstError != nil {
		t.Errorf("FirstError = %v, want nil", summary.FirstError)
	}

	// Verify counts
	if summary.Counts.Success != 2 {
		t.Errorf("Counts.Success = %d, want 2", summary.Counts.Success)
	}
	if summary.Counts.Failed != 0 {
		t.Errorf("Counts.Failed = %d, want 0", summary.Counts.Failed)
	}
}

func TestGetSummary_Empty(t *testing.T) {
	launcher := &config.Launcher{Name: "test"}
	manager := NewLaunchManager(launcher, []*bean.Bean{})

	summary := manager.GetSummary()

	// Empty should be complete and successful
	if !summary.Complete {
		t.Error("Complete = false, want true (empty is complete)")
	}
	if !summary.AllSuccessful {
		t.Error("AllSuccessful = false, want true (empty is successful)")
	}
	if summary.Counts.Total != 0 {
		t.Errorf("Counts.Total = %d, want 0", summary.Counts.Total)
	}
	if summary.FirstError != nil {
		t.Errorf("FirstError = %v, want nil", summary.FirstError)
	}
}
