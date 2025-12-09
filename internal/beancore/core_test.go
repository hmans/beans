package beancore

import (
	"os"
	"path/filepath"
	"strings"
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

	// Check .gitignore was created
	gitignorePath := filepath.Join(beansDir, ".gitignore")
	content, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf(".gitignore not created: %v", err)
	}
	if !strings.Contains(string(content), ".index/") {
		t.Error(".gitignore should contain .index/")
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
	createTestBean(t, core, "ghi3", "Third", "todo")

	t.Run("exact match", func(t *testing.T) {
		b, err := core.Get("abc1")
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if b.ID != "abc1" {
			t.Errorf("ID = %q, want %q", b.ID, "abc1")
		}
	})

	t.Run("prefix match", func(t *testing.T) {
		b, err := core.Get("de")
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if b.ID != "def2" {
			t.Errorf("ID = %q, want %q", b.ID, "def2")
		}
	})

	t.Run("single char prefix", func(t *testing.T) {
		b, err := core.Get("g")
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if b.ID != "ghi3" {
			t.Errorf("ID = %q, want %q", b.ID, "ghi3")
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

func TestGetAmbiguous(t *testing.T) {
	core, _ := setupTestCore(t)

	createTestBean(t, core, "abc1", "First", "todo")
	createTestBean(t, core, "abc2", "Second", "todo")

	_, err := core.Get("abc")
	if err != ErrAmbiguousID {
		t.Errorf("Get() error = %v, want ErrAmbiguousID", err)
	}
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

func TestDeleteByPrefix(t *testing.T) {
	core, _ := setupTestCore(t)

	createTestBean(t, core, "unique123", "Test", "todo")

	// Delete by prefix
	err := core.Delete("unique")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify it's gone
	_, err = core.Get("unique123")
	if err != ErrNotFound {
		t.Error("bean should be deleted")
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

func TestLinksPreserved(t *testing.T) {
	core, _ := setupTestCore(t)

	// Create bean A that blocks bean B
	beanA := &bean.Bean{
		ID:     "aaa1",
		Slug:   "blocker",
		Title:  "Blocker Bean",
		Status: "todo",
		Links: bean.Links{
			{Type: "blocks", Target: "bbb2"},
		},
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

	// Bean A should have direct link
	if !loadedA.Links.HasLink("blocks", "bbb2") {
		t.Errorf("Bean A Links = %v, want blocks:bbb2", loadedA.Links)
	}

	// Bean B should have no links
	if len(loadedB.Links) != 0 {
		t.Errorf("Bean B Links = %v, want empty", loadedB.Links)
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
