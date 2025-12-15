package launcher

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ExecutionResult holds a command and its cleanup function
type ExecutionResult struct {
	Cmd     *exec.Cmd
	cleanup func()
}

// Cleanup removes temporary files created for the execution
func (r *ExecutionResult) Cleanup() {
	if r.cleanup != nil {
		r.cleanup()
	}
}

// CreateExecCommand creates a command for executing an exec script.
// For multi-line scripts with shebang, passes script via stdin to the interpreter.
// For single-line scripts, it executes via sh -c.
//
// Parameters:
//   - execScript: The script to execute (single-line or multi-line with shebang)
//   - beansRoot: The root directory of the beans project
//   - beanID: The ID of the bean being worked on
//   - beanTitle: The title of the bean
//
// Returns:
//   - *exec.Cmd: The command to execute
//   - *ExecutionResult: Result containing cleanup function
//   - error: Any error that occurred during setup
func CreateExecCommand(execScript, beansRoot, beanID, beanTitle string) (*exec.Cmd, *ExecutionResult, error) {
	env := append(os.Environ(),
		fmt.Sprintf("BEANS_ROOT=%s", beansRoot),
		fmt.Sprintf("BEANS_ID=%s", beanID),
		fmt.Sprintf("BEANS_TASK=%s", beanTitle),
	)

	// Check if this is multi-line (contains newline)
	if !strings.Contains(execScript, "\n") {
		// Single-line: execute via sh -c
		cmd := exec.Command("sh", "-c", execScript)
		cmd.Env = env
		cmd.Dir = beansRoot

		result := &ExecutionResult{
			Cmd:     cmd,
			cleanup: func() {},
		}

		return cmd, result, nil
	}

	// Multi-line: extract shebang and pass script via stdin
	lines := strings.SplitN(execScript, "\n", 2)
	if !strings.HasPrefix(lines[0], "#!") {
		return nil, nil, fmt.Errorf("multi-line script must start with shebang")
	}

	// Extract interpreter from shebang
	shebang := strings.TrimPrefix(lines[0], "#!")
	shebang = strings.TrimSpace(shebang)

	// Parse shebang into command and args
	// e.g., "/bin/bash" or "/usr/bin/env bash" or "/usr/bin/env python3"
	parts := strings.Fields(shebang)
	if len(parts) == 0 {
		return nil, nil, fmt.Errorf("invalid shebang: %s", lines[0])
	}

	// Build command - pass script content via stdin
	var cmd *exec.Cmd
	if len(parts) == 1 {
		cmd = exec.Command(parts[0])
	} else {
		cmd = exec.Command(parts[0], parts[1:]...)
	}

	cmd.Env = env
	cmd.Dir = beansRoot

	// Pass the script (including shebang line) via stdin
	cmd.Stdin = strings.NewReader(execScript)

	// Redirect outputs to avoid TUI interference
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	result := &ExecutionResult{
		Cmd:     cmd,
		cleanup: func() {}, // No cleanup needed
	}

	return cmd, result, nil
}
