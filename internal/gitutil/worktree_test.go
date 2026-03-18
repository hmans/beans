package gitutil

import (
	"os/exec"
	"path/filepath"
	"testing"
)

// initTestRepo creates a temporary git repo with an initial commit.
func initTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	commands := [][]string{
		{"git", "init", "-b", "main"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
		{"git", "commit", "--allow-empty", "-m", "initial"},
	}

	for _, args := range commands {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v failed: %s: %v", args, out, err)
		}
	}

	return dir
}

func TestMainWorktreeRoot_MainWorktree(t *testing.T) {
	repoDir := initTestRepo(t)

	root, isSecondary := MainWorktreeRoot(repoDir)
	if isSecondary {
		t.Errorf("expected main worktree, got secondary with root %q", root)
	}
	if root != "" {
		t.Errorf("expected empty root, got %q", root)
	}
}

func TestMainWorktreeRoot_SecondaryWorktree(t *testing.T) {
	repoDir := initTestRepo(t)

	// Create a secondary worktree
	wtPath := filepath.Join(t.TempDir(), "secondary")
	cmd := exec.Command("git", "worktree", "add", wtPath, "-b", "test-branch")
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git worktree add failed: %s: %v", out, err)
	}

	root, isSecondary := MainWorktreeRoot(wtPath)
	if !isSecondary {
		t.Fatal("expected secondary worktree, got main")
	}

	// Resolve repoDir to handle symlinks (e.g., /tmp -> /private/tmp on macOS)
	expectedRoot, err := filepath.EvalSymlinks(repoDir)
	if err != nil {
		t.Fatalf("EvalSymlinks: %v", err)
	}
	actualRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatalf("EvalSymlinks: %v", err)
	}

	if actualRoot != expectedRoot {
		t.Errorf("got root %q, want %q", actualRoot, expectedRoot)
	}
}

func TestDefaultRemoteBranch_WithRemote(t *testing.T) {
	// Create a "remote" repo
	remoteDir := initTestRepo(t)

	// Clone it so we have an origin
	cloneDir := filepath.Join(t.TempDir(), "clone")
	cmd := exec.Command("git", "clone", remoteDir, cloneDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git clone failed: %s: %v", out, err)
	}

	ref, ok := DefaultRemoteBranch(cloneDir, "origin")
	if !ok {
		t.Fatal("expected to detect remote default branch, got ok=false")
	}
	// Default branch from git init is typically "main" (or "master" on older git)
	if ref != "origin/main" && ref != "origin/master" {
		t.Errorf("DefaultRemoteBranch() = %q, want origin/main or origin/master", ref)
	}
}

func TestDefaultRemoteBranch_NoRemote(t *testing.T) {
	// A plain repo with no remote
	repoDir := initTestRepo(t)

	_, ok := DefaultRemoteBranch(repoDir, "origin")
	if ok {
		t.Error("expected ok=false for repo with no remote")
	}
}

func TestDefaultRemoteBranch_NotGitRepo(t *testing.T) {
	dir := t.TempDir()

	_, ok := DefaultRemoteBranch(dir, "origin")
	if ok {
		t.Error("expected ok=false for non-git directory")
	}
}

func TestCurrentBranch_DefaultBranch(t *testing.T) {
	repoDir := initTestRepo(t)

	branch, ok := CurrentBranch(repoDir)
	if !ok {
		t.Fatal("expected ok=true for repo with branch")
	}
	// Default branch from git init is typically "main" or "master"
	if branch != "main" && branch != "master" {
		t.Errorf("CurrentBranch() = %q, want main or master", branch)
	}
}

func TestCurrentBranch_CustomBranch(t *testing.T) {
	repoDir := initTestRepo(t)

	cmd := exec.Command("git", "checkout", "-b", "develop")
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git checkout failed: %s: %v", out, err)
	}

	branch, ok := CurrentBranch(repoDir)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if branch != "develop" {
		t.Errorf("CurrentBranch() = %q, want develop", branch)
	}
}

func TestCurrentBranch_NotGitRepo(t *testing.T) {
	dir := t.TempDir()

	_, ok := CurrentBranch(dir)
	if ok {
		t.Error("expected ok=false for non-git directory")
	}
}

func TestMainWorktreeRoot_NotGitRepo(t *testing.T) {
	dir := t.TempDir()

	root, isSecondary := MainWorktreeRoot(dir)
	if isSecondary {
		t.Errorf("expected not secondary, got secondary with root %q", root)
	}
	if root != "" {
		t.Errorf("expected empty root, got %q", root)
	}
}
