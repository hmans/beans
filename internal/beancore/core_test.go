package beancore

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/config"
)

func setupTestCore(t *testing.T) (*Core, string) {
	t.Helper()
	tmpDir := t.TempDir()
	beansDir := filepath.Join(tmpDir, BeansDir)
	if err := os.MkdirAll(beansDir, 0755); err != nil {
		t.Fatalf("failed to create test .beans dir: %v", err)
	}

	cfg := config.Default()
	core := New(beansDir, cfg)
	core.SetWarnWriter(nil) // suppress warnings in tests
	if err := core.Load(); err != nil {
		t.Fatalf("failed to load core: %v", err)
	}

	return core, beansDir
}

func createTestBean(t *testing.T, core *Core, id, title, status string) *bean.Bean {
	t.Helper()
	b := &bean.Bean{
		ID:     id,
		Slug:   bean.Slugify(title),
		Title:  title,
		Status: status,
	}
	if err := core.Create(b); err != nil {
		t.Fatalf("failed to create test bean: %v", err)
	}
	return b
}

func TestNew(t *testing.T) {
	cfg := config.Default()
	core := New("/some/path", cfg)

	if core.Root() != "/some/path" {
		t.Errorf("Root() = %q, want %q", core.Root(), "/some/path")
	}
	if core.Config() != cfg {
		t.Error("Config() returned different config")
	}
}

func TestInit(t *testing.T) {
	tmpDir := t.TempDir()
	beansDir := filepath.Join(tmpDir, BeansDir)

	core := New(beansDir, nil)
	err := core.Init()
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	info, err := os.Stat(beansDir)
	if err != nil {
		t.Fatalf(".beans directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Error(".beans is not a directory")
	}
}

func TestInitIdempotent(t *testing.T) {
	tmpDir := t.TempDir()
	beansDir := filepath.Join(tmpDir, BeansDir)

	core := New(beansDir, nil)

	// Call Init twice - should not error
	if err := core.Init(); err != nil {
		t.Fatalf("first Init() error = %v", err)
	}
	if err := core.Init(); err != nil {
		t.Fatalf("second Init() error = %v", err)
	}
}

func TestCreate(t *testing.T) {
	core, beansDir := setupTestCore(t)

	b := &bean.Bean{
		ID:     "abc1",
		Slug:   "test-bean",
		Title:  "Test Bean",
		Status: "todo",
		Body:   "Some content here.",
	}

	err := core.Create(b)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Check file exists
	expectedPath := filepath.Join(beansDir, "abc1--test-bean.md")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("bean file not created at %s", expectedPath)
	}

	// Check timestamps were set
	if b.CreatedAt == nil {
		t.Error("CreatedAt not set")
	}
	if b.UpdatedAt == nil {
		t.Error("UpdatedAt not set")
	}

	// Check Path was set
	if b.Path != "abc1--test-bean.md" {
		t.Errorf("Path = %q, want %q", b.Path, "abc1--test-bean.md")
	}

	// Check in-memory state
	all := core.All()
	if len(all) != 1 {
		t.Errorf("All() returned %d beans, want 1", len(all))
	}
}

func TestCreateGeneratesID(t *testing.T) {
	core, _ := setupTestCore(t)

	b := &bean.Bean{
		Title:  "Auto ID Bean",
		Status: "todo",
	}

	err := core.Create(b)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if b.ID == "" {
		t.Error("ID was not generated")
	}
	if len(b.ID) != 4 { // Default ID length
		t.Errorf("ID length = %d, want 4", len(b.ID))
	}
}

func TestAll(t *testing.T) {
	core, _ := setupTestCore(t)

	createTestBean(t, core, "aaa1", "First Bean", "todo")
	createTestBean(t, core, "bbb2", "Second Bean", "in-progress")
	createTestBean(t, core, "ccc3", "Third Bean", "completed")

	beans := core.All()
	if len(beans) != 3 {
		t.Errorf("All() returned %d beans, want 3", len(beans))
	}
}

func TestAllEmpty(t *testing.T) {
	core, _ := setupTestCore(t)

	beans := core.All()
	if len(beans) != 0 {
		t.Errorf("All() returned %d beans, want 0", len(beans))
	}
}

func TestGet(t *testing.T) {
	core, _ := setupTestCore(t)

	createTestBean(t, core, "abc1", "First", "todo")
	createTestBean(t, core, "def2", "Second", "todo")

	t.Run("exact match", func(t *testing.T) {
		b, err := core.Get("abc1")
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if b.ID != "abc1" {
			t.Errorf("ID = %q, want %q", b.ID, "abc1")
		}
	})

	t.Run("partial ID not found", func(t *testing.T) {
		_, err := core.Get("abc")
		if err != ErrNotFound {
			t.Errorf("Get() error = %v, want ErrNotFound", err)
		}
	})
}

func TestGetNotFound(t *testing.T) {
	core, _ := setupTestCore(t)

	createTestBean(t, core, "abc1", "Test", "todo")

	_, err := core.Get("xyz")
	if err != ErrNotFound {
		t.Errorf("Get() error = %v, want ErrNotFound", err)
	}
}


func TestGetShortID(t *testing.T) {
	// Create a core with a configured prefix
	tmpDir := t.TempDir()
	beansDir := filepath.Join(tmpDir, BeansDir)
	if err := os.MkdirAll(beansDir, 0755); err != nil {
		t.Fatalf("failed to create test .beans dir: %v", err)
	}

	cfg := config.DefaultWithPrefix("beans-")
	core := New(beansDir, cfg)
	core.SetWarnWriter(nil)
	if err := core.Load(); err != nil {
		t.Fatalf("failed to load core: %v", err)
	}

	// Create beans with the prefix
	createTestBean(t, core, "beans-abc1", "First", "todo")
	createTestBean(t, core, "beans-def2", "Second", "todo")

	t.Run("short ID exact match", func(t *testing.T) {
		b, err := core.Get("abc1")
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if b.ID != "beans-abc1" {
			t.Errorf("ID = %q, want %q", b.ID, "beans-abc1")
		}
	})

	t.Run("full ID exact match", func(t *testing.T) {
		b, err := core.Get("beans-abc1")
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if b.ID != "beans-abc1" {
			t.Errorf("ID = %q, want %q", b.ID, "beans-abc1")
		}
	})

	t.Run("partial short ID not found", func(t *testing.T) {
		_, err := core.Get("abc")
		if err != ErrNotFound {
			t.Errorf("Get() error = %v, want ErrNotFound", err)
		}
	})

	t.Run("partial full ID not found", func(t *testing.T) {
		_, err := core.Get("beans-ab")
		if err != ErrNotFound {
			t.Errorf("Get() error = %v, want ErrNotFound", err)
		}
	})

	t.Run("nonexistent ID not found", func(t *testing.T) {
		_, err := core.Get("xyz")
		if err != ErrNotFound {
			t.Errorf("Get() error = %v, want ErrNotFound", err)
		}
	})
}

func TestUpdate(t *testing.T) {
	core, _ := setupTestCore(t)

	b := createTestBean(t, core, "upd1", "Original Title", "todo")
	originalCreatedAt := *b.CreatedAt

	// Update the bean
	b.Title = "Updated Title"
	b.Status = "in-progress"

	err := core.Update(b)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// CreatedAt should be preserved
	if !b.CreatedAt.Equal(originalCreatedAt) {
		t.Errorf("CreatedAt changed: got %v, want %v", b.CreatedAt, originalCreatedAt)
	}

	// UpdatedAt should be refreshed (might be same second, so just check it's set)
	if b.UpdatedAt == nil {
		t.Error("UpdatedAt not set")
	}

	// Verify in-memory state
	loaded, err := core.Get("upd1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if loaded.Title != "Updated Title" {
		t.Errorf("Title = %q, want %q", loaded.Title, "Updated Title")
	}
	if loaded.Status != "in-progress" {
		t.Errorf("Status = %q, want %q", loaded.Status, "in-progress")
	}
}

func TestUpdateNotFound(t *testing.T) {
	core, _ := setupTestCore(t)

	b := &bean.Bean{
		ID:     "nonexistent",
		Title:  "Ghost Bean",
		Status: "todo",
	}

	err := core.Update(b)
	if err != ErrNotFound {
		t.Errorf("Update() error = %v, want ErrNotFound", err)
	}
}

func TestDelete(t *testing.T) {
	core, beansDir := setupTestCore(t)

	b := createTestBean(t, core, "del1", "To Delete", "todo")
	filePath := filepath.Join(beansDir, b.Path)

	// Verify file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatal("bean file should exist before delete")
	}

	// Delete
	err := core.Delete("del1")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify file is gone
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Error("bean file should not exist after delete")
	}

	// Verify in-memory state
	_, err = core.Get("del1")
	if err != ErrNotFound {
		t.Error("bean should not be in memory after delete")
	}
}

func TestDeleteNotFound(t *testing.T) {
	core, _ := setupTestCore(t)

	err := core.Delete("nonexistent")
	if err != ErrNotFound {
		t.Errorf("Delete() error = %v, want ErrNotFound", err)
	}
}

func TestDeleteShortID(t *testing.T) {
	// Create a core with a configured prefix
	tmpDir := t.TempDir()
	beansDir := filepath.Join(tmpDir, BeansDir)
	if err := os.MkdirAll(beansDir, 0755); err != nil {
		t.Fatalf("failed to create test .beans dir: %v", err)
	}

	cfg := config.DefaultWithPrefix("beans-")
	core := New(beansDir, cfg)
	core.SetWarnWriter(nil)
	if err := core.Load(); err != nil {
		t.Fatalf("failed to load core: %v", err)
	}

	createTestBean(t, core, "beans-xyz1", "Test", "todo")

	// Delete by short ID (without prefix)
	err := core.Delete("xyz1")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify it's gone
	_, err = core.Get("beans-xyz1")
	if err != ErrNotFound {
		t.Error("bean should be deleted")
	}
}

func TestDeletePartialIDNotFound(t *testing.T) {
	core, _ := setupTestCore(t)

	createTestBean(t, core, "unique123", "Test", "todo")

	// Partial ID should not match
	err := core.Delete("unique")
	if err != ErrNotFound {
		t.Errorf("Delete() error = %v, want ErrNotFound", err)
	}

	// Verify bean still exists
	_, err = core.Get("unique123")
	if err != nil {
		t.Errorf("bean should still exist, got error: %v", err)
	}
}

func TestFullPath(t *testing.T) {
	core := New("/path/to/.beans", nil)

	b := &bean.Bean{
		ID:   "abc1",
		Path: "abc1--test.md",
	}

	got := core.FullPath(b)
	want := "/path/to/.beans/abc1--test.md"

	if got != want {
		t.Errorf("FullPath() = %q, want %q", got, want)
	}
}

func TestLoad(t *testing.T) {
	core, beansDir := setupTestCore(t)

	// Create a bean file manually
	content := `---
title: Manual Bean
status: open
---

Manual content.
`
	if err := os.WriteFile(filepath.Join(beansDir, "man1--manual.md"), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Reload
	if err := core.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	b, err := core.Get("man1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if b.Title != "Manual Bean" {
		t.Errorf("Title = %q, want %q", b.Title, "Manual Bean")
	}
}

func TestLoadIgnoresNonMdFiles(t *testing.T) {
	core, beansDir := setupTestCore(t)

	createTestBean(t, core, "abc1", "Real Bean", "todo")

	// Create non-.md files that should be ignored
	os.WriteFile(filepath.Join(beansDir, "config.yaml"), []byte("config"), 0644)
	os.WriteFile(filepath.Join(beansDir, "README.txt"), []byte("readme"), 0644)
	os.Mkdir(filepath.Join(beansDir, "subdir"), 0755)

	// Reload
	if err := core.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	beans := core.All()
	if len(beans) != 1 {
		t.Errorf("All() returned %d beans, want 1 (should ignore non-.md files)", len(beans))
	}
}

func TestBlocksPreserved(t *testing.T) {
	core, _ := setupTestCore(t)

	// Create bean A that blocks bean B
	beanA := &bean.Bean{
		ID:     "aaa1",
		Slug:   "blocker",
		Title:  "Blocker Bean",
		Status: "todo",
		Blocking: []string{"bbb2"},
	}
	if err := core.Create(beanA); err != nil {
		t.Fatalf("Create beanA error = %v", err)
	}

	// Create bean B
	beanB := &bean.Bean{
		ID:     "bbb2",
		Slug:   "blocked",
		Title:  "Blocked Bean",
		Status: "todo",
	}
	if err := core.Create(beanB); err != nil {
		t.Fatalf("Create beanB error = %v", err)
	}

	// Reload from disk
	if err := core.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Find the beans
	loadedA, err := core.Get("aaa1")
	if err != nil {
		t.Fatalf("Get aaa1 error = %v", err)
	}
	loadedB, err := core.Get("bbb2")
	if err != nil {
		t.Fatalf("Get bbb2 error = %v", err)
	}

	// Bean A should have direct blocks link
	if !loadedA.IsBlocking("bbb2") {
		t.Errorf("Bean A Blocks = %v, want [bbb2]", loadedA.Blocking)
	}

	// Bean B should have no blocks
	if len(loadedB.Blocking) != 0 {
		t.Errorf("Bean B Blocks = %v, want empty", loadedB.Blocking)
	}
}

func TestConcurrentAccess(t *testing.T) {
	core, _ := setupTestCore(t)

	// Create some initial beans
	for i := 0; i < 10; i++ {
		createTestBean(t, core, bean.NewID("", 4), "Initial Bean", "todo")
	}

	// Run concurrent operations
	var wg sync.WaitGroup
	errors := make(chan error, 100)

	// Readers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = core.All()
			}
		}()
	}

	// Writers (create)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				b := &bean.Bean{
					Title:  "Concurrent Bean",
					Status: "todo",
				}
				if err := core.Create(b); err != nil {
					errors <- err
				}
			}
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("concurrent operation error: %v", err)
	}
}

func TestWatch(t *testing.T) {
	core, beansDir := setupTestCore(t)

	createTestBean(t, core, "wat1", "Initial Bean", "todo")

	// Start watching
	changeCount := 0
	var mu sync.Mutex

	err := core.Watch(func() {
		mu.Lock()
		changeCount++
		mu.Unlock()
	})
	if err != nil {
		t.Fatalf("Watch() error = %v", err)
	}

	// Give watcher time to start
	time.Sleep(50 * time.Millisecond)

	// Create a new bean file manually (simulating external change)
	content := `---
title: External Bean
status: open
---
`
	if err := os.WriteFile(filepath.Join(beansDir, "ext1--external.md"), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Wait for debounce + processing
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	count := changeCount
	mu.Unlock()

	if count == 0 {
		t.Error("onChange callback was not invoked")
	}

	// Verify the new bean is in memory
	_, err = core.Get("ext1")
	if err != nil {
		t.Errorf("external bean not loaded: %v", err)
	}

	// Stop watching
	if err := core.Unwatch(); err != nil {
		t.Fatalf("Unwatch() error = %v", err)
	}
}

func TestWatchDeletedBean(t *testing.T) {
	core, beansDir := setupTestCore(t)

	b := createTestBean(t, core, "del1", "To Delete", "todo")

	// Start watching
	changed := make(chan struct{}, 1)
	err := core.Watch(func() {
		select {
		case changed <- struct{}{}:
		default:
		}
	})
	if err != nil {
		t.Fatalf("Watch() error = %v", err)
	}

	// Give watcher time to start
	time.Sleep(50 * time.Millisecond)

	// Delete the file manually
	if err := os.Remove(filepath.Join(beansDir, b.Path)); err != nil {
		t.Fatalf("failed to delete file: %v", err)
	}

	// Wait for change notification
	select {
	case <-changed:
		// OK
	case <-time.After(500 * time.Millisecond):
		t.Error("onChange callback was not invoked for delete")
	}

	// Verify the bean is gone from memory
	_, err = core.Get("del1")
	if err != ErrNotFound {
		t.Errorf("deleted bean still in memory: %v", err)
	}

	if err := core.Unwatch(); err != nil {
		t.Fatalf("Unwatch() error = %v", err)
	}
}

func TestUnwatchIdempotent(t *testing.T) {
	core, _ := setupTestCore(t)

	// Unwatch without watching should not error
	if err := core.Unwatch(); err != nil {
		t.Errorf("Unwatch() without Watch() error = %v", err)
	}

	// Start watching
	if err := core.Watch(func() {}); err != nil {
		t.Fatalf("Watch() error = %v", err)
	}

	// Unwatch twice should not error
	if err := core.Unwatch(); err != nil {
		t.Errorf("first Unwatch() error = %v", err)
	}
	if err := core.Unwatch(); err != nil {
		t.Errorf("second Unwatch() error = %v", err)
	}
}

func TestClose(t *testing.T) {
	core, _ := setupTestCore(t)

	// Start watching
	if err := core.Watch(func() {}); err != nil {
		t.Fatalf("Watch() error = %v", err)
	}

	// Close should stop the watcher
	if err := core.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestSubscribe(t *testing.T) {
	core, beansDir := setupTestCore(t)

	// Start watching
	if err := core.StartWatching(); err != nil {
		t.Fatalf("StartWatching() error = %v", err)
	}
	defer core.Unwatch()

	// Subscribe to events
	ch, unsub := core.Subscribe()
	defer unsub()

	// Give watcher time to start
	time.Sleep(50 * time.Millisecond)

	// Create a bean file (should trigger EventCreated)
	content := `---
title: New Bean
status: todo
---
`
	if err := os.WriteFile(filepath.Join(beansDir, "new1--new.md"), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Wait for events
	select {
	case events := <-ch:
		if len(events) == 0 {
			t.Error("expected at least one event")
		}
		found := false
		for _, e := range events {
			if e.Type == EventCreated && e.BeanID == "new1" {
				found = true
				if e.Bean == nil {
					t.Error("EventCreated should include Bean")
				}
			}
		}
		if !found {
			t.Errorf("expected EventCreated for new1, got: %+v", events)
		}
	case <-time.After(500 * time.Millisecond):
		t.Error("timeout waiting for events")
	}
}

func TestSubscribeMultiple(t *testing.T) {
	core, beansDir := setupTestCore(t)

	if err := core.StartWatching(); err != nil {
		t.Fatalf("StartWatching() error = %v", err)
	}
	defer core.Unwatch()

	// Create two subscribers
	ch1, unsub1 := core.Subscribe()
	defer unsub1()
	ch2, unsub2 := core.Subscribe()
	defer unsub2()

	// Give watcher time to start
	time.Sleep(50 * time.Millisecond)

	// Create a bean file
	content := `---
title: Multi Test
status: todo
---
`
	if err := os.WriteFile(filepath.Join(beansDir, "mult--multi.md"), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Both subscribers should receive events
	received1, received2 := false, false
	timeout := time.After(500 * time.Millisecond)

	for !received1 || !received2 {
		select {
		case <-ch1:
			received1 = true
		case <-ch2:
			received2 = true
		case <-timeout:
			t.Fatalf("timeout: received1=%v, received2=%v", received1, received2)
		}
	}
}

func TestUnsubscribe(t *testing.T) {
	core, _ := setupTestCore(t)

	if err := core.StartWatching(); err != nil {
		t.Fatalf("StartWatching() error = %v", err)
	}
	defer core.Unwatch()

	ch, unsub := core.Subscribe()
	unsub()

	// Channel should be closed
	_, ok := <-ch
	if ok {
		t.Error("expected channel to be closed after unsubscribe")
	}
}

func TestEventTypes(t *testing.T) {
	core, beansDir := setupTestCore(t)

	// Create an initial bean
	createTestBean(t, core, "evt1", "Event Test", "todo")

	if err := core.StartWatching(); err != nil {
		t.Fatalf("StartWatching() error = %v", err)
	}
	defer core.Unwatch()

	ch, unsub := core.Subscribe()
	defer unsub()

	// Give watcher time to start
	time.Sleep(50 * time.Millisecond)

	t.Run("update event", func(t *testing.T) {
		// Modify the existing bean file
		content := `---
title: Updated Title
status: in-progress
---
`
		if err := os.WriteFile(filepath.Join(beansDir, "evt1--event-test.md"), []byte(content), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		select {
		case events := <-ch:
			found := false
			for _, e := range events {
				if e.Type == EventUpdated && e.BeanID == "evt1" {
					found = true
					if e.Bean == nil {
						t.Error("EventUpdated should include Bean")
					}
					if e.Bean.Title != "Updated Title" {
						t.Errorf("expected updated title, got %q", e.Bean.Title)
					}
				}
			}
			if !found {
				t.Errorf("expected EventUpdated for evt1, got: %+v", events)
			}
		case <-time.After(500 * time.Millisecond):
			t.Error("timeout waiting for update event")
		}
	})

	t.Run("delete event", func(t *testing.T) {
		// Delete the bean file
		if err := os.Remove(filepath.Join(beansDir, "evt1--event-test.md")); err != nil {
			t.Fatalf("failed to delete file: %v", err)
		}

		select {
		case events := <-ch:
			found := false
			for _, e := range events {
				if e.Type == EventDeleted && e.BeanID == "evt1" {
					found = true
					if e.Bean != nil {
						t.Error("EventDeleted should have nil Bean")
					}
				}
			}
			if !found {
				t.Errorf("expected EventDeleted for evt1, got: %+v", events)
			}
		case <-time.After(500 * time.Millisecond):
			t.Error("timeout waiting for delete event")
		}
	})
}

func TestSubscribersClosedOnUnwatch(t *testing.T) {
	core, _ := setupTestCore(t)

	if err := core.StartWatching(); err != nil {
		t.Fatalf("StartWatching() error = %v", err)
	}

	ch, _ := core.Subscribe() // Note: not calling unsub

	// Unwatch should close subscriber channels
	if err := core.Unwatch(); err != nil {
		t.Fatalf("Unwatch() error = %v", err)
	}

	// Channel should be closed
	_, ok := <-ch
	if ok {
		t.Error("expected channel to be closed after Unwatch")
	}
}

func TestMultipleChangesInDebounceWindow(t *testing.T) {
	core, beansDir := setupTestCore(t)

	// Create an initial bean to update
	createTestBean(t, core, "upd1", "To Update", "todo")

	if err := core.StartWatching(); err != nil {
		t.Fatalf("StartWatching() error = %v", err)
	}
	defer core.Unwatch()

	ch, unsub := core.Subscribe()
	defer unsub()

	time.Sleep(50 * time.Millisecond)

	// Make multiple changes rapidly (within debounce window)
	// 1. Create a new bean
	content1 := `---
title: New Bean
status: todo
---
`
	os.WriteFile(filepath.Join(beansDir, "new1--new.md"), []byte(content1), 0644)

	// 2. Update existing bean
	content2 := `---
title: Updated Bean
status: in-progress
---
`
	os.WriteFile(filepath.Join(beansDir, "upd1--to-update.md"), []byte(content2), 0644)

	// 3. Create another bean then delete it (net effect: nothing)
	os.WriteFile(filepath.Join(beansDir, "tmp1--temp.md"), []byte(content1), 0644)
	os.Remove(filepath.Join(beansDir, "tmp1--temp.md"))

	// Wait for debounced events
	select {
	case events := <-ch:
		// Should have events for new1 (created) and upd1 (updated)
		// tmp1 might or might not appear depending on timing
		foundNew := false
		foundUpd := false
		for _, e := range events {
			if e.BeanID == "new1" && e.Type == EventCreated {
				foundNew = true
			}
			if e.BeanID == "upd1" && e.Type == EventUpdated {
				foundUpd = true
			}
		}
		if !foundNew {
			t.Error("expected EventCreated for new1")
		}
		if !foundUpd {
			t.Error("expected EventUpdated for upd1")
		}
	case <-time.After(500 * time.Millisecond):
		t.Error("timeout waiting for events")
	}

	// Verify state is correct
	_, err := core.Get("new1")
	if err != nil {
		t.Errorf("new1 should exist: %v", err)
	}

	upd, err := core.Get("upd1")
	if err != nil {
		t.Fatalf("upd1 should exist: %v", err)
	}
	if upd.Title != "Updated Bean" {
		t.Errorf("upd1 title = %q, want %q", upd.Title, "Updated Bean")
	}

	// tmp1 should not exist
	_, err = core.Get("tmp1")
	if err != ErrNotFound {
		t.Error("tmp1 should not exist (was created then deleted)")
	}
}

func TestInvalidFileIgnored(t *testing.T) {
	core, beansDir := setupTestCore(t)

	// Create a valid bean first
	createTestBean(t, core, "val1", "Valid Bean", "todo")

	if err := core.StartWatching(); err != nil {
		t.Fatalf("StartWatching() error = %v", err)
	}
	defer core.Unwatch()

	ch, unsub := core.Subscribe()
	defer unsub()

	time.Sleep(50 * time.Millisecond)

	// Create an invalid bean file (malformed YAML frontmatter)
	invalidContent := `---
title: [unclosed bracket
status: {broken yaml
---
`
	os.WriteFile(filepath.Join(beansDir, "bad1--invalid.md"), []byte(invalidContent), 0644)

	// Also create a valid bean to verify processing continues
	validContent := `---
title: Another Valid
status: todo
---
`
	os.WriteFile(filepath.Join(beansDir, "val2--another.md"), []byte(validContent), 0644)

	// Wait for events
	select {
	case events := <-ch:
		// Should have event for val2 (created), bad1 should be skipped
		foundVal2 := false
		for _, e := range events {
			if e.BeanID == "val2" && e.Type == EventCreated {
				foundVal2 = true
			}
			if e.BeanID == "bad1" {
				t.Error("bad1 should not produce an event (invalid file)")
			}
		}
		if !foundVal2 {
			t.Error("expected EventCreated for val2")
		}
	case <-time.After(500 * time.Millisecond):
		t.Error("timeout waiting for events")
	}

	// Valid beans should still be accessible
	if _, err := core.Get("val1"); err != nil {
		t.Errorf("val1 should still exist: %v", err)
	}
	if _, err := core.Get("val2"); err != nil {
		t.Errorf("val2 should exist: %v", err)
	}
}

func TestRapidUpdatesToSameFile(t *testing.T) {
	core, beansDir := setupTestCore(t)

	createTestBean(t, core, "rap1", "Rapid Updates", "todo")

	if err := core.StartWatching(); err != nil {
		t.Fatalf("StartWatching() error = %v", err)
	}
	defer core.Unwatch()

	ch, unsub := core.Subscribe()
	defer unsub()

	time.Sleep(50 * time.Millisecond)

	// Write to the same file multiple times rapidly
	for i := 1; i <= 5; i++ {
		content := fmt.Sprintf(`---
title: Update %d
status: todo
---
`, i)
		os.WriteFile(filepath.Join(beansDir, "rap1--rapid-updates.md"), []byte(content), 0644)
		time.Sleep(10 * time.Millisecond) // Small delay but within debounce
	}

	// Should get a single batch of events (debounced)
	select {
	case events := <-ch:
		// Count events for rap1 - should be exactly one
		rap1Count := 0
		var lastEvent BeanEvent
		for _, e := range events {
			if e.BeanID == "rap1" {
				rap1Count++
				lastEvent = e
			}
		}
		if rap1Count != 1 {
			t.Errorf("expected 1 event for rap1, got %d", rap1Count)
		}
		if lastEvent.Type != EventUpdated {
			t.Errorf("expected EventUpdated, got %v", lastEvent.Type)
		}
		// Should have the final value
		if lastEvent.Bean != nil && lastEvent.Bean.Title != "Update 5" {
			t.Errorf("expected title 'Update 5', got %q", lastEvent.Bean.Title)
		}
	case <-time.After(500 * time.Millisecond):
		t.Error("timeout waiting for events")
	}
}

// Archive functionality tests

func TestArchive(t *testing.T) {
	core, beansDir := setupTestCore(t)

	b := createTestBean(t, core, "arc1", "To Archive", "completed")
	originalFilename := filepath.Base(b.Path)

	// Archive the bean
	err := core.Archive("arc1")
	if err != nil {
		t.Fatalf("Archive() error = %v", err)
	}

	// Verify file moved to archive directory
	archivePath := filepath.Join(beansDir, ArchiveDir, originalFilename)
	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		t.Error("bean file should exist in archive directory")
	}

	// Verify file no longer in main directory
	mainPath := filepath.Join(beansDir, "arc1--to-archive.md")
	if _, err := os.Stat(mainPath); !os.IsNotExist(err) {
		t.Error("bean file should not exist in main directory")
	}

	// Verify bean is still accessible in memory
	archived, err := core.Get("arc1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	// Verify path is updated
	if archived.Path != filepath.Join(ArchiveDir, "arc1--to-archive.md") {
		t.Errorf("Path = %q, want %q", archived.Path, filepath.Join(ArchiveDir, "arc1--to-archive.md"))
	}
}

func TestArchiveIdempotent(t *testing.T) {
	core, _ := setupTestCore(t)

	createTestBean(t, core, "arc1", "To Archive", "completed")

	// Archive twice should not error
	if err := core.Archive("arc1"); err != nil {
		t.Fatalf("first Archive() error = %v", err)
	}
	if err := core.Archive("arc1"); err != nil {
		t.Fatalf("second Archive() error = %v", err)
	}
}

func TestArchiveNotFound(t *testing.T) {
	core, _ := setupTestCore(t)

	err := core.Archive("nonexistent")
	if err != ErrNotFound {
		t.Errorf("Archive() error = %v, want ErrNotFound", err)
	}
}

func TestUnarchive(t *testing.T) {
	core, beansDir := setupTestCore(t)

	b := createTestBean(t, core, "una1", "To Unarchive", "completed")
	originalFilename := filepath.Base(b.Path)

	// Archive first
	if err := core.Archive("una1"); err != nil {
		t.Fatalf("Archive() error = %v", err)
	}

	// Unarchive
	err := core.Unarchive("una1")
	if err != nil {
		t.Fatalf("Unarchive() error = %v", err)
	}

	// Verify file moved back to main directory
	mainPath := filepath.Join(beansDir, originalFilename)
	if _, err := os.Stat(mainPath); os.IsNotExist(err) {
		t.Error("bean file should exist in main directory")
	}

	// Verify file no longer in archive
	archivePath := filepath.Join(beansDir, ArchiveDir, originalFilename)
	if _, err := os.Stat(archivePath); !os.IsNotExist(err) {
		t.Error("bean file should not exist in archive directory")
	}

	// Verify path is updated
	unarchived, err := core.Get("una1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if unarchived.Path != "una1--to-unarchive.md" {
		t.Errorf("Path = %q, want %q", unarchived.Path, "una1--to-unarchive.md")
	}
}

func TestUnarchiveIdempotent(t *testing.T) {
	core, _ := setupTestCore(t)

	createTestBean(t, core, "una1", "To Unarchive", "completed")

	// Unarchive non-archived bean should not error
	if err := core.Unarchive("una1"); err != nil {
		t.Fatalf("Unarchive() on non-archived bean error = %v", err)
	}
}

func TestIsArchived(t *testing.T) {
	core, _ := setupTestCore(t)

	createTestBean(t, core, "isa1", "Test Archived", "completed")

	t.Run("not archived", func(t *testing.T) {
		if core.IsArchived("isa1") {
			t.Error("IsArchived() should return false for non-archived bean")
		}
	})

	// Archive the bean
	if err := core.Archive("isa1"); err != nil {
		t.Fatalf("Archive() error = %v", err)
	}

	t.Run("archived", func(t *testing.T) {
		if !core.IsArchived("isa1") {
			t.Error("IsArchived() should return true for archived bean")
		}
	})

	t.Run("nonexistent", func(t *testing.T) {
		if core.IsArchived("nonexistent") {
			t.Error("IsArchived() should return false for nonexistent bean")
		}
	})
}

func TestArchivedBeansAlwaysLoaded(t *testing.T) {
	core, beansDir := setupTestCore(t)

	// Create beans and archive one
	createTestBean(t, core, "act1", "Active Bean", "todo")
	createTestBean(t, core, "arc1", "Archived Bean", "completed")
	if err := core.Archive("arc1"); err != nil {
		t.Fatalf("Archive() error = %v", err)
	}

	// Create a new core and load - archived beans should always be included
	core2 := New(beansDir, config.Default())
	core2.SetWarnWriter(nil)
	if err := core2.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	t.Run("all beans loaded including archived", func(t *testing.T) {
		beans := core2.All()
		if len(beans) != 2 {
			t.Errorf("All() returned %d beans, want 2 (active + archived)", len(beans))
		}
	})

	t.Run("active bean accessible", func(t *testing.T) {
		if _, err := core2.Get("act1"); err != nil {
			t.Errorf("active bean should be found: %v", err)
		}
	})

	t.Run("archived bean accessible", func(t *testing.T) {
		if _, err := core2.Get("arc1"); err != nil {
			t.Errorf("archived bean should be found: %v", err)
		}
	})

	t.Run("archived bean has correct path", func(t *testing.T) {
		b, _ := core2.Get("arc1")
		if !core2.IsArchived("arc1") {
			t.Error("archived bean should be identified as archived")
		}
		if b.Path != "archive/arc1--archived-bean.md" {
			t.Errorf("archived bean path = %q, want %q", b.Path, "archive/arc1--archived-bean.md")
		}
	})
}

func TestGetFromArchive(t *testing.T) {
	core, beansDir := setupTestCore(t)

	createTestBean(t, core, "gfa1", "Get From Archive", "completed")
	if err := core.Archive("gfa1"); err != nil {
		t.Fatalf("Archive() error = %v", err)
	}

	// Create a new core - archived beans are loaded but GetFromArchive reads directly from disk
	core2 := New(beansDir, config.Default())
	core2.SetWarnWriter(nil)
	if err := core2.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	t.Run("bean in archive", func(t *testing.T) {
		b, err := core2.GetFromArchive("gfa1")
		if err != nil {
			t.Fatalf("GetFromArchive() error = %v", err)
		}
		if b == nil {
			t.Fatal("GetFromArchive() returned nil")
		}
		if b.ID != "gfa1" {
			t.Errorf("ID = %q, want %q", b.ID, "gfa1")
		}
	})

	t.Run("bean not in archive", func(t *testing.T) {
		b, err := core2.GetFromArchive("nonexistent")
		if err != nil {
			t.Fatalf("GetFromArchive() error = %v", err)
		}
		if b != nil {
			t.Error("GetFromArchive() should return nil for nonexistent bean")
		}
	})

	t.Run("no archive directory", func(t *testing.T) {
		// Create a fresh core with no archive
		tmpDir := t.TempDir()
		freshBeansDir := filepath.Join(tmpDir, BeansDir)
		if err := os.MkdirAll(freshBeansDir, 0755); err != nil {
			t.Fatalf("failed to create .beans dir: %v", err)
		}
		core3 := New(freshBeansDir, config.Default())
		core3.SetWarnWriter(nil)
		if err := core3.Load(); err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		b, err := core3.GetFromArchive("anything")
		if err != nil {
			t.Fatalf("GetFromArchive() error = %v", err)
		}
		if b != nil {
			t.Error("GetFromArchive() should return nil when archive doesn't exist")
		}
	})
}

func TestLoadAndUnarchive(t *testing.T) {
	core, beansDir := setupTestCore(t)

	createTestBean(t, core, "lau1", "Load And Unarchive", "completed")
	if err := core.Archive("lau1"); err != nil {
		t.Fatalf("Archive() error = %v", err)
	}

	// Create a new core - archived beans are now always loaded
	core2 := New(beansDir, config.Default())
	core2.SetWarnWriter(nil)
	if err := core2.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Bean should be accessible (archived beans are always loaded)
	b, err := core2.Get("lau1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !core2.IsArchived("lau1") {
		t.Error("bean should be identified as archived before LoadAndUnarchive")
	}

	// Load and unarchive should move the file
	unarchived, err := core2.LoadAndUnarchive("lau1")
	if err != nil {
		t.Fatalf("LoadAndUnarchive() error = %v", err)
	}
	if unarchived == nil {
		t.Fatal("LoadAndUnarchive() returned nil")
	}
	if unarchived.ID != b.ID {
		t.Errorf("LoadAndUnarchive returned different bean: got %q, want %q", unarchived.ID, b.ID)
	}

	// Bean should no longer be archived
	if core2.IsArchived("lau1") {
		t.Error("bean should not be archived after LoadAndUnarchive")
	}

	// File should be in main directory, not archive
	mainPath := filepath.Join(beansDir, "lau1--load-and-unarchive.md")
	if _, err := os.Stat(mainPath); os.IsNotExist(err) {
		t.Error("bean file should exist in main directory after LoadAndUnarchive")
	}

	// File should NOT be in archive directory
	archivePath := filepath.Join(beansDir, "archive", "lau1--load-and-unarchive.md")
	if _, err := os.Stat(archivePath); !os.IsNotExist(err) {
		t.Error("bean file should not exist in archive directory after LoadAndUnarchive")
	}
}

func TestLoadAndUnarchiveNotFound(t *testing.T) {
	core, _ := setupTestCore(t)

	_, err := core.LoadAndUnarchive("nonexistent")
	if err != ErrNotFound {
		t.Errorf("LoadAndUnarchive() error = %v, want ErrNotFound", err)
	}
}

func TestArchiveShortID(t *testing.T) {
	// Create a core with a configured prefix
	tmpDir := t.TempDir()
	beansDir := filepath.Join(tmpDir, BeansDir)
	if err := os.MkdirAll(beansDir, 0755); err != nil {
		t.Fatalf("failed to create test .beans dir: %v", err)
	}

	cfg := config.DefaultWithPrefix("beans-")
	core := New(beansDir, cfg)
	core.SetWarnWriter(nil)
	if err := core.Load(); err != nil {
		t.Fatalf("failed to load core: %v", err)
	}

	createTestBean(t, core, "beans-xyz1", "Test", "completed")

	// Archive by short ID (without prefix)
	err := core.Archive("xyz1")
	if err != nil {
		t.Fatalf("Archive() error = %v", err)
	}

	// Verify it's archived
	if !core.IsArchived("beans-xyz1") {
		t.Error("bean should be archived")
	}
}
