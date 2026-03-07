// Package worktree manages git worktrees associated with beans.
package worktree

import (
	"bufio"
	"fmt"
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
func parsePorcelain(output string) []Worktree {
	var worktrees []Worktree
	var currentPath, currentBranch string

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "worktree ") {
			currentPath = strings.TrimPrefix(line, "worktree ")
			currentBranch = ""
		} else if strings.HasPrefix(line, "branch ") {
			ref := strings.TrimPrefix(line, "branch ")
			// ref is like "refs/heads/beans/beans-abc1"
			currentBranch = strings.TrimPrefix(ref, "refs/heads/")
		} else if line == "" {
			// End of entry
			if strings.HasPrefix(currentBranch, branchPrefix) {
				beanID := strings.TrimPrefix(currentBranch, branchPrefix)
				worktrees = append(worktrees, Worktree{
					BeanID: beanID,
					Branch: currentBranch,
					Path:   currentPath,
				})
			}
			currentPath = ""
			currentBranch = ""
		}
	}

	// Handle last entry (porcelain output may not end with blank line)
	if currentPath != "" && strings.HasPrefix(currentBranch, branchPrefix) {
		beanID := strings.TrimPrefix(currentBranch, branchPrefix)
		worktrees = append(worktrees, Worktree{
			BeanID: beanID,
			Branch: currentBranch,
			Path:   currentPath,
		})
	}

	return worktrees
}

// Create creates a new git worktree for the given bean ID.
// The worktree is placed as a sibling of the repo root, named <dirname>-<beanID>.
// A new branch beans/<beanID> is created from the current HEAD.
func (m *Manager) Create(beanID string) (*Worktree, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	branch := branchPrefix + beanID
	worktreePath := m.worktreePath(beanID)

	// Check if the worktree path already exists
	if _, err := os.Stat(worktreePath); err == nil {
		return nil, fmt.Errorf("worktree path already exists: %s", worktreePath)
	}

	cmd := exec.Command("git", "worktree", "add", worktreePath, "-b", branch)
	cmd.Dir = m.repoRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("git worktree add: %s: %w", strings.TrimSpace(string(out)), err)
	}

	wt := &Worktree{
		BeanID: beanID,
		Branch: branch,
		Path:   worktreePath,
	}

	m.notify()
	return wt, nil
}

// Remove removes the worktree for the given bean ID.
func (m *Manager) Remove(beanID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	worktreePath := m.worktreePath(beanID)

	cmd := exec.Command("git", "worktree", "remove", worktreePath)
	cmd.Dir = m.repoRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git worktree remove: %s: %w", strings.TrimSpace(string(out)), err)
	}

	m.notify()
	return nil
}

// worktreePath returns the path for a worktree associated with a bean.
func (m *Manager) worktreePath(beanID string) string {
	dirName := filepath.Base(m.repoRoot)
	return filepath.Join(filepath.Dir(m.repoRoot), dirName+"-"+beanID)
}
