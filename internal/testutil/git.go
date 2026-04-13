package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// InitTestRepo creates a temporary git repo with an initial commit,
// a .beans directory inside it, and a separate worktree root directory.
func InitTestRepo(t *testing.T) (repoDir, beansDir, worktreeRoot string) {
	t.Helper()
	dir := t.TempDir()

	for _, args := range [][]string{
		{"git", "init", "-b", "main"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
		{"git", "commit", "--allow-empty", "-m", "initial"},
	} {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v failed: %s: %v", args, out, err)
		}
	}

	bd := filepath.Join(dir, ".beans")
	if err := os.MkdirAll(bd, 0755); err != nil {
		t.Fatalf("MkdirAll .beans: %v", err)
	}

	wtRoot := t.TempDir()

	return dir, bd, wtRoot
}
