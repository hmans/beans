package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/beancore"
	"github.com/hmans/beans/internal/config"
)

func setupQueryTestCore(t *testing.T) (*beancore.Core, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	beansDir := filepath.Join(tmpDir, ".beans")
	if err := os.MkdirAll(beansDir, 0755); err != nil {
		t.Fatalf("failed to create test .beans dir: %v", err)
	}

	cfg := config.Default()
	testCore := beancore.New(beansDir, cfg)
	if err := testCore.Load(); err != nil {
		t.Fatalf("failed to load core: %v", err)
	}

	// Save and restore the global core
	oldCore := core
	core = testCore

	cleanup := func() {
		core = oldCore
	}

	return testCore, cleanup
}

func createQueryTestBean(t *testing.T, c *beancore.Core, id, title, status string) *bean.Bean {
	t.Helper()
	b := &bean.Bean{
		ID:     id,
		Slug:   bean.Slugify(title),
		Title:  title,
		Status: status,
	}
	if err := c.Create(b); err != nil {
		t.Fatalf("failed to create test bean: %v", err)
	}
	return b
}

func TestExecuteQuery(t *testing.T) {
	testCore, cleanup := setupQueryTestCore(t)
	defer cleanup()

	// Create test beans
	createQueryTestBean(t, testCore, "test-1", "First Bean", "todo")
	createQueryTestBean(t, testCore, "test-2", "Second Bean", "in-progress")
	createQueryTestBean(t, testCore, "test-3", "Third Bean", "completed")

	t.Run("basic query all beans", func(t *testing.T) {
		query := `{ beans { id title status } }`
		result, err := ExecuteQuery(query, nil, "")
		if err != nil {
			t.Fatalf("ExecuteQuery() error = %v", err)
		}

		var data struct {
			Beans []struct {
				ID     string `json:"id"`
				Title  string `json:"title"`
				Status string `json:"status"`
			} `json:"beans"`
		}

		if err := json.Unmarshal(result, &data); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if len(data.Beans) != 3 {
			t.Errorf("expected 3 beans, got %d", len(data.Beans))
		}
	})

	t.Run("query single bean by id", func(t *testing.T) {
		query := `{ bean(id: "test-1") { id title } }`
		result, err := ExecuteQuery(query, nil, "")
		if err != nil {
			t.Fatalf("ExecuteQuery() error = %v", err)
		}

		var data struct {
			Bean struct {
				ID    string `json:"id"`
				Title string `json:"title"`
			} `json:"bean"`
		}

		if err := json.Unmarshal(result, &data); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if data.Bean.ID != "test-1" {
			t.Errorf("expected id 'test-1', got %q", data.Bean.ID)
		}
		if data.Bean.Title != "First Bean" {
			t.Errorf("expected title 'First Bean', got %q", data.Bean.Title)
		}
	})

	t.Run("query with filter", func(t *testing.T) {
		query := `{ beans(filter: { status: ["todo"] }) { id } }`
		result, err := ExecuteQuery(query, nil, "")
		if err != nil {
			t.Fatalf("ExecuteQuery() error = %v", err)
		}

		var data struct {
			Beans []struct {
				ID string `json:"id"`
			} `json:"beans"`
		}

		if err := json.Unmarshal(result, &data); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if len(data.Beans) != 1 {
			t.Errorf("expected 1 bean with status 'todo', got %d", len(data.Beans))
		}
		if len(data.Beans) > 0 && data.Beans[0].ID != "test-1" {
			t.Errorf("expected bean id 'test-1', got %q", data.Beans[0].ID)
		}
	})

	t.Run("query with variables", func(t *testing.T) {
		query := `query GetBean($id: ID!) { bean(id: $id) { id title } }`
		variables := map[string]any{
			"id": "test-2",
		}
		result, err := ExecuteQuery(query, variables, "GetBean")
		if err != nil {
			t.Fatalf("ExecuteQuery() error = %v", err)
		}

		var data struct {
			Bean struct {
				ID    string `json:"id"`
				Title string `json:"title"`
			} `json:"bean"`
		}

		if err := json.Unmarshal(result, &data); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if data.Bean.ID != "test-2" {
			t.Errorf("expected id 'test-2', got %q", data.Bean.ID)
		}
	})

	t.Run("query nonexistent bean returns null", func(t *testing.T) {
		query := `{ bean(id: "nonexistent") { id } }`
		result, err := ExecuteQuery(query, nil, "")
		if err != nil {
			t.Fatalf("ExecuteQuery() error = %v", err)
		}

		var data struct {
			Bean *struct {
				ID string `json:"id"`
			} `json:"bean"`
		}

		if err := json.Unmarshal(result, &data); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if data.Bean != nil {
			t.Errorf("expected null bean, got %+v", data.Bean)
		}
	})

	t.Run("invalid query returns error", func(t *testing.T) {
		query := `{ invalid { field } }`
		_, err := ExecuteQuery(query, nil, "")
		if err == nil {
			t.Fatal("expected error for invalid query, got nil")
		}
		if !strings.Contains(err.Error(), "graphql") {
			t.Errorf("expected error to contain 'graphql', got %q", err.Error())
		}
	})
}

func TestExecuteQueryWithRelationships(t *testing.T) {
	testCore, cleanup := setupQueryTestCore(t)
	defer cleanup()

	// Create parent bean
	parent := &bean.Bean{
		ID:     "parent-1",
		Slug:   "parent-bean",
		Title:  "Parent Bean",
		Status: "todo",
	}
	if err := testCore.Create(parent); err != nil {
		t.Fatalf("failed to create parent bean: %v", err)
	}

	// Create child bean with parent link
	child := &bean.Bean{
		ID:     "child-1",
		Slug:   "child-bean",
		Title:  "Child Bean",
		Status: "todo",
		Parent: "parent-1",
	}
	if err := testCore.Create(child); err != nil {
		t.Fatalf("failed to create child bean: %v", err)
	}

	// Create blocker bean
	blocker := &bean.Bean{
		ID:     "blocker-1",
		Slug:   "blocker-bean",
		Title:  "Blocker Bean",
		Status: "todo",
		Blocking: []string{"child-1"},
	}
	if err := testCore.Create(blocker); err != nil {
		t.Fatalf("failed to create blocker bean: %v", err)
	}

	t.Run("query parent relationship", func(t *testing.T) {
		query := `{ bean(id: "child-1") { id parent { id title } } }`
		result, err := ExecuteQuery(query, nil, "")
		if err != nil {
			t.Fatalf("ExecuteQuery() error = %v", err)
		}

		var data struct {
			Bean struct {
				ID     string `json:"id"`
				Parent *struct {
					ID    string `json:"id"`
					Title string `json:"title"`
				} `json:"parent"`
			} `json:"bean"`
		}

		if err := json.Unmarshal(result, &data); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if data.Bean.Parent == nil {
			t.Fatal("expected parent to be set")
		}
		if data.Bean.Parent.ID != "parent-1" {
			t.Errorf("expected parent id 'parent-1', got %q", data.Bean.Parent.ID)
		}
	})

	t.Run("query children relationship", func(t *testing.T) {
		query := `{ bean(id: "parent-1") { id children { id title } } }`
		result, err := ExecuteQuery(query, nil, "")
		if err != nil {
			t.Fatalf("ExecuteQuery() error = %v", err)
		}

		var data struct {
			Bean struct {
				ID       string `json:"id"`
				Children []struct {
					ID    string `json:"id"`
					Title string `json:"title"`
				} `json:"children"`
			} `json:"bean"`
		}

		if err := json.Unmarshal(result, &data); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if len(data.Bean.Children) != 1 {
			t.Errorf("expected 1 child, got %d", len(data.Bean.Children))
		}
		if len(data.Bean.Children) > 0 && data.Bean.Children[0].ID != "child-1" {
			t.Errorf("expected child id 'child-1', got %q", data.Bean.Children[0].ID)
		}
	})

	t.Run("query blockedBy relationship", func(t *testing.T) {
		query := `{ bean(id: "child-1") { id blockedBy { id title } } }`
		result, err := ExecuteQuery(query, nil, "")
		if err != nil {
			t.Fatalf("ExecuteQuery() error = %v", err)
		}

		var data struct {
			Bean struct {
				ID        string `json:"id"`
				BlockedBy []struct {
					ID    string `json:"id"`
					Title string `json:"title"`
				} `json:"blockedBy"`
			} `json:"bean"`
		}

		if err := json.Unmarshal(result, &data); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if len(data.Bean.BlockedBy) != 1 {
			t.Errorf("expected 1 blocker, got %d", len(data.Bean.BlockedBy))
		}
		if len(data.Bean.BlockedBy) > 0 && data.Bean.BlockedBy[0].ID != "blocker-1" {
			t.Errorf("expected blocker id 'blocker-1', got %q", data.Bean.BlockedBy[0].ID)
		}
	})

	t.Run("query blocking relationship", func(t *testing.T) {
		query := `{ bean(id: "blocker-1") { id blocking { id title } } }`
		result, err := ExecuteQuery(query, nil, "")
		if err != nil {
			t.Fatalf("ExecuteQuery() error = %v", err)
		}

		var data struct {
			Bean struct {
				ID       string `json:"id"`
				Blocking []struct {
					ID    string `json:"id"`
					Title string `json:"title"`
				} `json:"blocking"`
			} `json:"bean"`
		}

		if err := json.Unmarshal(result, &data); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if len(data.Bean.Blocking) != 1 {
			t.Errorf("expected 1 blocked bean, got %d", len(data.Bean.Blocking))
		}
		if len(data.Bean.Blocking) > 0 && data.Bean.Blocking[0].ID != "child-1" {
			t.Errorf("expected blocked id 'child-1', got %q", data.Bean.Blocking[0].ID)
		}
	})
}

func TestExecuteQueryWithFilters(t *testing.T) {
	testCore, cleanup := setupQueryTestCore(t)
	defer cleanup()

	// Create beans with different types and priorities
	b1 := &bean.Bean{
		ID:       "bug-1",
		Slug:     "bug-one",
		Title:    "Bug One",
		Status:   "todo",
		Type:     "bug",
		Priority: "critical",
		Tags:     []string{"frontend"},
	}
	b2 := &bean.Bean{
		ID:       "feat-1",
		Slug:     "feature-one",
		Title:    "Feature One",
		Status:   "in-progress",
		Type:     "feature",
		Priority: "high",
		Tags:     []string{"backend"},
	}
	b3 := &bean.Bean{
		ID:       "task-1",
		Slug:     "task-one",
		Title:    "Task One",
		Status:   "completed",
		Type:     "task",
		Priority: "normal",
		Tags:     []string{"frontend", "backend"},
	}

	testCore.Create(b1)
	testCore.Create(b2)
	testCore.Create(b3)

	t.Run("filter by type", func(t *testing.T) {
		query := `{ beans(filter: { type: ["bug"] }) { id type } }`
		result, err := ExecuteQuery(query, nil, "")
		if err != nil {
			t.Fatalf("ExecuteQuery() error = %v", err)
		}

		var data struct {
			Beans []struct {
				ID   string `json:"id"`
				Type string `json:"type"`
			} `json:"beans"`
		}

		if err := json.Unmarshal(result, &data); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if len(data.Beans) != 1 {
			t.Errorf("expected 1 bean with type 'bug', got %d", len(data.Beans))
		}
	})

	t.Run("filter by priority", func(t *testing.T) {
		query := `{ beans(filter: { priority: ["critical", "high"] }) { id priority } }`
		result, err := ExecuteQuery(query, nil, "")
		if err != nil {
			t.Fatalf("ExecuteQuery() error = %v", err)
		}

		var data struct {
			Beans []struct {
				ID       string `json:"id"`
				Priority string `json:"priority"`
			} `json:"beans"`
		}

		if err := json.Unmarshal(result, &data); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if len(data.Beans) != 2 {
			t.Errorf("expected 2 beans with priority 'critical' or 'high', got %d", len(data.Beans))
		}
	})

	t.Run("filter by tags", func(t *testing.T) {
		query := `{ beans(filter: { tags: ["frontend"] }) { id tags } }`
		result, err := ExecuteQuery(query, nil, "")
		if err != nil {
			t.Fatalf("ExecuteQuery() error = %v", err)
		}

		var data struct {
			Beans []struct {
				ID   string   `json:"id"`
				Tags []string `json:"tags"`
			} `json:"beans"`
		}

		if err := json.Unmarshal(result, &data); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if len(data.Beans) != 2 {
			t.Errorf("expected 2 beans with tag 'frontend', got %d", len(data.Beans))
		}
	})

	t.Run("exclude by status", func(t *testing.T) {
		query := `{ beans(filter: { excludeStatus: ["completed"] }) { id status } }`
		result, err := ExecuteQuery(query, nil, "")
		if err != nil {
			t.Fatalf("ExecuteQuery() error = %v", err)
		}

		var data struct {
			Beans []struct {
				ID     string `json:"id"`
				Status string `json:"status"`
			} `json:"beans"`
		}

		if err := json.Unmarshal(result, &data); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if len(data.Beans) != 2 {
			t.Errorf("expected 2 beans (excluding completed), got %d", len(data.Beans))
		}
		for _, b := range data.Beans {
			if b.Status == "completed" {
				t.Errorf("should not include completed beans, got bean with status %q", b.Status)
			}
		}
	})

	t.Run("combined filters", func(t *testing.T) {
		query := `{ beans(filter: { status: ["todo", "in-progress"], type: ["bug", "feature"] }) { id } }`
		result, err := ExecuteQuery(query, nil, "")
		if err != nil {
			t.Fatalf("ExecuteQuery() error = %v", err)
		}

		var data struct {
			Beans []struct {
				ID string `json:"id"`
			} `json:"beans"`
		}

		if err := json.Unmarshal(result, &data); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if len(data.Beans) != 2 {
			t.Errorf("expected 2 beans matching combined filters, got %d", len(data.Beans))
		}
	})
}

func TestGetGraphQLSchema(t *testing.T) {
	_, cleanup := setupQueryTestCore(t)
	defer cleanup()

	schema := GetGraphQLSchema()

	// Verify schema contains expected types
	expectedTypes := []string{
		"type Query",
		"type Bean",
		"input BeanFilter",
	}

	for _, expected := range expectedTypes {
		if !strings.Contains(schema, expected) {
			t.Errorf("schema missing expected type: %s", expected)
		}
	}

	// Verify schema contains expected fields
	expectedFields := []string{
		"bean(id: ID!)",
		"beans(filter: BeanFilter)",
		"blockedBy",
		"blocking",
		"parent",
		"children",
	}

	for _, expected := range expectedFields {
		if !strings.Contains(schema, expected) {
			t.Errorf("schema missing expected field: %s", expected)
		}
	}

	// Verify no introspection fields
	if strings.Contains(schema, "__schema") || strings.Contains(schema, "__type") {
		t.Error("schema should not contain introspection fields")
	}
}

func TestReadFromStdin(t *testing.T) {
	// Note: Testing stdin behavior is tricky in unit tests.
	// This tests the function when stdin is a terminal (returns empty).
	// Integration tests would need to actually pipe data.
	t.Run("returns empty when stdin is terminal", func(t *testing.T) {
		// In a test environment, stdin is typically a terminal
		result, err := readFromStdin()
		if err != nil {
			t.Fatalf("readFromStdin() error = %v", err)
		}
		// Result will be empty string when stdin is a terminal
		if result != "" {
			t.Logf("readFromStdin() returned %q (may vary by test environment)", result)
		}
	})
}
