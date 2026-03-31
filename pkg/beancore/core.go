// Package beancore provides a thread-safe in-memory store for beans with filesystem persistence
// and optional file watching for long-running processes.
package beancore

import (
	"encoding/hex"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/hmans/beans/pkg/bean"
	"github.com/hmans/beans/pkg/config"
	"github.com/hmans/beans/internal/search"
)

const BeansDir = ".beans"
const ArchiveDir = "archive"

var ErrNotFound = errors.New("bean not found")

// ETagMismatchError is returned when an ETag validation fails.
// This allows callers to distinguish concurrency conflicts from other errors.
type ETagMismatchError struct {
	Provided string
	Current  string
}

func (e *ETagMismatchError) Error() string {
	return fmt.Sprintf("etag mismatch: provided %s, current is %s", e.Provided, e.Current)
}

// ETagRequiredError is returned when require_if_match is enabled and no ETag is provided.
type ETagRequiredError struct{}

func (e *ETagRequiredError) Error() string {
	return "if-match etag is required (set require_if_match: false in config to disable)"
}

// Core provides thread-safe in-memory storage for beans with filesystem persistence.
type Core struct {
	root   string         // absolute path to .beans directory
	config *config.Config // project configuration

	// In-memory state
	mu             sync.RWMutex
	beans          map[string]*bean.Bean // ID -> Bean
	dirty          map[string]bool       // IDs of beans modified in runtime but not yet persisted to disk
	worktreeLinks  map[string]string     // bean ID -> worktree path (beans linked to a worktree)

	// Search index (optional, lazy-initialized)
	searchIndex *search.Index

	// File watching (optional)
	watching          bool
	done              chan struct{}
	onChange          func() // callback when beans change (legacy API)
	worktreeWatchers       map[string]*worktreeWatcher // worktree path -> watcher
	onWorktreeBeansChanged func() // called when worktree bean files change

	// Event subscribers (for channel-based API)
	subscribers map[uint64]*subscription
	subMu       sync.RWMutex
	nextSubID   uint64

	// Warning logger for non-fatal errors (defaults to stderr)
	warnWriter io.Writer
}

// New creates a new Core with the given root path and configuration.
func New(root string, cfg *config.Config) *Core {
	// Resolve symlinks so that c.root matches the real paths returned by
	// filepath.WalkDir. Without this, walking a symlinked directory (e.g.
	// .beans -> repo/beans/.beans) causes the dot-prefix guard in
	// loadFromDisk to skip the entire root because path != c.root.
	if resolved, err := filepath.EvalSymlinks(root); err == nil {
		root = resolved
	}
	return &Core{
		root:        root,
		config:      cfg,
		beans:         make(map[string]*bean.Bean),
		dirty:         make(map[string]bool),
		worktreeLinks: make(map[string]string),
		subscribers: make(map[uint64]*subscription),
		warnWriter:  os.Stderr,
	}
}

// SetOnWorktreeBeansChanged sets a callback that fires whenever bean files
// change in a watched worktree. This is used to trigger worktree subscription
// refreshes so the frontend sees updated bean lists.
func (c *Core) SetOnWorktreeBeansChanged(fn func()) {
	c.onWorktreeBeansChanged = fn
}

// SetWarnWriter sets the writer for warning messages.
// Pass nil to disable warnings.
func (c *Core) SetWarnWriter(w io.Writer) {
	c.warnWriter = w
}

// logWarn logs a warning message if a warn writer is configured.
func (c *Core) logWarn(format string, args ...any) {
	if c.warnWriter != nil {
		fmt.Fprintf(c.warnWriter, "warning: "+format+"\n", args...)
	}
}

// Root returns the absolute path to the .beans directory.
func (c *Core) Root() string {
	return c.root
}

// Config returns the configuration.
func (c *Core) Config() *config.Config {
	return c.config
}

// Load reads all beans from disk into memory.
func (c *Core) Load() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.loadFromDisk()
}

// loadFromDisk reads all beans from disk (must be called with lock held).
// Loads all .md files from the root directory and any subdirectories.
func (c *Core) loadFromDisk() error {
	// Migrate legacy directory names (worktrees/ → .worktrees/, conversations/ → .conversations/)
	c.migrateLegacyDirs()

	// Clear existing beans and dirty state
	c.beans = make(map[string]*bean.Bean)
	c.dirty = make(map[string]bool)

	// Walk the .beans directory tree, loading all .md files
	err := filepath.WalkDir(c.root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip dot-prefixed subdirectories (e.g. .worktrees/, .conversations/)
		// — these contain non-bean data and should never be walked.
		if d.IsDir() && path != c.root {
			if strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip non-.md files
		if !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}

		b, loadErr := c.loadBean(path)
		if loadErr != nil {
			return fmt.Errorf("loading %s: %w", path, loadErr)
		}

		c.beans[b.ID] = b
		return nil
	})
	if err != nil {
		return err
	}

	// Reinitialize search index if it was active: close and re-create (best-effort, don't fail load)
	if c.searchIndex != nil {
		c.searchIndex.Close()
		c.searchIndex = nil

		if err := c.ensureSearchIndexLocked(); err != nil {
			c.logWarn("failed to reinitialize search index after reload: %v", err)
		}
	}

	return nil
}

// loadBean reads and parses a single bean file.
func (c *Core) loadBean(path string) (*bean.Bean, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	b, err := bean.Parse(f)
	if err != nil {
		return nil, err
	}

	// Set metadata from path
	relPath, err := filepath.Rel(c.root, path)
	if err != nil {
		return nil, err
	}
	b.Path = relPath

	// Extract ID and slug from filename
	filename := filepath.Base(path)
	b.ID, b.Slug = bean.ParseFilename(filename)

	// Apply defaults for GraphQL non-nullable fields
	if b.Type == "" {
		b.Type = "task"
	}
	if b.Priority == "" {
		b.Priority = "normal"
	}
	if b.Tags == nil {
		b.Tags = []string{}
	}
	if b.Blocking == nil {
		b.Blocking = []string{}
	}
	if b.CreatedAt == nil {
		if b.UpdatedAt != nil {
			b.CreatedAt = b.UpdatedAt
		} else {
			// Use file modification time as fallback
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

// ensureSearchIndexLocked initializes the in-memory search index if not already created.
// Must be called with lock held or from a method that holds the lock.
func (c *Core) ensureSearchIndexLocked() error {
	if c.searchIndex != nil {
		return nil
	}

	idx, err := search.NewIndex()
	if err != nil {
		return fmt.Errorf("initializing search index: %w", err)
	}

	c.searchIndex = idx

	// Populate the in-memory index with existing beans
	allBeans := make([]*bean.Bean, 0, len(c.beans))
	for _, b := range c.beans {
		allBeans = append(allBeans, b)
	}
	if err := c.searchIndex.IndexBeans(allBeans); err != nil {
		return fmt.Errorf("populating search index: %w", err)
	}

	return nil
}

// Search performs full-text search and returns matching beans.
// The search index is lazily initialized on first use.
func (c *Core) Search(query string) ([]*bean.Bean, error) {
	// Ensure index is initialized (needs write lock for lazy init)
	c.mu.Lock()
	if err := c.ensureSearchIndexLocked(); err != nil {
		c.mu.Unlock()
		return nil, err
	}
	// Capture searchIndex reference while holding lock
	idx := c.searchIndex
	c.mu.Unlock()

	// Perform search outside the lock (Bleve is thread-safe)
	ids, err := idx.Search(query, search.DefaultSearchLimit)
	if err != nil {
		return nil, err
	}

	// Read from beans map (needs read lock only)
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]*bean.Bean, 0, len(ids))
	for _, id := range ids {
		if b, ok := c.beans[id]; ok {
			result = append(result, b)
		}
	}
	return result, nil
}

// All returns a slice of all beans.
func (c *Core) All() []*bean.Bean {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]*bean.Bean, 0, len(c.beans))
	for _, b := range c.beans {
		result = append(result, b)
	}
	return result
}

// Get finds a bean by exact ID match.
// If a prefix is configured and the query doesn't include it, the prefix is automatically prepended.
// For example, with prefix "beans-", Get("abc") will match "beans-abc" but Get("ab") will not.
func (c *Core) Get(id string) (*bean.Bean, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Try exact match
	if b, ok := c.beans[id]; ok {
		return b, nil
	}

	// If not found and we have a configured prefix that isn't already in the query,
	// try with the prefix prepended (allows short IDs like "abc" to match "beans-abc")
	if c.config != nil && c.config.Beans.Prefix != "" && !strings.HasPrefix(id, c.config.Beans.Prefix) {
		if b, ok := c.beans[c.config.Beans.Prefix+id]; ok {
			return b, nil
		}
	}

	return nil, ErrNotFound
}

// NormalizeID resolves a potentially short ID to its full form.
// If a prefix is configured and the query doesn't include it, the prefix is automatically prepended.
// Returns the full ID and true if found, or the original ID and false if not found.
func (c *Core) NormalizeID(id string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Try exact match
	if _, ok := c.beans[id]; ok {
		return id, true
	}

	// If not found and we have a configured prefix that isn't already in the query,
	// try with the prefix prepended (allows short IDs like "abc" to match "beans-abc")
	if c.config != nil && c.config.Beans.Prefix != "" && !strings.HasPrefix(id, c.config.Beans.Prefix) {
		fullID := c.config.Beans.Prefix + id
		if _, ok := c.beans[fullID]; ok {
			return fullID, true
		}
	}

	return id, false
}

// UpdateOption configures optional behavior for Create/Update operations.
type UpdateOption func(*updateOptions)

type updateOptions struct {
	persist      bool   // whether to write to disk (default: true)
	worktreePath string // if set, write to this worktree's .beans/ dir instead of main
}

func defaultUpdateOptions() updateOptions {
	return updateOptions{persist: true}
}

// WithPersist controls whether the operation writes to disk.
// When false, the bean is only updated in memory and marked as dirty.
func WithPersist(persist bool) UpdateOption {
	return func(o *updateOptions) {
		o.persist = persist
	}
}

// WithWorktreePath writes the bean to a worktree's .beans/ directory instead of main.
// The bean is kept dirty in runtime (not persisted to main).
func WithWorktreePath(path string) UpdateOption {
	return func(o *updateOptions) {
		o.worktreePath = path
	}
}

// IsDirty returns true if the bean with the given ID has been modified in
// runtime but not yet persisted to disk.
func (c *Core) IsDirty(id string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.dirty[id]
}

// DirtyIDs returns the IDs of all beans that have unsaved changes.
func (c *Core) DirtyIDs() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	ids := make([]string, 0, len(c.dirty))
	for id := range c.dirty {
		ids = append(ids, id)
	}
	return ids
}

// HasDirty returns true if there are any unsaved bean changes.
func (c *Core) HasDirty() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.dirty) > 0
}

// WorktreeForBean returns the worktree path linked to the given bean ID,
// or empty string if the bean is not linked to any worktree.
func (c *Core) WorktreeForBean(id string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.worktreeLinks[id]
}

// BeansForWorktree returns the IDs of all beans linked to the given worktree path.
func (c *Core) BeansForWorktree(worktreePath string) []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var ids []string
	for id, wt := range c.worktreeLinks {
		if wt == worktreePath {
			ids = append(ids, id)
		}
	}
	return ids
}

// SaveDirty persists all dirty beans to disk and clears their dirty flags.
// Returns the number of beans saved.
func (c *Core) SaveDirty() (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	saved := 0
	for id := range c.dirty {
		b, ok := c.beans[id]
		if !ok {
			// Bean was deleted after being dirtied, just clear flag
			delete(c.dirty, id)
			continue
		}

		if err := c.saveToDisk(b); err != nil {
			return saved, fmt.Errorf("saving bean %s: %w", id, err)
		}

		delete(c.dirty, id)
		saved++
	}

	return saved, nil
}

// SaveBean persists a specific dirty bean to disk and clears its dirty flag.
func (c *Core) SaveBean(id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.dirty[id] {
		return nil // Not dirty, nothing to do
	}

	b, ok := c.beans[id]
	if !ok {
		delete(c.dirty, id)
		return ErrNotFound
	}

	if err := c.saveToDisk(b); err != nil {
		return fmt.Errorf("saving bean %s: %w", id, err)
	}

	delete(c.dirty, id)
	return nil
}

// Create adds a new bean, generating an ID if needed.
// By default, persists to disk. Use WithPersist(false) to only update runtime state.
func (c *Core) Create(b *bean.Bean, opts ...UpdateOption) error {
	o := defaultUpdateOptions()
	for _, opt := range opts {
		opt(&o)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Generate ID if not provided
	if b.ID == "" {
		prefix := ""
		length := 4
		if c.config != nil {
			prefix = c.config.Beans.Prefix
			if c.config.Beans.IDLength > 0 {
				length = c.config.Beans.IDLength
			}
		}
		id, err := bean.NewID(prefix, length)
		if err != nil {
			return fmt.Errorf("generating bean ID: %w", err)
		}
		b.ID = id
	}

	// Set timestamps
	now := time.Now().UTC().Truncate(time.Second)
	b.CreatedAt = &now
	b.UpdatedAt = &now

	if o.persist {
		// Write to disk
		if err := c.saveToDisk(b); err != nil {
			return err
		}
	} else {
		// Mark as dirty (runtime-only change)
		c.dirty[b.ID] = true
	}

	// Add to in-memory map
	c.beans[b.ID] = b

	// Update search index if active (best-effort, don't fail create)
	if c.searchIndex != nil {
		if err := c.searchIndex.IndexBean(b); err != nil {
			c.logWarn("failed to index bean %s: %v", b.ID, err)
		}
	}

	return nil
}

// Update modifies an existing bean.
// If ifMatch is provided, validates the current version's etag matches before updating.
// By default, persists to disk. Use WithPersist(false) to only update runtime state.
func (c *Core) Update(b *bean.Bean, ifMatch *string, opts ...UpdateOption) error {
	o := defaultUpdateOptions()
	for _, opt := range opts {
		opt(&o)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Verify bean exists in memory
	storedBean, ok := c.beans[b.ID]
	if !ok {
		return ErrNotFound
	}

	// Validate etag if provided or required
	requireIfMatch := c.config != nil && c.config.Beans.RequireIfMatch

	if requireIfMatch && (ifMatch == nil || *ifMatch == "") {
		return &ETagRequiredError{}
	}

	if ifMatch != nil && *ifMatch != "" {
		// Calculate etag from the on-disk version by reading the stored bean's path
		// This is necessary because the in-memory bean may have already been modified
		// (Go uses pointers, so modifying the bean passed to Update also modifies c.beans[id])
		var currentETag string
		if storedBean.Path != "" && !c.dirty[b.ID] {
			// Read current file from disk to calculate etag
			diskPath := filepath.Join(c.root, storedBean.Path)
			content, err := os.ReadFile(diskPath)
			if err != nil {
				// If file doesn't exist yet, fall back to stored bean's etag
				currentETag = storedBean.ETag()
			} else {
				// Calculate etag from on-disk content
				h := fnv.New64a()
				h.Write(content)
				currentETag = hex.EncodeToString(h.Sum(nil))
			}
		} else {
			// No path yet or bean is dirty (not on disk), use in-memory etag
			currentETag = storedBean.ETag()
		}

		if currentETag != *ifMatch {
			return &ETagMismatchError{
				Provided: *ifMatch,
				Current:  currentETag,
			}
		}
	}

	// Update timestamp
	now := time.Now().UTC().Truncate(time.Second)
	b.UpdatedAt = &now

	// Auto-route to linked worktree if no explicit path given
	wtPath := o.worktreePath
	if wtPath == "" {
		wtPath = c.worktreeLinks[b.ID]
	}

	if wtPath != "" {
		// Write to the worktree's .beans/ dir; keep dirty in main
		if err := c.saveToWorktree(b, wtPath); err != nil {
			return err
		}
		c.dirty[b.ID] = true
	} else if o.persist {
		// Write to disk
		if err := c.saveToDisk(b); err != nil {
			return err
		}
		// Clear dirty flag since we just persisted
		delete(c.dirty, b.ID)
	} else {
		// Mark as dirty (runtime-only change)
		c.dirty[b.ID] = true
	}

	// Update in-memory map
	c.beans[b.ID] = b

	// Update search index if active (best-effort, don't fail update)
	if c.searchIndex != nil {
		if err := c.searchIndex.IndexBean(b); err != nil {
			c.logWarn("failed to update bean %s in search index: %v", b.ID, err)
		}
	}

	return nil
}

// saveToDisk writes a bean to the filesystem.
func (c *Core) saveToDisk(b *bean.Bean) error {
	// Determine the file path
	var path string
	if b.Path != "" {
		path = filepath.Join(c.root, b.Path)
	} else {
		filename := bean.BuildFilename(b.ID, b.Slug)
		path = filepath.Join(c.root, filename)
		b.Path = filename
	}

	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	// Render and write
	content, err := b.Render()
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, content, 0644); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	return nil
}

// saveToWorktree writes a bean to a worktree's .beans/ directory.
func (c *Core) saveToWorktree(b *bean.Bean, worktreePath string) error {
	beansDir := filepath.Join(worktreePath, BeansDir)

	// Ensure the .beans/ directory exists in the worktree
	if err := os.MkdirAll(beansDir, 0755); err != nil {
		return fmt.Errorf("creating worktree beans dir: %w", err)
	}

	filename := bean.BuildFilename(b.ID, b.Slug)
	path := filepath.Join(beansDir, filename)

	content, err := b.Render()
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, content, 0644); err != nil {
		return fmt.Errorf("writing worktree bean: %w", err)
	}

	return nil
}

// Delete removes a bean by exact ID match.
// Supports short IDs (without prefix) if a prefix is configured.
func (c *Core) Delete(id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Find the bean by exact match
	targetID := id
	targetBean, ok := c.beans[id]

	// If not found and we have a configured prefix, try with prefix prepended
	if !ok && c.config != nil && c.config.Beans.Prefix != "" && !strings.HasPrefix(id, c.config.Beans.Prefix) {
		fullID := c.config.Beans.Prefix + id
		if b, found := c.beans[fullID]; found {
			targetID = fullID
			targetBean = b
			ok = true
		}
	}

	if !ok {
		return ErrNotFound
	}

	// Remove from disk
	path := filepath.Join(c.root, targetBean.Path)
	if err := os.Remove(path); err != nil {
		return err
	}

	// Remove from in-memory map
	delete(c.beans, targetID)

	// Update search index if active (best-effort, don't fail delete)
	if c.searchIndex != nil {
		if err := c.searchIndex.DeleteBean(targetID); err != nil {
			c.logWarn("failed to remove bean %s from search index: %v", targetID, err)
		}
	}

	return nil
}

// Archive moves a bean to the archive directory.
// Supports short IDs (without prefix) if a prefix is configured.
func (c *Core) Archive(id string) error {
	c.mu.Lock()

	// Find the bean
	targetBean, targetID, err := c.findBeanLocked(id)
	if err != nil {
		c.mu.Unlock()
		return err
	}

	// Check if already archived
	if c.isArchivedPath(targetBean.Path) {
		c.mu.Unlock()
		return nil // Already archived, nothing to do
	}

	// Ensure archive directory exists
	archivePath := filepath.Join(c.root, ArchiveDir)
	if err := os.MkdirAll(archivePath, 0755); err != nil {
		c.mu.Unlock()
		return fmt.Errorf("creating archive directory: %w", err)
	}

	// Move the file
	oldPath := filepath.Join(c.root, targetBean.Path)
	newRelPath := filepath.Join(ArchiveDir, filepath.Base(targetBean.Path))
	newPath := filepath.Join(c.root, newRelPath)

	if err := os.Rename(oldPath, newPath); err != nil {
		c.mu.Unlock()
		return fmt.Errorf("moving bean to archive: %w", err)
	}

	// Update bean's path in store and notify subscribers
	targetBean.Path = newRelPath
	c.beans[targetID] = targetBean
	c.mu.Unlock()

	c.fanOut([]BeanEvent{{
		Type:   EventUpdated,
		Bean:   targetBean,
		BeanID: targetID,
	}})

	return nil
}

// Unarchive moves a bean from the archive directory back to the main directory.
// Supports short IDs (without prefix) if a prefix is configured.
func (c *Core) Unarchive(id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Find the bean
	targetBean, targetID, err := c.findBeanLocked(id)
	if err != nil {
		return err
	}

	// Check if not archived
	if !c.isArchivedPath(targetBean.Path) {
		return nil // Not archived, nothing to do
	}

	// Move the file back to main directory
	oldPath := filepath.Join(c.root, targetBean.Path)
	newRelPath := filepath.Base(targetBean.Path)
	newPath := filepath.Join(c.root, newRelPath)

	if err := os.Rename(oldPath, newPath); err != nil {
		return fmt.Errorf("moving bean from archive: %w", err)
	}

	// Update bean's path
	targetBean.Path = newRelPath
	c.beans[targetID] = targetBean

	return nil
}

// IsArchived returns true if the bean with the given ID is in the archive.
// Supports short IDs (without prefix) if a prefix is configured.
func (c *Core) IsArchived(id string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	b, _, err := c.findBeanLocked(id)
	if err != nil {
		return false
	}

	return c.isArchivedPath(b.Path)
}

// isArchivedPath returns true if the path indicates an archived bean.
func (c *Core) isArchivedPath(path string) bool {
	return strings.HasPrefix(path, ArchiveDir+string(filepath.Separator)) ||
		strings.HasPrefix(path, ArchiveDir+"/")
}

// normalizeID returns the full ID with prefix if a prefix is configured
// and the ID doesn't already have it.
func (c *Core) normalizeID(id string) string {
	if c.config != nil && c.config.Beans.Prefix != "" && !strings.HasPrefix(id, c.config.Beans.Prefix) {
		return c.config.Beans.Prefix + id
	}
	return id
}

// findBeanLocked finds a bean by ID, supporting short IDs.
// Must be called with lock held.
func (c *Core) findBeanLocked(id string) (*bean.Bean, string, error) {
	// Try exact match
	if b, ok := c.beans[id]; ok {
		return b, id, nil
	}

	// Try with prefix prepended
	fullID := c.normalizeID(id)
	if fullID != id {
		if b, ok := c.beans[fullID]; ok {
			return b, fullID, nil
		}
	}

	return nil, "", ErrNotFound
}

// GetFromArchive loads a bean directly from the archive directory.
// This is used when a bean isn't in the main loaded set but might be archived.
// Returns nil, nil if the archive directory doesn't exist or bean not found.
func (c *Core) GetFromArchive(id string) (*bean.Bean, error) {
	fullID := c.normalizeID(id)

	archiveDir := filepath.Join(c.root, ArchiveDir)
	if _, err := os.Stat(archiveDir); os.IsNotExist(err) {
		return nil, nil
	}

	// Look for the bean file in the archive
	entries, err := os.ReadDir(archiveDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		fileID, _ := bean.ParseFilename(entry.Name())
		if fileID == fullID {
			path := filepath.Join(archiveDir, entry.Name())
			return c.loadBean(path)
		}
	}

	return nil, nil
}

// LoadAndUnarchive finds a bean in the archive, loads it, unarchives it,
// and adds it to the in-memory store. Returns the bean or ErrNotFound.
func (c *Core) LoadAndUnarchive(id string) (*bean.Bean, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Find the bean (always loaded since we now include archived beans)
	b, targetID, err := c.findBeanLocked(id)
	if err != nil {
		return nil, ErrNotFound
	}

	// If already in main directory, just return it
	if !c.isArchivedPath(b.Path) {
		return b, nil
	}

	// Move file from archive to main directory
	oldPath := filepath.Join(c.root, b.Path)
	newRelPath := filepath.Base(b.Path)
	newPath := filepath.Join(c.root, newRelPath)

	if err := os.Rename(oldPath, newPath); err != nil {
		return nil, fmt.Errorf("moving bean from archive: %w", err)
	}

	// Update bean's path
	b.Path = newRelPath
	c.beans[targetID] = b

	return b, nil
}

// Init creates the .beans directory if it doesn't exist,
// and writes a .gitignore to exclude generated content.
func (c *Core) Init() error {
	if err := os.MkdirAll(c.root, 0755); err != nil {
		return err
	}
	return writeGitignore(c.root)
}

// FullPath returns the absolute path to a bean file.
func (c *Core) FullPath(b *bean.Bean) string {
	return filepath.Join(c.root, b.Path)
}

// Close stops any active file watcher and cleans up resources.
func (c *Core) Close() error {
	// Stop worktree watchers first (outside main lock to avoid deadlock)
	c.UnwatchAllWorktrees()

	c.mu.Lock()
	defer c.mu.Unlock()

	// Close search index if open
	if c.searchIndex != nil {
		if err := c.searchIndex.Close(); err != nil {
			return err
		}
		c.searchIndex = nil
	}

	return c.unwatchLocked()
}

// Init creates the .beans directory at the given path if it doesn't exist,
// and writes a .gitignore to exclude generated content.
// This is a standalone function for use before a Core is created.
func Init(dir string) error {
	beansPath := filepath.Join(dir, BeansDir)
	if err := os.MkdirAll(beansPath, 0755); err != nil {
		return err
	}
	return writeGitignore(beansPath)
}

// writeGitignore creates or overwrites a .gitignore in the beans directory
// to exclude conversation logs from version control.
// Note: worktrees are stored outside the repo (in ~/.beans/worktrees/<project>/).
func writeGitignore(beansDir string) error {
	content := "# Generated by beans init\n.conversations/\n"
	return os.WriteFile(filepath.Join(beansDir, ".gitignore"), []byte(content), 0644)
}

// migrateLegacyDirs renames old-style directories (worktrees/, conversations/)
// to their dot-prefixed equivalents (.worktrees/, .conversations/). Best-effort.
func (c *Core) migrateLegacyDirs() {
	for _, name := range []string{"worktrees", "conversations"} {
		oldPath := filepath.Join(c.root, name)
		newPath := filepath.Join(c.root, "."+name)
		if _, err := os.Stat(oldPath); err == nil {
			if _, err := os.Stat(newPath); os.IsNotExist(err) {
				_ = os.Rename(oldPath, newPath)
			}
		}
	}
}
