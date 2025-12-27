package beancore

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/hmans/beans/internal/bean"
)

const debounceDelay = 100 * time.Millisecond

// EventType represents the type of change that occurred to a bean.
type EventType int

const (
	// EventCreated indicates a new bean was created.
	EventCreated EventType = iota
	// EventUpdated indicates an existing bean was modified.
	EventUpdated
	// EventDeleted indicates a bean was deleted.
	EventDeleted
)

// String returns a human-readable representation of the event type.
func (e EventType) String() string {
	switch e {
	case EventCreated:
		return "created"
	case EventUpdated:
		return "updated"
	case EventDeleted:
		return "deleted"
	default:
		return "unknown"
	}
}

// BeanEvent represents a change to a bean.
type BeanEvent struct {
	Type   EventType  // The type of change
	Bean   *bean.Bean // The bean (nil for Deleted events)
	BeanID string     // Always set, useful for Deleted when Bean is nil
}

// subscription represents a subscriber to bean events.
type subscription struct {
	ch chan []BeanEvent
	id uint64
}

// Subscribe creates a new subscription to bean change events.
// Returns the event channel and an unsubscribe function.
// The channel receives batches of events after debouncing.
// Callers should use defer to call the unsubscribe function.
func (c *Core) Subscribe() (<-chan []BeanEvent, func()) {
	c.subMu.Lock()
	defer c.subMu.Unlock()

	id := atomic.AddUint64(&c.nextSubID, 1)
	ch := make(chan []BeanEvent, 16)

	sub := &subscription{ch: ch, id: id}
	c.subscribers[id] = sub

	unsubscribe := func() {
		c.subMu.Lock()
		defer c.subMu.Unlock()
		if _, ok := c.subscribers[id]; ok {
			close(ch)
			delete(c.subscribers, id)
		}
	}

	return ch, unsubscribe
}

// fanOut sends events to all subscribers (non-blocking).
// Slow subscribers will have events dropped rather than blocking others.
func (c *Core) fanOut(events []BeanEvent) {
	if len(events) == 0 {
		return
	}

	c.subMu.RLock()
	defer c.subMu.RUnlock()

	for _, sub := range c.subscribers {
		select {
		case sub.ch <- events:
			// Sent successfully
		default:
			// Subscriber is slow, drop events
		}
	}
}

// StartWatching begins filesystem monitoring.
// Use Subscribe() to receive bean change events via a channel.
// This is the preferred API for new code; Watch() is kept for backward compatibility.
func (c *Core) StartWatching() error {
	return c.Watch(nil)
}

// Watch starts watching the .beans directory for changes.
// The onChange callback is invoked (after debouncing) whenever beans are created, modified, or deleted.
// The internal state is automatically reloaded before the callback is invoked.
// Deprecated: Use StartWatching() + Subscribe() for new code.
func (c *Core) Watch(onChange func()) error {
	c.mu.Lock()
	if c.watching {
		c.mu.Unlock()
		return nil // Already watching
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		c.mu.Unlock()
		return err
	}

	if err := watcher.Add(c.root); err != nil {
		watcher.Close()
		c.mu.Unlock()
		return err
	}

	// Also watch the archive directory if withArchived is set
	if c.withArchived {
		archivePath := filepath.Join(c.root, ArchiveDir)
		if info, statErr := os.Stat(archivePath); statErr == nil && info.IsDir() {
			// Best effort - don't fail if archive dir can't be watched
			_ = watcher.Add(archivePath)
		}
	}

	c.watching = true
	c.done = make(chan struct{})
	c.onChange = onChange
	c.mu.Unlock()

	// Start the watcher goroutine
	go c.watchLoop(watcher)

	return nil
}

// Unwatch stops watching the .beans directory.
func (c *Core) Unwatch() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.unwatchLocked()
}

// unwatchLocked stops watching (must be called with lock held).
func (c *Core) unwatchLocked() error {
	if !c.watching {
		return nil
	}

	close(c.done)
	c.watching = false
	c.onChange = nil

	// Close all subscriber channels
	c.subMu.Lock()
	for id, sub := range c.subscribers {
		close(sub.ch)
		delete(c.subscribers, id)
	}
	c.subMu.Unlock()

	return nil
}

// watchLoop processes filesystem events with debouncing.
func (c *Core) watchLoop(watcher *fsnotify.Watcher) {
	defer watcher.Close()

	var debounceTimer *time.Timer
	var pendingMu sync.Mutex
	pendingChanges := make(map[string]fsnotify.Op)

	for {
		select {
		case <-c.done:
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

			// Only care about files directly in .beans or in .beans/archive (if withArchived)
			dir := filepath.Dir(event.Name)
			archivePath := filepath.Join(c.root, ArchiveDir)
			validDir := dir == c.root || (c.withArchived && dir == archivePath)
			if !validDir {
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

			// Accumulate changes during debounce window
			pendingMu.Lock()
			pendingChanges[event.Name] |= event.Op
			pendingMu.Unlock()

			// Start/reset debounce timer
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			debounceTimer = time.AfterFunc(debounceDelay, func() {
				// Swap out pending changes atomically
				pendingMu.Lock()
				changes := pendingChanges
				pendingChanges = make(map[string]fsnotify.Op)
				pendingMu.Unlock()

				c.handleChanges(changes)
			})

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			// Log errors but continue watching
			_ = err // In production, you might want to log this
		}
	}
}

// handleChanges processes only the files that changed, updating state incrementally.
func (c *Core) handleChanges(changes map[string]fsnotify.Op) {
	if len(changes) == 0 {
		return
	}

	c.mu.Lock()

	// Check if we're still watching
	if !c.watching {
		c.mu.Unlock()
		return
	}

	var events []BeanEvent

	for path, op := range changes {
		filename := filepath.Base(path)
		id, _ := bean.ParseFilename(filename)

		// Handle removes/renames (file is gone)
		if op&fsnotify.Remove != 0 || op&fsnotify.Rename != 0 {
			// Check if the file actually exists (rename might be followed by create)
			if _, exists := c.beans[id]; exists {
				// Only delete if it was in our map and file is actually gone
				if !c.fileExists(path) {
					delete(c.beans, id)

					// Update search index
					if c.searchIndex != nil {
						if err := c.searchIndex.DeleteBean(id); err != nil {
							c.logWarn("failed to remove bean %s from search index: %v", id, err)
						}
					}

					events = append(events, BeanEvent{
						Type:   EventDeleted,
						Bean:   nil,
						BeanID: id,
					})
				}
			}
			continue
		}

		// Handle creates/writes (file exists or was created)
		if op&fsnotify.Create != 0 || op&fsnotify.Write != 0 {
			newBean, err := c.loadBean(path)
			if err != nil {
				c.logWarn("failed to load bean from %s: %v", path, err)
				continue
			}

			_, existed := c.beans[newBean.ID]
			c.beans[newBean.ID] = newBean

			// Update search index
			if c.searchIndex != nil {
				if err := c.searchIndex.IndexBean(newBean); err != nil {
					c.logWarn("failed to index bean %s: %v", newBean.ID, err)
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
	}

	callback := c.onChange
	c.mu.Unlock()

	// Fan out to subscribers (outside lock)
	c.fanOut(events)

	// Invoke legacy callback
	if callback != nil {
		callback()
	}
}

// fileExists checks if a file exists at the given path.
func (c *Core) fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
