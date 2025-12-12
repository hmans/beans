package search

import (
	"slices"
	"testing"

	"github.com/hmans/beans/internal/bean"
)

func setupTestIndex(t *testing.T) *Index {
	t.Helper()
	idx, err := NewIndex()
	if err != nil {
		t.Fatalf("NewIndex() error = %v", err)
	}
	t.Cleanup(func() {
		idx.Close()
	})
	return idx
}

func TestNewIndex(t *testing.T) {
	idx, err := NewIndex()
	if err != nil {
		t.Fatalf("NewIndex() error = %v", err)
	}
	defer idx.Close()
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
	if !slices.Contains(ids, "aaa1") {
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

func TestIndexBeans(t *testing.T) {
	idx := setupTestIndex(t)

	beans := []*bean.Bean{
		{ID: "aaa1", Title: "Bean One", Body: "First content"},
		{ID: "bbb2", Title: "Bean Two", Body: "Second content"},
		{ID: "ccc3", Title: "Bean Three", Body: "Third content"},
	}

	if err := idx.IndexBeans(beans); err != nil {
		t.Fatalf("IndexBeans() error = %v", err)
	}

	// All beans should be searchable
	ids, err := idx.Search("Bean", 10)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(ids) != 3 {
		t.Errorf("Search(Bean) returned %d results, want 3", len(ids))
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
	for range 20 {
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

func TestFindLinkingBeans(t *testing.T) {
	idx := setupTestIndex(t)

	// Create beans with various link types
	epic := &bean.Bean{ID: "epic1", Title: "Epic"}
	milestone := &bean.Bean{ID: "milestone1", Title: "Milestone"}
	feature := &bean.Bean{ID: "feature1", Title: "Feature", Epic: "epic1"}
	task1 := &bean.Bean{ID: "task1", Title: "Task 1", Epic: "epic1", Milestone: "milestone1"}
	task2 := &bean.Bean{ID: "task2", Title: "Task 2", Epic: "epic1", Blocks: []string{"task1"}}

	beans := []*bean.Bean{epic, milestone, feature, task1, task2}
	if err := idx.IndexBeans(beans); err != nil {
		t.Fatalf("IndexBeans() error = %v", err)
	}

	t.Run("find beans linking to epic", func(t *testing.T) {
		ids, err := idx.FindLinkingBeans("epic1")
		if err != nil {
			t.Fatalf("FindLinkingBeans() error = %v", err)
		}

		// Should find feature1, task1, task2 all linking to epic1
		if len(ids) != 3 {
			t.Errorf("FindLinkingBeans(epic1) returned %d results, want 3: %v", len(ids), ids)
		}
	})

	t.Run("find beans linking to milestone", func(t *testing.T) {
		ids, err := idx.FindLinkingBeans("milestone1")
		if err != nil {
			t.Fatalf("FindLinkingBeans() error = %v", err)
		}

		if len(ids) != 1 {
			t.Errorf("FindLinkingBeans(milestone1) returned %d results, want 1", len(ids))
		}
		if ids[0] != "task1" {
			t.Errorf("expected task1, got %s", ids[0])
		}
	})

	t.Run("find beans linking to task1", func(t *testing.T) {
		ids, err := idx.FindLinkingBeans("task1")
		if err != nil {
			t.Fatalf("FindLinkingBeans() error = %v", err)
		}

		// task2 blocks task1
		if len(ids) != 1 {
			t.Errorf("FindLinkingBeans(task1) returned %d results, want 1: %v", len(ids), ids)
		}
	})

	t.Run("no linking beans", func(t *testing.T) {
		ids, err := idx.FindLinkingBeans("nonexistent")
		if err != nil {
			t.Fatalf("FindLinkingBeans() error = %v", err)
		}

		if len(ids) != 0 {
			t.Errorf("FindLinkingBeans(nonexistent) returned %d results, want 0", len(ids))
		}
	})
}
