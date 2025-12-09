package search

import (
	"testing"

	"github.com/hmans/beans/internal/bean"
)

func setupTestIndex(t *testing.T) *Index {
	t.Helper()
	indexPath := t.TempDir() + "/test.bleve"
	idx, _, err := NewIndex(indexPath)
	if err != nil {
		t.Fatalf("NewIndex() error = %v", err)
	}
	t.Cleanup(func() {
		idx.Close()
	})
	return idx
}

func TestNewIndex(t *testing.T) {
	indexPath := t.TempDir() + "/test.bleve"
	idx, rebuilt, err := NewIndex(indexPath)
	if err != nil {
		t.Fatalf("NewIndex() error = %v", err)
	}
	defer idx.Close()

	if idx.Path() != indexPath {
		t.Errorf("Path() = %q, want %q", idx.Path(), indexPath)
	}
	if !rebuilt {
		t.Error("NewIndex() should return rebuilt=true for new index")
	}
}

func TestNewIndex_OpenExisting(t *testing.T) {
	indexPath := t.TempDir() + "/test.bleve"

	// Create index
	idx1, rebuilt1, err := NewIndex(indexPath)
	if err != nil {
		t.Fatalf("NewIndex() error = %v", err)
	}
	if !rebuilt1 {
		t.Error("first NewIndex() should return rebuilt=true")
	}

	// Index a document
	b := &bean.Bean{
		ID:    "abc1",
		Title: "Test Bean",
		Body:  "Test content",
	}
	if err := idx1.IndexBean(b); err != nil {
		t.Fatalf("IndexBean() error = %v", err)
	}
	idx1.Close()

	// Reopen index
	idx2, rebuilt2, err := NewIndex(indexPath)
	if err != nil {
		t.Fatalf("NewIndex() reopen error = %v", err)
	}
	defer idx2.Close()

	if rebuilt2 {
		t.Error("second NewIndex() should return rebuilt=false for existing index")
	}

	// Search should find the document
	ids, err := idx2.Search("Test", 10)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(ids) != 1 || ids[0] != "abc1" {
		t.Errorf("Search() = %v, want [abc1]", ids)
	}
}

func TestIndexBean(t *testing.T) {
	idx := setupTestIndex(t)

	b := &bean.Bean{
		ID:    "abc1",
		Title: "Authentication System",
		Body:  "Implement JWT tokens for user authentication",
	}

	err := idx.IndexBean(b)
	if err != nil {
		t.Fatalf("IndexBean() error = %v", err)
	}

	// Search should find by title
	ids, err := idx.Search("Authentication", 10)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(ids) != 1 {
		t.Errorf("Search() returned %d results, want 1", len(ids))
	}
}

func TestSearch_MatchTitle(t *testing.T) {
	idx := setupTestIndex(t)

	beans := []*bean.Bean{
		{ID: "aaa1", Title: "User Authentication", Body: "Login system"},
		{ID: "bbb2", Title: "Database Schema", Body: "Table definitions"},
		{ID: "ccc3", Title: "API Endpoints", Body: "REST interface"},
	}

	for _, b := range beans {
		if err := idx.IndexBean(b); err != nil {
			t.Fatalf("IndexBean() error = %v", err)
		}
	}

	ids, err := idx.Search("Authentication", 10)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(ids) != 1 || ids[0] != "aaa1" {
		t.Errorf("Search(Authentication) = %v, want [aaa1]", ids)
	}
}

func TestSearch_MatchBody(t *testing.T) {
	idx := setupTestIndex(t)

	beans := []*bean.Bean{
		{ID: "aaa1", Title: "Feature A", Body: "Implement JWT tokens"},
		{ID: "bbb2", Title: "Feature B", Body: "Add database migrations"},
		{ID: "ccc3", Title: "Feature C", Body: "Update UI components"},
	}

	for _, b := range beans {
		if err := idx.IndexBean(b); err != nil {
			t.Fatalf("IndexBean() error = %v", err)
		}
	}

	ids, err := idx.Search("JWT", 10)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(ids) != 1 || ids[0] != "aaa1" {
		t.Errorf("Search(JWT) = %v, want [aaa1]", ids)
	}
}

func TestSearch_MatchSlug(t *testing.T) {
	idx := setupTestIndex(t)

	beans := []*bean.Bean{
		{ID: "aaa1", Slug: "auth-feature", Title: "Feature A", Body: "Some content"},
		{ID: "bbb2", Slug: "database-migration", Title: "Feature B", Body: "Other content"},
		{ID: "ccc3", Slug: "ui-update", Title: "Feature C", Body: "More content"},
	}

	for _, b := range beans {
		if err := idx.IndexBean(b); err != nil {
			t.Fatalf("IndexBean() error = %v", err)
		}
	}

	// Search by slug content
	ids, err := idx.Search("auth", 10)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(ids) != 1 || ids[0] != "aaa1" {
		t.Errorf("Search(auth) = %v, want [aaa1]", ids)
	}
}

func TestSearch_MultipleResults(t *testing.T) {
	idx := setupTestIndex(t)

	beans := []*bean.Bean{
		{ID: "aaa1", Title: "User Login", Body: "Authentication flow"},
		{ID: "bbb2", Title: "User Registration", Body: "Sign up form"},
		{ID: "ccc3", Title: "Admin Panel", Body: "Dashboard"},
	}

	for _, b := range beans {
		if err := idx.IndexBean(b); err != nil {
			t.Fatalf("IndexBean() error = %v", err)
		}
	}

	ids, err := idx.Search("User", 10)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(ids) != 2 {
		t.Errorf("Search(User) returned %d results, want 2", len(ids))
	}
}

func TestSearch_NoResults(t *testing.T) {
	idx := setupTestIndex(t)

	b := &bean.Bean{
		ID:    "abc1",
		Title: "Test Bean",
		Body:  "Some content",
	}
	if err := idx.IndexBean(b); err != nil {
		t.Fatalf("IndexBean() error = %v", err)
	}

	ids, err := idx.Search("nonexistent", 10)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(ids) != 0 {
		t.Errorf("Search(nonexistent) = %v, want []", ids)
	}
}

func TestSearch_EmptyQuery(t *testing.T) {
	idx := setupTestIndex(t)

	b := &bean.Bean{
		ID:    "abc1",
		Title: "Test Bean",
		Body:  "Some content",
	}
	if err := idx.IndexBean(b); err != nil {
		t.Fatalf("IndexBean() error = %v", err)
	}

	// Empty query returns no results (Bleve matches nothing)
	ids, err := idx.Search("", 10)
	if err != nil {
		t.Fatalf("Search('') error = %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("Search('') = %v, want []", ids)
	}
}

func TestSearch_BooleanQuery(t *testing.T) {
	idx := setupTestIndex(t)

	beans := []*bean.Bean{
		{ID: "aaa1", Title: "User Authentication", Body: "Login with password"},
		{ID: "bbb2", Title: "User Registration", Body: "Create account"},
		{ID: "ccc3", Title: "Admin Authentication", Body: "Admin login"},
	}

	for _, b := range beans {
		if err := idx.IndexBean(b); err != nil {
			t.Fatalf("IndexBean() error = %v", err)
		}
	}

	// Search for "User AND Authentication"
	ids, err := idx.Search("User Authentication", 10)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	// Should match aaa1 (has both terms)
	found := false
	for _, id := range ids {
		if id == "aaa1" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Search(User Authentication) = %v, expected to include aaa1", ids)
	}
}

func TestSearch_Wildcard(t *testing.T) {
	idx := setupTestIndex(t)

	beans := []*bean.Bean{
		{ID: "aaa1", Title: "Authentication", Body: ""},
		{ID: "bbb2", Title: "Authorization", Body: ""},
		{ID: "ccc3", Title: "Automation", Body: ""},
	}

	for _, b := range beans {
		if err := idx.IndexBean(b); err != nil {
			t.Fatalf("IndexBean() error = %v", err)
		}
	}

	// Search with wildcard - note: Bleve wildcards are case-sensitive and work on lowercase tokens
	ids, err := idx.Search("auth*", 10)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(ids) != 2 {
		t.Errorf("Search(auth*) returned %d results, want 2 (Authentication, Authorization)", len(ids))
	}
}

func TestDeleteBean(t *testing.T) {
	idx := setupTestIndex(t)

	b := &bean.Bean{
		ID:    "abc1",
		Title: "Test Bean",
		Body:  "Some content",
	}
	if err := idx.IndexBean(b); err != nil {
		t.Fatalf("IndexBean() error = %v", err)
	}

	// Verify it's indexed
	ids, _ := idx.Search("Test", 10)
	if len(ids) != 1 {
		t.Fatal("bean should be indexed before delete")
	}

	// Delete
	if err := idx.DeleteBean("abc1"); err != nil {
		t.Fatalf("DeleteBean() error = %v", err)
	}

	// Verify it's gone
	ids, _ = idx.Search("Test", 10)
	if len(ids) != 0 {
		t.Errorf("Search after delete = %v, want []", ids)
	}
}

func TestRebuildFromBeans(t *testing.T) {
	idx := setupTestIndex(t)

	// Index initial beans
	initialBeans := []*bean.Bean{
		{ID: "old1", Title: "Old Bean 1", Body: ""},
		{ID: "old2", Title: "Old Bean 2", Body: ""},
	}
	for _, b := range initialBeans {
		if err := idx.IndexBean(b); err != nil {
			t.Fatalf("IndexBean() error = %v", err)
		}
	}

	// Rebuild with new beans
	newBeans := []*bean.Bean{
		{ID: "new1", Title: "New Bean 1", Body: ""},
		{ID: "new2", Title: "New Bean 2", Body: ""},
		{ID: "new3", Title: "New Bean 3", Body: ""},
	}

	if err := idx.RebuildFromBeans(newBeans); err != nil {
		t.Fatalf("RebuildFromBeans() error = %v", err)
	}

	// Old beans should not be found
	ids, _ := idx.Search("Old", 10)
	if len(ids) != 0 {
		t.Errorf("Search(Old) after rebuild = %v, want []", ids)
	}

	// New beans should be found
	ids, _ = idx.Search("New", 10)
	if len(ids) != 3 {
		t.Errorf("Search(New) after rebuild returned %d results, want 3", len(ids))
	}
}

func TestIndexBean_Update(t *testing.T) {
	idx := setupTestIndex(t)

	// Index initial version
	b := &bean.Bean{
		ID:    "abc1",
		Title: "Original Title",
		Body:  "Original content",
	}
	if err := idx.IndexBean(b); err != nil {
		t.Fatalf("IndexBean() error = %v", err)
	}

	// Update the bean
	b.Title = "Updated Title"
	b.Body = "Updated content"
	if err := idx.IndexBean(b); err != nil {
		t.Fatalf("IndexBean() update error = %v", err)
	}

	// Should find by new title
	ids, _ := idx.Search("Updated", 10)
	if len(ids) != 1 || ids[0] != "abc1" {
		t.Errorf("Search(Updated) = %v, want [abc1]", ids)
	}

	// Should NOT find by old title
	ids, _ = idx.Search("Original", 10)
	if len(ids) != 0 {
		t.Errorf("Search(Original) after update = %v, want []", ids)
	}
}

func TestSearch_Limit(t *testing.T) {
	idx := setupTestIndex(t)

	// Index many beans
	for i := 0; i < 20; i++ {
		b := &bean.Bean{
			ID:    bean.NewID("", 4),
			Title: "Test Bean",
			Body:  "Content",
		}
		if err := idx.IndexBean(b); err != nil {
			t.Fatalf("IndexBean() error = %v", err)
		}
	}

	// Search with limit
	ids, err := idx.Search("Test", 5)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(ids) != 5 {
		t.Errorf("Search with limit 5 returned %d results, want 5", len(ids))
	}
}

func TestSearch_DefaultLimit(t *testing.T) {
	idx := setupTestIndex(t)

	b := &bean.Bean{
		ID:    "abc1",
		Title: "Test",
		Body:  "",
	}
	if err := idx.IndexBean(b); err != nil {
		t.Fatalf("IndexBean() error = %v", err)
	}

	// Search with 0 limit should use default
	ids, err := idx.Search("Test", 0)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(ids) != 1 {
		t.Errorf("Search with limit 0 (default) returned %d results, want 1", len(ids))
	}
}

func TestIndexPath(t *testing.T) {
	got := IndexPath("/path/to/.beans")
	want := "/path/to/.beans/.index"

	if got != want {
		t.Errorf("IndexPath() = %q, want %q", got, want)
	}
}
