// Package worktree manages git worktrees associated with beans.
package worktree

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

const branchPrefix = "beans/"

// Worktree represents a git worktree associated with a bean.
type Worktree struct {
	BeanID string
	Branch string
	Path   string
}

// Manager handles git worktree operations for a repository.
type Manager struct {
	repoRoot string
	mu       sync.RWMutex

	// subscribers for worktree change events
	subMu       sync.Mutex
	subscribers []chan struct{}
}

// NewManager creates a new worktree manager for the given repository root.
func NewManager(repoRoot string) *Manager {
	return &Manager{repoRoot: repoRoot}
}

// Subscribe returns a channel that receives a signal whenever worktrees change.
// The caller should call Unsubscribe when done.
func (m *Manager) Subscribe() chan struct{} {
	m.subMu.Lock()
	defer m.subMu.Unlock()
	ch := make(chan struct{}, 1)
	m.subscribers = append(m.subscribers, ch)
	return ch
}

// Unsubscribe removes a subscription channel.
func (m *Manager) Unsubscribe(ch chan struct{}) {
	m.subMu.Lock()
	defer m.subMu.Unlock()
	for i, sub := range m.subscribers {
		if sub == ch {
			m.subscribers = append(m.subscribers[:i], m.subscribers[i+1:]...)
			close(ch)
			return
		}
	}
}

// notify sends a signal to all subscribers.
func (m *Manager) notify() {
	m.subMu.Lock()
	defer m.subMu.Unlock()
	for _, ch := range m.subscribers {
		select {
		case ch <- struct{}{}:
		default:
			// Non-blocking: if the channel already has a pending signal, skip
		}
	}
}

// List returns all active worktrees that were created by beans (branch prefix "beans/").
func (m *Manager) List() ([]Worktree, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = m.repoRoot
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git worktree list: %w", err)
	}

	return parsePorcelain(string(out)), nil
}

// parsePorcelain parses `git worktree list --porcelain` output and returns
// worktrees whose branch starts with the beans prefix.
// Entries marked as "prunable" (stale/missing directory) are skipped.
func parsePorcelain(output string) []Worktree {
	var worktrees []Worktree
	var currentPath, currentBranch string
	var prunable bool

	emit := func() {
		if !prunable && currentPath != "" && strings.HasPrefix(currentBranch, branchPrefix) {
			beanID := strings.TrimPrefix(currentBranch, branchPrefix)
			worktrees = append(worktrees, Worktree{
				BeanID: beanID,
				Branch: currentBranch,
				Path:   currentPath,
			})
		}
		currentPath = ""
		currentBranch = ""
		prunable = false
	}

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "worktree ") {
			currentPath = strings.TrimPrefix(line, "worktree ")
			currentBranch = ""
			prunable = false
		} else if strings.HasPrefix(line, "branch ") {
			ref := strings.TrimPrefix(line, "branch ")
			// ref is like "refs/heads/beans/beans-abc1"
			currentBranch = strings.TrimPrefix(ref, "refs/heads/")
		} else if strings.HasPrefix(line, "prunable ") {
			prunable = true
		} else if line == "" {
			emit()
		}
	}

	// Handle last entry (porcelain output may not end with blank line)
	emit()

	return worktrees
}

// Create creates a new git worktree for the given bean ID.
// The worktree is placed as a sibling of the repo root, named <dirname>-<beanID>.
// If the branch beans/<beanID> already exists, it is reused; otherwise a new branch
// is created from the current HEAD.
func (m *Manager) Create(beanID string) (*Worktree, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	branch := branchPrefix + beanID
	worktreePath := m.worktreePath(beanID)

	// Check if the worktree path already exists
	if _, err := os.Stat(worktreePath); err == nil {
		log.Printf("[worktree] failed to create worktree for %s: path already exists: %s", beanID, worktreePath)
		return nil, fmt.Errorf("worktree path already exists: %s", worktreePath)
	}

	// Try creating with a new branch first; if the branch already exists
	// (e.g. from a previously removed worktree), reuse it.
	cmd := exec.Command("git", "worktree", "add", worktreePath, "-b", branch)
	cmd.Dir = m.repoRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		if !strings.Contains(string(out), "already exists") {
			log.Printf("[worktree] failed to create worktree for %s at %s: %s: %v", beanID, worktreePath, strings.TrimSpace(string(out)), err)
			return nil, fmt.Errorf("git worktree add: %s: %w", strings.TrimSpace(string(out)), err)
		}

		// Branch exists — reuse it
		log.Printf("[worktree] branch %s already exists, reusing for worktree", branch)
		cmd = exec.Command("git", "worktree", "add", worktreePath, branch)
		cmd.Dir = m.repoRoot
		if out, err := cmd.CombinedOutput(); err != nil {
			log.Printf("[worktree] failed to create worktree for %s at %s: %s: %v", beanID, worktreePath, strings.TrimSpace(string(out)), err)
			return nil, fmt.Errorf("git worktree add: %s: %w", strings.TrimSpace(string(out)), err)
		}
	}

	wt := &Worktree{
		BeanID: beanID,
		Branch: branch,
		Path:   worktreePath,
	}

	log.Printf("[worktree] created worktree for %s (branch=%s, path=%s)", beanID, branch, worktreePath)
	m.notify()
	return wt, nil
}

// Remove removes the worktree for the given bean ID.
// The actual worktree path is looked up from git (not computed), so this works
// even when the worktree was created from a different repo root/workspace.
// If the worktree directory is already gone (stale entry), it prunes instead.
func (m *Manager) Remove(beanID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Look up the actual path from git rather than computing it,
	// since the worktree may have been created from a different workspace.
	worktreePath, err := m.findWorktreePath(beanID)
	if err != nil {
		// Worktree not found in active list — it may be stale (prunable).
		// Run git worktree prune to clean up stale entries.
		log.Printf("[worktree] worktree for %s not found in active list, pruning stale entries", beanID)
		pruneCmd := exec.Command("git", "worktree", "prune")
		pruneCmd.Dir = m.repoRoot
		if pruneOut, pruneErr := pruneCmd.CombinedOutput(); pruneErr != nil {
			log.Printf("[worktree] failed to prune worktrees: %s: %v", strings.TrimSpace(string(pruneOut)), pruneErr)
			return fmt.Errorf("git worktree prune: %s: %w", strings.TrimSpace(string(pruneOut)), pruneErr)
		}
		log.Printf("[worktree] pruned stale worktree entries for %s", beanID)
		m.notify()
		return nil
	}

	cmd := exec.Command("git", "worktree", "remove", worktreePath)
	cmd.Dir = m.repoRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		outStr := strings.TrimSpace(string(out))

		// If the directory is already gone, git worktree remove fails with
		// "is not a working tree". Prune stale entries instead.
		if strings.Contains(outStr, "is not a working tree") {
			log.Printf("[worktree] worktree for %s is stale, pruning", beanID)
			pruneCmd := exec.Command("git", "worktree", "prune")
			pruneCmd.Dir = m.repoRoot
			if pruneOut, pruneErr := pruneCmd.CombinedOutput(); pruneErr != nil {
				log.Printf("[worktree] failed to prune worktrees: %s: %v", strings.TrimSpace(string(pruneOut)), pruneErr)
				return fmt.Errorf("git worktree prune: %s: %w", strings.TrimSpace(string(pruneOut)), pruneErr)
			}
			log.Printf("[worktree] pruned stale worktree for %s", beanID)
			m.notify()
			return nil
		}

		log.Printf("[worktree] failed to remove worktree for %s at %s: %s: %v", beanID, worktreePath, outStr, err)
		return fmt.Errorf("git worktree remove: %s: %w", outStr, err)
	}

	log.Printf("[worktree] removed worktree for %s (path=%s)", beanID, worktreePath)
	m.notify()
	return nil
}

// findWorktreePath looks up the actual filesystem path for a bean's worktree
// by parsing git worktree list output. This is needed because the worktree may
// have been created from a different workspace/repo root.
// Must be called with m.mu held.
func (m *Manager) findWorktreePath(beanID string) (string, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = m.repoRoot
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git worktree list: %w", err)
	}

	for _, wt := range parsePorcelain(string(out)) {
		if wt.BeanID == beanID {
			return wt.Path, nil
		}
	}
	return "", fmt.Errorf("no worktree for bean %s", beanID)
}

// worktreePath returns the path for a worktree associated with a bean.
func (m *Manager) worktreePath(beanID string) string {
	dirName := filepath.Base(m.repoRoot)
	return filepath.Join(filepath.Dir(m.repoRoot), dirName+"-"+beanID)
}
