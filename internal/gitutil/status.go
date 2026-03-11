package gitutil

import (
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

// HasChanges returns true if there are any uncommitted changes or untracked files.
func HasChanges(dir string) bool {
	changes, err := FileChanges(dir)
	if err != nil {
		return false
	}
	return len(changes) > 0
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
