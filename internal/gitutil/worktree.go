// Package gitutil provides git-related utility functions.
package gitutil

import (
	"os/exec"
	"path/filepath"
	"strings"
)

// MainWorktreeRoot returns the root directory of the main git worktree
// if the given directory is inside a secondary worktree.
// Returns ("", false) if in the main worktree, not a git repo, or git unavailable.
func MainWorktreeRoot(dir string) (string, bool) {
	commonDir, err := gitRevParse(dir, "--git-common-dir")
	if err != nil {
		return "", false
	}

	gitDir, err := gitRevParse(dir, "--git-dir")
	if err != nil {
		return "", false
	}

	// Resolve to absolute paths for reliable comparison.
	// git rev-parse may return relative or absolute paths.
	commonDir = resolveGitPath(dir, commonDir)
	gitDir = resolveGitPath(dir, gitDir)

	// If they're the same, we're in the main worktree
	if commonDir == gitDir {
		return "", false
	}

	// Main repo root is the parent of the common .git directory
	return filepath.Dir(commonDir), true
}

// resolveGitPath makes a git path absolute. If the path is already absolute,
// it's cleaned and returned as-is. Otherwise it's joined with the base dir.
func resolveGitPath(base, p string) string {
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	return filepath.Join(base, p)
}

// DefaultRemoteBranch returns the default branch ref for the given remote
// (e.g. "origin/main" or "origin/master"). It uses `git symbolic-ref` to
// read the remote's HEAD. Returns ("", false) if not in a git repo, the
// remote doesn't exist, or the remote HEAD hasn't been fetched yet.
func DefaultRemoteBranch(dir, remote string) (string, bool) {
	cmd := exec.Command("git", "-C", dir, "symbolic-ref", "--short", "refs/remotes/"+remote+"/HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(out)), true
}

// CurrentBranch returns the current branch name for the repo at dir.
// Returns ("", false) if not a git repo or HEAD is detached.
func CurrentBranch(dir string) (string, bool) {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", false
	}
	branch := strings.TrimSpace(string(out))
	if branch == "HEAD" {
		return "", false // detached HEAD
	}
	return branch, true
}

func gitRevParse(dir, flag string) (string, error) {
	cmd := exec.Command("git", "-C", dir, "rev-parse", flag)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
