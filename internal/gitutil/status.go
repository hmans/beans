package gitutil

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// FileChange represents a single changed file with diff stats.
type FileChange struct {
	Path      string
	Status    string // "modified", "added", "deleted", "untracked", "renamed"
	Additions int
	Deletions int
	Staged    bool
}

// FileChanges returns the list of changed files in the given directory,
// combining staged changes, unstaged changes, and untracked files.
func FileChanges(dir string) ([]FileChange, error) {
	var changes []FileChange

	// Staged changes
	staged, err := diffNumstat(dir, true)
	if err != nil {
		return nil, err
	}
	changes = append(changes, staged...)

	// Unstaged changes (only files not already covered by staged)
	unstaged, err := diffNumstat(dir, false)
	if err != nil {
		return nil, err
	}
	changes = append(changes, unstaged...)

	// Untracked files
	untracked, err := untrackedFiles(dir)
	if err != nil {
		return nil, err
	}
	changes = append(changes, untracked...)

	return changes, nil
}

// diffNumstat runs git diff --numstat (with or without --cached) and parses
// the output into FileChange structs.
func diffNumstat(dir string, cached bool) ([]FileChange, error) {
	args := []string{"-C", dir, "diff", "--numstat"}
	if cached {
		args = append(args, "--cached")
	}

	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return parseNumstat(string(out), cached)
}

// parseNumstat parses git diff --numstat output. Each line is:
//
//	<additions>\t<deletions>\t<path>
//
// Binary files show "-" for additions/deletions.
func parseNumstat(output string, staged bool) ([]FileChange, error) {
	output = strings.TrimSpace(output)
	if output == "" {
		return nil, nil
	}

	var changes []FileChange
	for _, line := range strings.Split(output, "\n") {
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) < 3 {
			continue
		}

		adds, _ := strconv.Atoi(parts[0]) // "-" for binary → 0
		dels, _ := strconv.Atoi(parts[1])
		path := parts[2]

		// Detect renames: git uses "old => new" or "{old => new}/rest" syntax
		status := "modified"
		if strings.Contains(path, " => ") {
			status = "renamed"
		}

		changes = append(changes, FileChange{
			Path:      path,
			Status:    status,
			Additions: adds,
			Deletions: dels,
			Staged:    staged,
		})
	}

	return changes, nil
}

// MergeBase returns the merge-base commit between HEAD and the given base ref.
// If baseRef is empty, falls back to the default remote branch (e.g. origin/main).
// Returns ("", false) if it can't be determined.
func MergeBase(dir, baseRef string) (string, bool) {
	if baseRef == "" {
		remote, ok := DefaultRemoteBranch(dir, "origin")
		if !ok {
			return "", false
		}
		baseRef = remote
	}
	cmd := exec.Command("git", "-C", dir, "merge-base", "HEAD", baseRef)
	out, err := cmd.Output()
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(out)), true
}

// AllChangesVsUpstream returns all file changes compared to the
// merge-base: committed + staged + unstaged + untracked, deduplicated into
// a single entry per file showing the total diff from merge-base to working tree.
// baseRef is the branch to compare against (e.g. "main"); if empty, falls back
// to the default remote branch.
func AllChangesVsUpstream(dir, baseRef string) ([]FileChange, error) {
	base, ok := MergeBase(dir, baseRef)
	if !ok {
		// Fallback: if no merge-base, just return regular working tree changes
		return FileChanges(dir)
	}

	// Diff from merge-base to working tree (includes committed + staged + unstaged)
	args := []string{"-C", dir, "diff", "--numstat", base}
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	changes, err := parseNumstat(string(out), false)
	if err != nil {
		return nil, err
	}

	// Also include untracked files (not covered by git diff)
	untracked, err := untrackedFiles(dir)
	if err != nil {
		return nil, err
	}
	changes = append(changes, untracked...)

	// Detect added files: files that don't exist at the merge-base
	existsAtBase := make(map[string]bool)
	lsCmd := exec.Command("git", "-C", dir, "ls-tree", "--name-only", "-r", base)
	lsOut, err := lsCmd.Output()
	if err == nil {
		for _, p := range strings.Split(strings.TrimSpace(string(lsOut)), "\n") {
			if p != "" {
				existsAtBase[p] = true
			}
		}
		for i := range changes {
			if changes[i].Status == "modified" && !existsAtBase[changes[i].Path] {
				changes[i].Status = "added"
			}
		}
	}

	return changes, nil
}

// AllFileDiff returns the unified diff for a file compared to the
// merge-base. This shows the complete change from merge-base to working tree.
// baseRef is the branch to compare against; if empty, falls back to the default
// remote branch.
func AllFileDiff(dir, filePath, baseRef string) (string, error) {
	base, ok := MergeBase(dir, baseRef)
	if !ok {
		// Fallback to regular unstaged diff
		return FileDiff(dir, filePath, false)
	}

	// Check if the file is untracked
	cmd := exec.Command("git", "-C", dir, "ls-files", "--others", "--exclude-standard", "--", filePath)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(string(out)) != "" {
		// Untracked file — show full content as a diff
		cmd = exec.Command("git", "-C", dir, "diff", "--no-index", "/dev/null", filePath)
		out, err = cmd.Output()
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
				return string(out), nil
			}
			return "", err
		}
		return string(out), nil
	}

	// Diff from merge-base to working tree for this file
	cmd = exec.Command("git", "-C", dir, "diff", base, "--", filePath)
	out, err = cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// HasUnmergedCommits returns true if the current branch in dir has commits
// that are not in the given base branch (i.e., commits ahead).
func HasUnmergedCommits(dir, baseBranch string) bool {
	cmd := exec.Command("git", "-C", dir, "rev-list", "--count", baseBranch+"..HEAD")
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	count, _ := strconv.Atoi(strings.TrimSpace(string(out)))
	return count > 0
}

// CommitsBehind returns the number of commits on baseBranch that are not
// reachable from HEAD (i.e., how far behind the worktree branch is).
func CommitsBehind(dir, baseBranch string) int {
	if baseBranch == "" {
		remote, ok := DefaultRemoteBranch(dir, "origin")
		if !ok {
			return 0
		}
		baseBranch = remote
	}
	cmd := exec.Command("git", "-C", dir, "rev-list", "--count", "HEAD.."+baseBranch)
	out, err := cmd.Output()
	if err != nil {
		return 0
	}
	count, _ := strconv.Atoi(strings.TrimSpace(string(out)))
	return count
}

// HasConflicts checks whether rebasing the current branch onto baseBranch
// would produce merge conflicts, without actually modifying the working tree.
// It performs a tree-level merge in memory using git merge-tree.
func HasConflicts(dir, baseBranch string) bool {
	if baseBranch == "" {
		remote, ok := DefaultRemoteBranch(dir, "origin")
		if !ok {
			return false
		}
		baseBranch = remote
	}
	// git merge-tree --write-tree exits 0 if clean, 1 if conflicts
	cmd := exec.Command("git", "-C", dir, "merge-tree", "--write-tree", "--no-messages", baseBranch, "HEAD")
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return true
		}
		// Other errors (e.g., git too old) — assume no conflicts
		return false
	}
	return false
}

// HasUnpushedCommits returns true if the current branch has commits
// that have not been pushed to its remote tracking branch.
// Returns false if there is no tracking branch (e.g., branch was never pushed).
func HasUnpushedCommits(dir string) bool {
	// Check if upstream tracking branch exists
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--abbrev-ref", "@{upstream}")
	if err := cmd.Run(); err != nil {
		// No tracking branch — treat as "has unpushed" since nothing was pushed yet
		return true
	}
	// Count commits ahead of upstream
	cmd = exec.Command("git", "-C", dir, "rev-list", "--count", "@{upstream}..HEAD")
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	count, _ := strconv.Atoi(strings.TrimSpace(string(out)))
	return count > 0
}

// HasChanges returns true if there are any uncommitted changes or untracked files.
func HasChanges(dir string) bool {
	changes, err := FileChanges(dir)
	if err != nil {
		return false
	}
	return len(changes) > 0
}

// FileDiff returns the unified diff for a specific file.
// If staged is true, shows the staged diff (--cached); otherwise shows the working tree diff.
// For untracked files, it shows the full file content as an added diff.
func FileDiff(dir, filePath string, staged bool) (string, error) {
	if staged {
		// Staged diff
		cmd := exec.Command("git", "-C", dir, "diff", "--cached", "--", filePath)
		out, err := cmd.Output()
		if err != nil {
			return "", err
		}
		return string(out), nil
	}

	// Check if the file is untracked
	cmd := exec.Command("git", "-C", dir, "ls-files", "--others", "--exclude-standard", "--", filePath)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(string(out)) != "" {
		// Untracked file — show full content as a diff-like output
		cmd = exec.Command("git", "-C", dir, "diff", "--no-index", "/dev/null", filePath)
		out, err = cmd.Output()
		// git diff --no-index exits with 1 when files differ (which they always will)
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
				return string(out), nil
			}
			return "", err
		}
		return string(out), nil
	}

	// Unstaged diff for tracked file
	cmd = exec.Command("git", "-C", dir, "diff", "--", filePath)
	out, err = cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// DiscardFileChange discards a change to a single file.
// If staged is true, the file is unstaged first. Untracked files are removed;
// tracked files are restored to their committed state.
func DiscardFileChange(dir, filePath string, staged bool) error {
	if staged {
		// Unstage the file first
		cmd := exec.Command("git", "-C", dir, "reset", "HEAD", "--", filePath)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("unstage file: %s", strings.TrimSpace(string(out)))
		}
	}

	// Check if the file is untracked
	cmd := exec.Command("git", "-C", dir, "ls-files", "--others", "--exclude-standard", "--", filePath)
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("check untracked status: %w", err)
	}
	if strings.TrimSpace(string(out)) != "" {
		// Untracked file — remove it
		cmd = exec.Command("git", "-C", dir, "clean", "-f", "--", filePath)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("clean untracked file: %s", strings.TrimSpace(string(out)))
		}
		return nil
	}

	// Tracked file — restore working tree copy
	cmd = exec.Command("git", "-C", dir, "checkout", "--", filePath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("restore file: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

// untrackedFiles returns untracked files via git ls-files.
func untrackedFiles(dir string) ([]FileChange, error) {
	cmd := exec.Command("git", "-C", dir, "ls-files", "--others", "--exclude-standard")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	output := strings.TrimSpace(string(out))
	if output == "" {
		return nil, nil
	}

	var changes []FileChange
	for _, path := range strings.Split(output, "\n") {
		if path == "" {
			continue
		}
		changes = append(changes, FileChange{
			Path:   path,
			Status: "untracked",
			Staged: false,
		})
	}

	return changes, nil
}
