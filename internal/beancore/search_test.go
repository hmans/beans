package beancore

import (
	"os"
	"testing"

	"github.com/hmans/beans/internal/bean"
)

func TestSearch(t *testing.T) {
	core, _ := setupTestCore(t)
	defer core.Close()

	// Create beans with searchable content
	beans := []*bean.Bean{
		{ID: "aaa1", Slug: "auth", Title: "User Authentication", Body: "Implement login system"},
		{ID: "bbb2", Slug: "db", Title: "Database Schema", Body: "Create tables for users"},
		{ID: "ccc3", Slug: "api", Title: "API Endpoints", Body: "REST interface for authentication"},
	}

	for _, b := range beans {
		if err := core.Create(b); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	// Search by title
	results, err := core.Search("Authentication")
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Search(Authentication) returned %d results, want 2", len(results))
	}
}

func TestSearch_ByBody(t *testing.T) {
	core, _ := setupTestCore(t)
	defer core.Close()

	beans := []*bean.Bean{
		{ID: "aaa1", Title: "Feature A", Body: "Implement JWT tokens"},
		{ID: "bbb2", Title: "Feature B", Body: "Add database migrations"},
	}

	for _, b := range beans {
		if err := core.Create(b); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	results, err := core.Search("JWT")
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(results) != 1 || results[0].ID != "aaa1" {
		t.Errorf("Search(JWT) = %v, want [aaa1]", results)
	}
}

func TestSearch_LazyInit(t *testing.T) {
	core, _ := setupTestCore(t)
	defer core.Close()

	// Create a bean first (before any search)
	b := &bean.Bean{
		ID:    "abc1",
		Title: "Test Bean",
		Body:  "Content",
	}
	if err := core.Create(b); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Search should lazily initialize the index and index existing beans
	results, err := core.Search("Test")
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Search(Test) returned %d results, want 1 (lazy init should index existing beans)", len(results))
	}
}

func TestSearch_CreateUpdatesIndex(t *testing.T) {
	core, _ := setupTestCore(t)
	defer core.Close()

	// Initialize search index by doing a search first
	_, _ = core.Search("anything")

	// Create a new bean
	b := &bean.Bean{
		ID:    "new1",
		Title: "New Bean",
		Body:  "Fresh content",
	}
	if err := core.Create(b); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Search should find the new bean
	results, err := core.Search("Fresh")
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(results) != 1 || results[0].ID != "new1" {
		t.Errorf("Search(Fresh) = %v, want [new1]", results)
	}
}

func TestSearch_UpdateUpdatesIndex(t *testing.T) {
	core, _ := setupTestCore(t)
	defer core.Close()

	// Create and index a bean
	b := &bean.Bean{
		ID:    "upd1",
		Title: "Original Title",
		Body:  "Original content",
	}
	if err := core.Create(b); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Initialize index
	_, _ = core.Search("Original")

	// Update the bean
	b.Title = "Updated Title"
	b.Body = "Modified content"
	if err := core.Update(b); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Search should find by new content
	results, err := core.Search("Modified")
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(results) != 1 || results[0].ID != "upd1" {
		t.Errorf("Search(Modified) = %v, want [upd1]", results)
	}
}

func TestSearch_DeleteUpdatesIndex(t *testing.T) {
	core, _ := setupTestCore(t)
	defer core.Close()

	// Create and index a bean
	b := &bean.Bean{
		ID:    "del1",
		Title: "To Delete",
		Body:  "Unique keyword deleteme",
	}
	if err := core.Create(b); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Initialize index
	results, _ := core.Search("deleteme")
	if len(results) != 1 {
		t.Fatal("bean should be indexed before delete")
	}

	// Delete the bean
	if err := core.Delete("del1"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Search should NOT find the deleted bean
	results, err := core.Search("deleteme")
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Search(deleteme) after delete = %v, want []", results)
	}
}

func TestSearch_LoadRebuildsIndex(t *testing.T) {
	core, beansDir := setupTestCore(t)
	defer core.Close()

	// Create a bean
	b := &bean.Bean{
		ID:    "abc1",
		Title: "Initial Bean",
		Body:  "Content",
	}
	if err := core.Create(b); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Initialize index
	_, _ = core.Search("Initial")

	// Write a new bean file directly (simulating external change)
	content := `---
title: External Bean
status: todo
---

External content keyword.
`
	if err := writeTestFile(beansDir, "ext1--external.md", content); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Reload from disk
	if err := core.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Search should find the externally added bean
	results, err := core.Search("External")
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(results) != 1 || results[0].ID != "ext1" {
		t.Errorf("Search(External) = %v, want [ext1]", results)
	}
}

func TestSearch_NoResults(t *testing.T) {
	core, _ := setupTestCore(t)
	defer core.Close()

	b := &bean.Bean{
		ID:    "abc1",
		Title: "Test Bean",
		Body:  "Content",
	}
	if err := core.Create(b); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	results, err := core.Search("nonexistent")
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Search(nonexistent) = %v, want []", results)
	}
}

func TestClose_ClosesSearchIndex(t *testing.T) {
	core, _ := setupTestCore(t)

	// Create a bean and search to initialize index
	b := &bean.Bean{
		ID:    "abc1",
		Title: "Test Bean",
		Body:  "Content",
	}
	if err := core.Create(b); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	_, _ = core.Search("Test")

	// Close should not error
	if err := core.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

// Helper to write test files
func writeTestFile(dir, name, content string) error {
	return os.WriteFile(dir+"/"+name, []byte(content), 0644)
}
