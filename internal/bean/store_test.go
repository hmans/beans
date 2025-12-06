package bean

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestStore(t *testing.T) (*Store, string) {
	t.Helper()
	tmpDir := t.TempDir()
	beansDir := filepath.Join(tmpDir, BeansDir)
	if err := os.MkdirAll(beansDir, 0755); err != nil {
		t.Fatalf("failed to create test .beans dir: %v", err)
	}
	return NewStore(beansDir), beansDir
}

func createTestBean(t *testing.T, store *Store, id, title, status string) *Bean {
	t.Helper()
	bean := &Bean{
		ID:     id,
		Slug:   Slugify(title),
		Title:  title,
		Status: status,
	}
	if err := store.Save(bean); err != nil {
		t.Fatalf("failed to create test bean: %v", err)
	}
	return bean
}

func TestNewStore(t *testing.T) {
	store := NewStore("/some/path")
	if store.Root != "/some/path" {
		t.Errorf("Root = %q, want %q", store.Root, "/some/path")
	}
}

func TestInit(t *testing.T) {
	tmpDir := t.TempDir()

	err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	beansPath := filepath.Join(tmpDir, BeansDir)
	info, err := os.Stat(beansPath)
	if err != nil {
		t.Fatalf(".beans directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Error(".beans is not a directory")
	}
}

func TestInitIdempotent(t *testing.T) {
	tmpDir := t.TempDir()

	// Call Init twice - should not error
	if err := Init(tmpDir); err != nil {
		t.Fatalf("first Init() error = %v", err)
	}
	if err := Init(tmpDir); err != nil {
		t.Fatalf("second Init() error = %v", err)
	}
}

func TestSave(t *testing.T) {
	store, beansDir := setupTestStore(t)

	bean := &Bean{
		ID:     "abc1",
		Slug:   "test-bean",
		Title:  "Test Bean",
		Status: "open",
		Body:   "Some content here.",
	}

	err := store.Save(bean)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Check file exists
	expectedPath := filepath.Join(beansDir, "abc1--test-bean.md")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("bean file not created at %s", expectedPath)
	}

	// Check timestamps were set
	if bean.CreatedAt == nil {
		t.Error("CreatedAt not set")
	}
	if bean.UpdatedAt == nil {
		t.Error("UpdatedAt not set")
	}

	// Check Path was set
	if bean.Path != "abc1--test-bean.md" {
		t.Errorf("Path = %q, want %q", bean.Path, "abc1--test-bean.md")
	}
}

func TestSavePreservesCreatedAt(t *testing.T) {
	store, _ := setupTestStore(t)

	bean := &Bean{
		ID:     "abc2",
		Title:  "Test",
		Status: "open",
	}

	// First save
	if err := store.Save(bean); err != nil {
		t.Fatalf("first Save() error = %v", err)
	}
	originalCreatedAt := *bean.CreatedAt

	// Second save (update)
	bean.Title = "Updated Title"
	if err := store.Save(bean); err != nil {
		t.Fatalf("second Save() error = %v", err)
	}

	// CreatedAt should be preserved
	if !bean.CreatedAt.Equal(originalCreatedAt) {
		t.Errorf("CreatedAt changed: got %v, want %v", bean.CreatedAt, originalCreatedAt)
	}
}

func TestFindAll(t *testing.T) {
	store, _ := setupTestStore(t)

	// Create some beans
	createTestBean(t, store, "aaa1", "First Bean", "open")
	createTestBean(t, store, "bbb2", "Second Bean", "in-progress")
	createTestBean(t, store, "ccc3", "Third Bean", "done")

	beans, err := store.FindAll()
	if err != nil {
		t.Fatalf("FindAll() error = %v", err)
	}

	if len(beans) != 3 {
		t.Errorf("FindAll() returned %d beans, want 3", len(beans))
	}
}

func TestFindAllEmpty(t *testing.T) {
	store, _ := setupTestStore(t)

	beans, err := store.FindAll()
	if err != nil {
		t.Fatalf("FindAll() error = %v", err)
	}

	if len(beans) != 0 {
		t.Errorf("FindAll() returned %d beans, want 0", len(beans))
	}
}

func TestFindAllIgnoresNonMdFiles(t *testing.T) {
	store, beansDir := setupTestStore(t)

	// Create a bean
	createTestBean(t, store, "abc1", "Real Bean", "open")

	// Create non-.md files that should be ignored
	os.WriteFile(filepath.Join(beansDir, "config.yaml"), []byte("config"), 0644)
	os.WriteFile(filepath.Join(beansDir, "README.txt"), []byte("readme"), 0644)
	os.Mkdir(filepath.Join(beansDir, "subdir"), 0755)

	beans, err := store.FindAll()
	if err != nil {
		t.Fatalf("FindAll() error = %v", err)
	}

	if len(beans) != 1 {
		t.Errorf("FindAll() returned %d beans, want 1 (should ignore non-.md files)", len(beans))
	}
}

func TestFindByID(t *testing.T) {
	store, _ := setupTestStore(t)

	createTestBean(t, store, "abc1", "First", "open")
	createTestBean(t, store, "def2", "Second", "open")
	createTestBean(t, store, "ghi3", "Third", "open")

	t.Run("exact match", func(t *testing.T) {
		bean, err := store.FindByID("abc1")
		if err != nil {
			t.Fatalf("FindByID() error = %v", err)
		}
		if bean.ID != "abc1" {
			t.Errorf("ID = %q, want %q", bean.ID, "abc1")
		}
	})

	t.Run("prefix match", func(t *testing.T) {
		bean, err := store.FindByID("de")
		if err != nil {
			t.Fatalf("FindByID() error = %v", err)
		}
		if bean.ID != "def2" {
			t.Errorf("ID = %q, want %q", bean.ID, "def2")
		}
	})

	t.Run("single char prefix", func(t *testing.T) {
		bean, err := store.FindByID("g")
		if err != nil {
			t.Fatalf("FindByID() error = %v", err)
		}
		if bean.ID != "ghi3" {
			t.Errorf("ID = %q, want %q", bean.ID, "ghi3")
		}
	})
}

func TestFindByIDNotFound(t *testing.T) {
	store, _ := setupTestStore(t)

	createTestBean(t, store, "abc1", "Test", "open")

	_, err := store.FindByID("xyz")
	if err != ErrNotFound {
		t.Errorf("FindByID() error = %v, want ErrNotFound", err)
	}
}

func TestFindByIDAmbiguous(t *testing.T) {
	store, _ := setupTestStore(t)

	// Create beans with similar IDs
	createTestBean(t, store, "abc1", "First", "open")
	createTestBean(t, store, "abc2", "Second", "open")

	_, err := store.FindByID("abc")
	if err != ErrAmbiguousID {
		t.Errorf("FindByID() error = %v, want ErrAmbiguousID", err)
	}
}

func TestDelete(t *testing.T) {
	store, beansDir := setupTestStore(t)

	bean := createTestBean(t, store, "del1", "To Delete", "open")
	filePath := filepath.Join(beansDir, bean.Path)

	// Verify file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatal("bean file should exist before delete")
	}

	// Delete
	err := store.Delete("del1")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify file is gone
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Error("bean file should not exist after delete")
	}
}

func TestDeleteNotFound(t *testing.T) {
	store, _ := setupTestStore(t)

	err := store.Delete("nonexistent")
	if err != ErrNotFound {
		t.Errorf("Delete() error = %v, want ErrNotFound", err)
	}
}

func TestDeleteByPrefix(t *testing.T) {
	store, _ := setupTestStore(t)

	createTestBean(t, store, "unique123", "Test", "open")

	// Delete by prefix
	err := store.Delete("unique")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify it's gone
	_, err = store.FindByID("unique123")
	if err != ErrNotFound {
		t.Error("bean should be deleted")
	}
}

func TestFullPath(t *testing.T) {
	store := NewStore("/path/to/.beans")

	bean := &Bean{
		ID:   "abc1",
		Path: "abc1--test.md",
	}

	got := store.FullPath(bean)
	want := "/path/to/.beans/abc1--test.md"

	if got != want {
		t.Errorf("FullPath() = %q, want %q", got, want)
	}
}

func TestLoadBeanParsesCorrectly(t *testing.T) {
	store, _ := setupTestStore(t)

	// Create a bean with specific content
	original := &Bean{
		ID:     "load1",
		Slug:   "load-test",
		Title:  "Load Test Bean",
		Status: "in-progress",
		Body:   "This is the body content.\n\nWith multiple paragraphs.",
	}
	if err := store.Save(original); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Load it back via FindByID
	loaded, err := store.FindByID("load1")
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}

	// Verify all fields
	if loaded.ID != "load1" {
		t.Errorf("ID = %q, want %q", loaded.ID, "load1")
	}
	if loaded.Slug != "load-test" {
		t.Errorf("Slug = %q, want %q", loaded.Slug, "load-test")
	}
	if loaded.Title != "Load Test Bean" {
		t.Errorf("Title = %q, want %q", loaded.Title, "Load Test Bean")
	}
	if loaded.Status != "in-progress" {
		t.Errorf("Status = %q, want %q", loaded.Status, "in-progress")
	}
	if loaded.Path != "load1--load-test.md" {
		t.Errorf("Path = %q, want %q", loaded.Path, "load1--load-test.md")
	}
}
