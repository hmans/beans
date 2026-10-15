package beancore

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/hmans/beans/pkg/bean"
)

// worktreeWatcher tracks a single worktree's .beans/ directory.
type worktreeWatcher struct {
	worktreePath string // root of the worktree (e.g., ~/.beans/worktrees/<project>/<id>)
	beansDir     string // .beans/ dir inside the worktree
	done         chan struct{}
}

// WatchWorktreeBeans starts watching a worktree's .beans/ directory for bean changes.
// When beans change in the worktree, they are merged into the runtime state as dirty
// (not persisted to the main repo's disk).
// Returns nil if the worktree's .beans/ directory doesn't exist.
func (c *Core) WatchWorktreeBeans(worktreePath string) error {
	beansDir := filepath.Join(worktreePath, BeansDir)

	// Check if the worktree has a .beans/ directory
	if _, err := os.Stat(beansDir); os.IsNotExist(err) {
		return nil // No .beans/ dir in this worktree, nothing to watch
	}

	c.mu.Lock()
	// Check if already watching this worktree
	if c.worktreeWatchers == nil {
		c.worktreeWatchers = make(map[string]*worktreeWatcher)
	}
	if _, exists := c.worktreeWatchers[worktreePath]; exists {
		c.mu.Unlock()
		return nil // Already watching
	}

	wt := &worktreeWatcher{
		worktreePath: worktreePath,
		beansDir:     beansDir,
		done:         make(chan struct{}),
	}
	c.worktreeWatchers[worktreePath] = wt
	c.mu.Unlock()

	// Create fsnotify watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		c.mu.Lock()
		delete(c.worktreeWatchers, worktreePath)
		c.mu.Unlock()
		return err
	}

	if err := watcher.Add(beansDir); err != nil {
		watcher.Close()
		c.mu.Lock()
		delete(c.worktreeWatchers, worktreePath)
		c.mu.Unlock()
		return err
	}

	// Also watch subdirectories (like archive/)
	_ = filepath.WalkDir(beansDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || !d.IsDir() || path == beansDir {
			return nil
		}
		if strings.HasPrefix(d.Name(), ".") {
			return filepath.SkipDir
		}
		_ = watcher.Add(path)
		return nil
	})

	// Initial scan: load existing bean files from the worktree into runtime state.
	// Without this, beans created in worktrees won't be resolvable via Get()
	// until the watcher picks up a change event.
	c.loadWorktreeBeansInitial(wt)

	// Start the watcher goroutine
	go c.worktreeWatchLoop(wt, watcher)

	c.logWarn("watching worktree beans: %s", beansDir)
	return nil
}

// loadWorktreeBeansInitial scans a worktree's .beans/ directory and loads
// bean files that differ from the main version into runtime state.
// Only beans that are new or modified compared to main are loaded and linked.
func (c *Core) loadWorktreeBeansInitial(wt *worktreeWatcher) {
	entries, err := os.ReadDir(wt.beansDir)
	if err != nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		path := filepath.Join(wt.beansDir, entry.Name())
		newBean, err := c.loadBeanFrom(path, wt.beansDir)
		if err != nil {
			continue
		}

		// Only load and link if the bean differs from the main version.
		// Worktrees branch from main, so most beans are identical copies.
		if existing, exists := c.beans[newBean.ID]; exists {
			if existing.ETag() == newBean.ETag() {
				continue // Same content as main — skip
			}
		}

		c.beans[newBean.ID] = newBean
		c.dirty[newBean.ID] = true
		c.worktreeLinks[newBean.ID] = wt.worktreePath

		if c.searchIndex != nil {
			_ = c.searchIndex.IndexBean(newBean)
		}
	}
}

// UnwatchWorktreeBeans stops watching a worktree's .beans/ directory.
func (c *Core) UnwatchWorktreeBeans(worktreePath string) {
	c.mu.Lock()
	wt, exists := c.worktreeWatchers[worktreePath]
	if exists {
		delete(c.worktreeWatchers, worktreePath)
		// Clear all worktree links for this worktree
		for id, wtp := range c.worktreeLinks {
			if wtp == worktreePath {
				delete(c.worktreeLinks, id)
			}
		}
	}
	c.mu.Unlock()

	if exists {
		close(wt.done)
		c.logWarn("stopped watching worktree beans: %s", wt.beansDir)
	}
}

// UnwatchAllWorktrees stops watching all worktree .beans/ directories.
func (c *Core) UnwatchAllWorktrees() {
	c.mu.Lock()
	watchers := c.worktreeWatchers
	c.worktreeWatchers = make(map[string]*worktreeWatcher)
	c.worktreeLinks = make(map[string]string)
	c.mu.Unlock()

	for _, wt := range watchers {
		close(wt.done)
	}
}

// worktreeWatchLoop processes filesystem events from a worktree with debouncing.
func (c *Core) worktreeWatchLoop(wt *worktreeWatcher, watcher *fsnotify.Watcher) {
	defer watcher.Close()

	var debounceTimer *time.Timer
	var pendingMu sync.Mutex
	pendingChanges := make(map[string]fsnotify.Op)

	for {
		select {
		case <-wt.done:
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			return

		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			// Only care about .md files
			if !strings.HasSuffix(event.Name, ".md") {
				continue
			}

			// Verify the file is within the worktree's .beans/ directory
			relPath, err := filepath.Rel(wt.beansDir, event.Name)
			if err != nil || strings.HasPrefix(relPath, "..") {
				continue
			}

			// Skip events from dot-prefixed subdirectories
			if topDir, _, ok := strings.Cut(relPath, string(filepath.Separator)); ok && strings.HasPrefix(topDir, ".") {
				continue
			}

			// Check if this is a relevant event
			relevant := event.Op&fsnotify.Create != 0 ||
				event.Op&fsnotify.Write != 0 ||
				event.Op&fsnotify.Remove != 0 ||
				event.Op&fsnotify.Rename != 0

			if !relevant {
				continue
			}

			pendingMu.Lock()
			pendingChanges[event.Name] |= event.Op
			pendingMu.Unlock()

			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			debounceTimer = time.AfterFunc(debounceDelay, func() {
				pendingMu.Lock()
				changes := pendingChanges
				pendingChanges = make(map[string]fsnotify.Op)
				pendingMu.Unlock()

				c.handleWorktreeChanges(wt, changes)
			})

		case _, ok := <-watcher.Errors:
			if !ok {
				return
			}
		}
	}
}

// handleWorktreeChanges processes bean file changes from a worktree.
// Creates/writes are merged into runtime state as dirty.
// Deletes either revert to the main-repo version or remove the bean from runtime.
func (c *Core) handleWorktreeChanges(wt *worktreeWatcher, changes map[string]fsnotify.Op) {
	if len(changes) == 0 {
		return
	}

	c.mu.Lock()

	var events []BeanEvent

	for path, op := range changes {
		isDelete := op&fsnotify.Remove != 0 || op&fsnotify.Rename != 0
		isCreateOrWrite := op&fsnotify.Create != 0 || op&fsnotify.Write != 0

		if isDelete && !isCreateOrWrite {
			// File was deleted in the worktree. Either revert to the
			// main-repo version or remove the bean from runtime.
			filename := filepath.Base(path)
			id, _ := bean.ParseFilename(filename)
			if id == "" {
				continue
			}

			// Try to load from the main .beans/ directory.
			// The filename may differ (different slug), so search by ID.
			mainPath := c.findMainBeanFile(id)
			if mainPath != "" {
				mainBean, err := c.loadBeanFrom(mainPath, c.root)
				if err != nil {
					continue
				}
				// Bean exists in main — revert to that version
				c.beans[id] = mainBean
				delete(c.dirty, id)
				delete(c.worktreeLinks, id)

				if c.searchIndex != nil {
					_ = c.searchIndex.IndexBean(mainBean)
				}

				events = append(events, BeanEvent{
					Type:   EventUpdated,
					Bean:   mainBean,
					BeanID: id,
				})
			} else if _, existed := c.beans[id]; existed {
				// Bean was worktree-only — remove from runtime
				delete(c.beans, id)
				delete(c.dirty, id)
				delete(c.worktreeLinks, id)

				if c.searchIndex != nil {
					_ = c.searchIndex.DeleteBean(id)
				}

				events = append(events, BeanEvent{
					Type:   EventDeleted,
					BeanID: id,
				})
			}
			continue
		}

		if !isCreateOrWrite {
			continue
		}

		// Check the file still exists (handles rapid create+delete)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		}

		newBean, err := c.loadBeanFrom(path, wt.beansDir)
		if err != nil {
			c.logWarn("failed to load worktree bean from %s: %v", path, err)
			continue
		}

		// Compare against the main repo's version on disk.
		// If they match (e.g. after a rebase), don't link — this bean
		// isn't actually modified in the worktree.
		mainPath := c.findMainBeanFile(newBean.ID)
		if mainPath != "" {
			mainBean, mainErr := c.loadBeanFrom(mainPath, c.root)
			if mainErr == nil && mainBean.ETag() == newBean.ETag() {
				// Worktree version matches main — clear any stale link
				if _, wasLinked := c.worktreeLinks[newBean.ID]; wasLinked {
					c.beans[newBean.ID] = mainBean
					delete(c.dirty, newBean.ID)
					delete(c.worktreeLinks, newBean.ID)
					if c.searchIndex != nil {
						_ = c.searchIndex.IndexBean(mainBean)
					}
					events = append(events, BeanEvent{
						Type:   EventUpdated,
						Bean:   mainBean,
						BeanID: newBean.ID,
					})
				}
				continue
			}
		}

		_, existed := c.beans[newBean.ID]
		c.beans[newBean.ID] = newBean
		c.dirty[newBean.ID] = true // Mark as dirty — came from worktree, not persisted to main
		c.worktreeLinks[newBean.ID] = wt.worktreePath

		// Update search index
		if c.searchIndex != nil {
			if err := c.searchIndex.IndexBean(newBean); err != nil {
				c.logWarn("failed to index worktree bean %s: %v", newBean.ID, err)
			}
		}

		if existed {
			events = append(events, BeanEvent{
				Type:   EventUpdated,
				Bean:   newBean,
				BeanID: newBean.ID,
			})
		} else {
			events = append(events, BeanEvent{
				Type:   EventCreated,
				Bean:   newBean,
				BeanID: newBean.ID,
			})
		}
	}

	c.mu.Unlock()

	// Fan out events to subscribers
	c.fanOut(events)

	// Notify worktree manager so the worktree subscription re-emits
	// with updated detected bean IDs
	if len(events) > 0 && c.onWorktreeBeansChanged != nil {
		c.onWorktreeBeansChanged()
	}
}

// findMainBeanFile searches the main .beans/ directory for a file matching the
// given bean ID. Returns the full path or empty string if not found.
// Must be called with c.mu held.
func (c *Core) findMainBeanFile(id string) string {
	entries, err := os.ReadDir(c.root)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		fileID, _ := bean.ParseFilename(entry.Name())
		if fileID == id {
			return filepath.Join(c.root, entry.Name())
		}
	}
	return ""
}

// loadBeanFrom reads and parses a bean file, calculating its relative path from the given root.
func (c *Core) loadBeanFrom(path string, root string) (*bean.Bean, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	b, err := bean.Parse(f)
	if err != nil {
		return nil, err
	}

	// Set metadata from path (relative to the root, not the worktree)
	relPath, err := filepath.Rel(root, path)
	if err != nil {
		return nil, err
	}
	b.Path = relPath

	// Extract ID and slug from filename
	filename := filepath.Base(path)
	b.ID, b.Slug = bean.ParseFilename(filename)

	b.ApplyDefaults()
	if b.CreatedAt == nil {
		if b.UpdatedAt != nil {
			b.CreatedAt = b.UpdatedAt
		} else {
			info, statErr := os.Stat(path)
			if statErr == nil {
				modTime := info.ModTime().UTC().Truncate(time.Second)
				b.CreatedAt = &modTime
			}
		}
	}
	if b.UpdatedAt == nil {
		b.UpdatedAt = b.CreatedAt
	}

	return b, nil
}
