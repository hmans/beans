package beancore

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

const debounceDelay = 100 * time.Millisecond

// Watch starts watching the .beans directory for changes.
// The onChange callback is invoked (after debouncing) whenever beans are created, modified, or deleted.
// The internal state is automatically reloaded before the callback is invoked.
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

	return nil
}

// watchLoop processes filesystem events with debouncing.
func (c *Core) watchLoop(watcher *fsnotify.Watcher) {
	defer watcher.Close()

	var debounceTimer *time.Timer
	var pendingReload bool

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

			// Only care about files directly in .beans (not subdirectories)
			dir := filepath.Dir(event.Name)
			if dir != c.root {
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

			// Mark that we have a pending change and start/reset debounce timer
			pendingReload = true
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			debounceTimer = time.AfterFunc(debounceDelay, func() {
				c.handleChange()
			})

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			// Log errors but continue watching
			_ = err // In production, you might want to log this
		}

		// Silence the unused variable warning
		_ = pendingReload
	}
}

// handleChange reloads beans from disk and invokes the onChange callback.
func (c *Core) handleChange() {
	c.mu.Lock()

	// Check if we're still watching
	if !c.watching {
		c.mu.Unlock()
		return
	}

	// Reload from disk
	if err := c.loadFromDisk(); err != nil {
		// On error, just continue - the beans map may be stale but that's better than crashing
		c.mu.Unlock()
		return
	}

	callback := c.onChange
	c.mu.Unlock()

	// Invoke callback outside of lock
	if callback != nil {
		callback()
	}
}
