// Package worktree manages git worktrees associated with beans.
package worktree

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/hmans/beans/pkg/bean"
)

const branchPrefix = "beans/"

// Worktree represents a git worktree.
type Worktree struct {
	ID     string
	Branch string
	Path   string
	Name   string // Human-readable name
}

// Manager handles git worktree operations for a repository.
type Manager struct {
	repoRoot string
	beansDir string
	baseRef  string
	mu       sync.RWMutex

	// subscribers for worktree change events
	subMu       sync.Mutex
	subscribers []chan struct{}
}

// NewManager creates a new worktree manager for the given repository root.
// beansDir is the path to the .beans directory where worktrees are stored.
// baseRef is the git ref to use as the starting point for new branches (e.g. "main").
func NewManager(repoRoot, beansDir, baseRef string) *Manager {
	return &Manager{repoRoot: repoRoot, beansDir: beansDir, baseRef: baseRef}
}

// BaseRef returns the configured base ref for worktree branches.
func (m *Manager) BaseRef() string {
	return m.baseRef
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

	worktrees := parsePorcelain(string(out))

	// Enrich with metadata (name for standalone worktrees)
	for i := range worktrees {
		if meta := m.loadMeta(worktrees[i].ID); meta != nil {
			worktrees[i].Name = meta.Name
		}
	}

	return worktrees, nil
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
			id := strings.TrimPrefix(currentBranch, branchPrefix)
			worktrees = append(worktrees, Worktree{
				ID: id,
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

// Create creates a new git worktree with the given name.
// It generates a unique ID and stores the human-readable name as metadata.
// The worktree is placed inside .beans/.worktrees/<id>.
func (m *Manager) Create(name string) (*Worktree, error) {
	if name == "" {
		return nil, fmt.Errorf("worktree name must not be empty")
	}

	// Generate a unique ID with "wt-" prefix
	id := "wt-" + bean.NewID("", 4)

	m.mu.Lock()
	defer m.mu.Unlock()

	branch := branchPrefix + id
	worktreePath := m.worktreePath(id)

	// Check if the worktree path already exists
	if _, err := os.Stat(worktreePath); err == nil {
		return nil, fmt.Errorf("worktree path already exists: %s", worktreePath)
	}

	// Create the worktree with a new branch
	args := []string{"worktree", "add", worktreePath, "-b", branch}
	if m.baseRef != "" {
		args = append(args, m.baseRef)
	}
	cmd := exec.Command("git", args...)
	cmd.Dir = m.repoRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Printf("[worktree] failed to create worktree %s at %s: %s: %v", id, worktreePath, strings.TrimSpace(string(out)), err)
		return nil, fmt.Errorf("git worktree add: %s: %w", strings.TrimSpace(string(out)), err)
	}

	// Save the name metadata
	if err := m.saveMeta(id, &worktreeMeta{Name: name}); err != nil {
		log.Printf("[worktree] warning: failed to save metadata for %s: %v", id, err)
	}

	wt := &Worktree{
		ID:     id,
		Branch: branch,
		Path:   worktreePath,
		Name:   name,
	}

	log.Printf("[worktree] created worktree %s (name=%s, branch=%s, path=%s)", id, name, branch, worktreePath)
	m.notify()
	return wt, nil
}

// worktreeMeta is the metadata stored alongside standalone worktrees.
type worktreeMeta struct {
	Name string `json:"name"`
}

// metaPath returns the path to the metadata file for a worktree ID.
func (m *Manager) metaPath(id string) string {
	return filepath.Join(m.beansDir, ".worktrees", id+".meta.json")
}

// loadMeta loads the metadata for a worktree, if it exists.
func (m *Manager) loadMeta(id string) *worktreeMeta {
	data, err := os.ReadFile(m.metaPath(id))
	if err != nil {
		return nil
	}
	var meta worktreeMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil
	}
	return &meta
}

// saveMeta saves metadata for a worktree.
func (m *Manager) saveMeta(id string, meta *worktreeMeta) error {
	data, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	return os.WriteFile(m.metaPath(id), data, 0644)
}

// removeMeta removes the metadata file for a worktree.
func (m *Manager) removeMeta(id string) {
	os.Remove(m.metaPath(id))
}


// Remove removes the worktree with the given ID.
// The actual worktree path is looked up from git (not computed), so this works
// even when the worktree was created from a different repo root/workspace.
func (m *Manager) Remove(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Look up the actual path from git rather than computing it,
	// since the worktree may have been created from a different workspace.
	worktreePath, err := m.findWorktreePathByID(id)
	if err != nil {
		return fmt.Errorf("worktree %s not found: %w", id, err)
	}

	cmd := exec.Command("git", "worktree", "remove", worktreePath)
	cmd.Dir = m.repoRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		outStr := strings.TrimSpace(string(out))
		log.Printf("[worktree] failed to remove worktree %s at %s: %s: %v", id, worktreePath, outStr, err)
		return fmt.Errorf("git worktree remove: %s: %w", outStr, err)
	}

	log.Printf("[worktree] removed worktree %s (path=%s)", id, worktreePath)
	m.removeMeta(id)
	m.notify()
	return nil
}

// findWorktreePathByID looks up the actual filesystem path for a worktree
// by parsing git worktree list output.
// Must be called with m.mu held.
func (m *Manager) findWorktreePathByID(id string) (string, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = m.repoRoot
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git worktree list: %w", err)
	}

	for _, wt := range parsePorcelain(string(out)) {
		if wt.ID == id {
			return wt.Path, nil
		}
	}
	return "", fmt.Errorf("no worktree with id %s", id)
}

// worktreePath returns the filesystem path for a worktree with the given ID.
// Worktrees are stored inside the .beans/.worktrees/ directory.
func (m *Manager) worktreePath(id string) string {
	return filepath.Join(m.beansDir, ".worktrees", id)
}
