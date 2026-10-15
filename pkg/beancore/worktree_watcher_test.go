package beancore

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWorktreeWatcher(t *testing.T) {
	t.Run("watches worktree beans dir and merges changes", func(t *testing.T) {
		core, _ := setupTestCore(t)

		// Create a bean in the main repo first
		createTestBean(t, core, "wt-test-1", "Original", "todo")

		// Start watching
		if err := core.StartWatching(); err != nil {
			t.Fatalf("StartWatching() error = %v", err)
		}
		defer core.Unwatch()

		// Create a fake worktree directory with a .beans/ subdir
		wtDir := t.TempDir()
		wtBeansDir := filepath.Join(wtDir, BeansDir)
		if err := os.MkdirAll(wtBeansDir, 0755); err != nil {
			t.Fatalf("failed to create worktree .beans dir: %v", err)
		}

		// Subscribe to events
		events, unsub := core.Subscribe()
		defer unsub()

		// Start watching the worktree
		if err := core.WatchWorktreeBeans(wtDir); err != nil {
			t.Fatalf("WatchWorktreeBeans() error = %v", err)
		}
		defer core.UnwatchWorktreeBeans(wtDir)

		// Write a modified version of the bean in the worktree
		content := `---
title: Updated in Worktree
status: in-progress
type: task
---

Working on this in a worktree.
`
		beanPath := filepath.Join(wtBeansDir, "wt-test-1--original.md")
		if err := os.WriteFile(beanPath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write worktree bean: %v", err)
		}

		// Wait for the event to propagate
		select {
		case batch := <-events:
			found := false
			for _, ev := range batch {
				if ev.BeanID == "wt-test-1" && ev.Type == EventUpdated {
					found = true
					if ev.Bean.Title != "Updated in Worktree" {
						t.Errorf("Title = %q, want %q", ev.Bean.Title, "Updated in Worktree")
					}
					if ev.Bean.Status != "in-progress" {
						t.Errorf("Status = %q, want %q", ev.Bean.Status, "in-progress")
					}
				}
			}
			if !found {
				t.Error("expected EventUpdated for wt-test-1")
			}
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for worktree bean change event")
		}

		// Bean should be dirty (came from worktree, not persisted to main)
		if !core.IsDirty("wt-test-1") {
			t.Error("bean should be dirty after worktree update")
		}

		// In-memory state should reflect the worktree's version
		got, err := core.Get("wt-test-1")
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if got.Title != "Updated in Worktree" {
			t.Errorf("Title = %q, want %q", got.Title, "Updated in Worktree")
		}
	})

	t.Run("does not crash when worktree has no .beans dir", func(t *testing.T) {
		core, _ := setupTestCore(t)

		wtDir := t.TempDir()
		// No .beans/ dir inside

		err := core.WatchWorktreeBeans(wtDir)
		if err != nil {
			t.Errorf("WatchWorktreeBeans() should return nil for missing .beans/ dir, got %v", err)
		}
	})

	t.Run("delete in worktree reverts to main-repo version", func(t *testing.T) {
		core, _ := setupTestCore(t)

		// Create a bean in the main repo
		createTestBean(t, core, "wt-del-1", "Original Title", "todo")

		// Create a worktree with a modified version of the bean
		wtDir := t.TempDir()
		wtBeansDir := filepath.Join(wtDir, BeansDir)
		os.MkdirAll(wtBeansDir, 0755)

		content := "---\ntitle: Modified in Worktree\nstatus: in-progress\ntype: task\n---\n"
		beanPath := filepath.Join(wtBeansDir, "wt-del-1--original-title.md")
		os.WriteFile(beanPath, []byte(content), 0644)

		// Start watching the worktree
		if err := core.WatchWorktreeBeans(wtDir); err != nil {
			t.Fatalf("WatchWorktreeBeans() error = %v", err)
		}
		defer core.UnwatchWorktreeBeans(wtDir)

		// Wait for initial load to merge the worktree version
		time.Sleep(100 * time.Millisecond)

		// Verify the worktree version was merged
		got, err := core.Get("wt-del-1")
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if got.Title != "Modified in Worktree" {
			t.Errorf("Title = %q, want %q (worktree version should be active)", got.Title, "Modified in Worktree")
		}

		// Subscribe to events
		events, unsub := core.Subscribe()
		defer unsub()

		// Delete the bean from the worktree
		os.Remove(beanPath)

		// Should emit an Updated event reverting to the main-repo version
		select {
		case batch := <-events:
			found := false
			for _, ev := range batch {
				if ev.BeanID == "wt-del-1" && ev.Type == EventUpdated {
					found = true
					if ev.Bean.Title != "Original Title" {
						t.Errorf("reverted Title = %q, want %q", ev.Bean.Title, "Original Title")
					}
				}
			}
			if !found {
				t.Error("expected EventUpdated reverting to main-repo version")
			}
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for revert event")
		}

		// Bean should still exist and be reverted to main-repo version
		got, err = core.Get("wt-del-1")
		if err != nil {
			t.Fatalf("bean should still exist after worktree delete, got error: %v", err)
		}
		if got.Title != "Original Title" {
			t.Errorf("Title = %q, want %q", got.Title, "Original Title")
		}

		// Should no longer be dirty
		if core.IsDirty("wt-del-1") {
			t.Error("bean should not be dirty after reverting to main-repo version")
		}
	})

	t.Run("delete worktree-only bean removes from runtime", func(t *testing.T) {
		core, _ := setupTestCore(t)

		// Create a worktree with a bean that doesn't exist in main
		wtDir := t.TempDir()
		wtBeansDir := filepath.Join(wtDir, BeansDir)
		os.MkdirAll(wtBeansDir, 0755)

		content := "---\ntitle: Worktree Only\nstatus: todo\ntype: task\n---\n"
		beanPath := filepath.Join(wtBeansDir, "wt-only-1--worktree-only.md")
		os.WriteFile(beanPath, []byte(content), 0644)

		// Start watching — initial scan loads the bean
		if err := core.WatchWorktreeBeans(wtDir); err != nil {
			t.Fatalf("WatchWorktreeBeans() error = %v", err)
		}
		defer core.UnwatchWorktreeBeans(wtDir)

		// Verify the bean was loaded
		got, err := core.Get("wt-only-1")
		if err != nil {
			t.Fatalf("bean should exist after initial load, got error: %v", err)
		}
		if got.Title != "Worktree Only" {
			t.Errorf("Title = %q, want %q", got.Title, "Worktree Only")
		}

		// Subscribe to events
		events, unsub := core.Subscribe()
		defer unsub()

		// Delete the bean from the worktree
		os.Remove(beanPath)

		// Should emit a Deleted event
		select {
		case batch := <-events:
			found := false
			for _, ev := range batch {
				if ev.BeanID == "wt-only-1" && ev.Type == EventDeleted {
					found = true
				}
			}
			if !found {
				t.Error("expected EventDeleted for worktree-only bean")
			}
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for delete event")
		}

		// Bean should no longer exist
		if _, err := core.Get("wt-only-1"); err != ErrNotFound {
			t.Errorf("expected ErrNotFound after delete, got %v", err)
		}
	})

	t.Run("worktree links are set on initial load", func(t *testing.T) {
		core, _ := setupTestCore(t)

		// Create a worktree with a bean
		wtDir := t.TempDir()
		wtBeansDir := filepath.Join(wtDir, BeansDir)
		os.MkdirAll(wtBeansDir, 0755)

		content := "---\ntitle: Linked Bean\nstatus: todo\ntype: task\n---\n"
		beanPath := filepath.Join(wtBeansDir, "wt-link-1--linked-bean.md")
		os.WriteFile(beanPath, []byte(content), 0644)

		// Watch the worktree — initial load should set the link
		if err := core.WatchWorktreeBeans(wtDir); err != nil {
			t.Fatalf("WatchWorktreeBeans() error = %v", err)
		}
		defer core.UnwatchWorktreeBeans(wtDir)

		// Verify the link was set
		if got := core.WorktreeForBean("wt-link-1"); got != wtDir {
			t.Errorf("WorktreeForBean() = %q, want %q", got, wtDir)
		}

		// Verify BeansForWorktree returns the bean
		ids := core.BeansForWorktree(wtDir)
		if len(ids) != 1 || ids[0] != "wt-link-1" {
			t.Errorf("BeansForWorktree() = %v, want [wt-link-1]", ids)
		}
	})

	t.Run("worktree links are cleared on unwatch", func(t *testing.T) {
		core, _ := setupTestCore(t)

		// Create a worktree with a bean
		wtDir := t.TempDir()
		wtBeansDir := filepath.Join(wtDir, BeansDir)
		os.MkdirAll(wtBeansDir, 0755)

		content := "---\ntitle: Unlink Bean\nstatus: todo\ntype: task\n---\n"
		os.WriteFile(filepath.Join(wtBeansDir, "wt-unlink-1--unlink-bean.md"), []byte(content), 0644)

		if err := core.WatchWorktreeBeans(wtDir); err != nil {
			t.Fatalf("WatchWorktreeBeans() error = %v", err)
		}

		// Verify link exists
		if got := core.WorktreeForBean("wt-unlink-1"); got == "" {
			t.Fatal("expected worktree link to be set")
		}

		// Unwatch — link should be cleared
		core.UnwatchWorktreeBeans(wtDir)

		if got := core.WorktreeForBean("wt-unlink-1"); got != "" {
			t.Errorf("WorktreeForBean() after unwatch = %q, want empty", got)
		}
	})

	t.Run("update auto-routes to linked worktree", func(t *testing.T) {
		core, _ := setupTestCore(t)

		// Create a bean in main
		createTestBean(t, core, "wt-auto-1", "Main Version", "todo")

		// Create a worktree with a modified version
		wtDir := t.TempDir()
		wtBeansDir := filepath.Join(wtDir, BeansDir)
		os.MkdirAll(wtBeansDir, 0755)

		content := "---\ntitle: WT Version\nstatus: in-progress\ntype: task\n---\n"
		os.WriteFile(filepath.Join(wtBeansDir, "wt-auto-1--wt-version.md"), []byte(content), 0644)

		if err := core.WatchWorktreeBeans(wtDir); err != nil {
			t.Fatalf("WatchWorktreeBeans() error = %v", err)
		}
		defer core.UnwatchWorktreeBeans(wtDir)

		// Update the bean via Core.Update() — should auto-route to worktree
		b, base := cloneStored(t, core, "wt-auto-1")
		b.Title = "Updated via Core"
		if err := core.Update(b, base); err != nil {
			t.Fatalf("Update() error = %v", err)
		}

		// Bean should still be dirty (written to worktree, not main)
		if !core.IsDirty("wt-auto-1") {
			t.Error("bean should be dirty after auto-routed update")
		}

		// Verify the file was written to the worktree
		entries, err := os.ReadDir(wtBeansDir)
		if err != nil {
			t.Fatalf("ReadDir() error = %v", err)
		}
		found := false
		for _, e := range entries {
			if filepath.Ext(e.Name()) == ".md" {
				found = true
			}
		}
		if !found {
			t.Error("expected updated bean file in worktree .beans/ dir")
		}
	})

	t.Run("rebase with identical beans does not link them", func(t *testing.T) {
		core, _ := setupTestCore(t)

		// Create beans in main with explicit content (stable timestamps)
		bean1Content := "---\ntitle: Bean One\nstatus: todo\ntype: task\ncreated_at: 2025-01-01T00:00:00Z\nupdated_at: 2025-01-01T00:00:00Z\n---\n"
		os.WriteFile(filepath.Join(core.Root(), "rebase-1--bean-one.md"), []byte(bean1Content), 0644)
		bean2Content := "---\ntitle: Bean Two\nstatus: todo\ntype: task\ncreated_at: 2025-01-01T00:00:00Z\nupdated_at: 2025-01-01T00:00:00Z\n---\n"
		os.WriteFile(filepath.Join(core.Root(), "rebase-2--bean-two.md"), []byte(bean2Content), 0644)
		// Reload so core picks them up
		core.Load()

		// Start watching so fsnotify picks up changes
		if err := core.StartWatching(); err != nil {
			t.Fatalf("StartWatching() error = %v", err)
		}
		defer core.Unwatch()

		// Create a worktree with one modified bean and one identical to main
		wtDir := t.TempDir()
		wtBeansDir := filepath.Join(wtDir, BeansDir)
		os.MkdirAll(wtBeansDir, 0755)

		// Modified bean — should be linked
		modifiedContent := "---\ntitle: Bean One Modified\nstatus: in-progress\ntype: task\ncreated_at: 2025-01-01T00:00:00Z\nupdated_at: 2025-01-01T00:00:00Z\n---\n"
		os.WriteFile(filepath.Join(wtBeansDir, "rebase-1--bean-one.md"), []byte(modifiedContent), 0644)

		// Identical content — should NOT be linked (simulates rebase pulling in main's version)
		os.WriteFile(filepath.Join(wtBeansDir, "rebase-2--bean-two.md"), []byte(bean2Content), 0644)

		if err := core.WatchWorktreeBeans(wtDir); err != nil {
			t.Fatalf("WatchWorktreeBeans() error = %v", err)
		}
		defer core.UnwatchWorktreeBeans(wtDir)

		// Modified bean should be linked
		if got := core.WorktreeForBean("rebase-1"); got != wtDir {
			t.Errorf("WorktreeForBean(rebase-1) = %q, want %q", got, wtDir)
		}

		// Identical bean should NOT be linked
		if got := core.WorktreeForBean("rebase-2"); got != "" {
			t.Errorf("WorktreeForBean(rebase-2) = %q, want empty (identical to main)", got)
		}

		// Now simulate a rebase: write the main version of rebase-1 into the worktree
		// (i.e., the modification was reverted by the rebase)
		events, unsub := core.Subscribe()
		defer unsub()

		os.WriteFile(filepath.Join(wtBeansDir, "rebase-1--bean-one.md"), []byte(bean1Content), 0644)

		// Wait for the watcher to process the change
		select {
		case <-events:
			// Event received
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for rebase event")
		}

		// After the "rebase", rebase-1 should no longer be linked
		if got := core.WorktreeForBean("rebase-1"); got != "" {
			t.Errorf("WorktreeForBean(rebase-1) after rebase = %q, want empty", got)
		}

		// And should no longer be dirty
		if core.IsDirty("rebase-1") {
			t.Error("rebase-1 should not be dirty after reverting to main version")
		}
	})

	t.Run("UnwatchAllWorktrees stops all watchers", func(t *testing.T) {
		core, _ := setupTestCore(t)

		// Create two worktrees
		for i := 0; i < 2; i++ {
			wtDir := t.TempDir()
			wtBeansDir := filepath.Join(wtDir, BeansDir)
			os.MkdirAll(wtBeansDir, 0755)
			if err := core.WatchWorktreeBeans(wtDir); err != nil {
				t.Fatalf("WatchWorktreeBeans() error = %v", err)
			}
		}

		core.UnwatchAllWorktrees()

		// Verify no worktree watchers remain
		core.mu.RLock()
		count := len(core.worktreeWatchers)
		core.mu.RUnlock()

		if count != 0 {
			t.Errorf("expected 0 worktree watchers after UnwatchAll, got %d", count)
		}
	})
}
