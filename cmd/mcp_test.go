package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/beancore"
	"github.com/hmans/beans/internal/config"
)

func setupMCPTestCore(t *testing.T) (*beancore.Core, func()) {
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

	// Save and restore the MCP global state
	oldCore := mcpCore
	oldOnce := mcpCoreOnce
	oldErr := mcpCoreErr

	// Reset MCP state
	mcpCore = testCore
	mcpCoreOnce = sync.Once{}
	mcpCoreErr = nil
	// Mark as already initialized
	mcpCoreOnce.Do(func() {})

	cleanup := func() {
		mcpCore = oldCore
		mcpCoreOnce = oldOnce
		mcpCoreErr = oldErr
	}

	return testCore, cleanup
}

func TestExecuteMCPQuery(t *testing.T) {
	testCore, cleanup := setupMCPTestCore(t)
	defer cleanup()

	// Create test beans
	b1 := &bean.Bean{
		ID:     "mcp-test-1",
		Slug:   "mcp-test-bean",
		Title:  "MCP Test Bean",
		Status: "todo",
		Type:   "task",
	}
	if err := testCore.Create(b1); err != nil {
		t.Fatalf("failed to create test bean: %v", err)
	}

	t.Run("basic query", func(t *testing.T) {
		result, err := executeMCPQuery(`{ beans { id title } }`, nil, "")
		if err != nil {
			t.Fatalf("executeMCPQuery() error = %v", err)
		}

		if !strings.Contains(string(result), "mcp-test-1") {
			t.Errorf("expected result to contain 'mcp-test-1', got %s", string(result))
		}
	})

	t.Run("query with variables", func(t *testing.T) {
		variables := map[string]any{"id": "mcp-test-1"}
		result, err := executeMCPQuery(`query GetBean($id: ID!) { bean(id: $id) { title } }`, variables, "GetBean")
		if err != nil {
			t.Fatalf("executeMCPQuery() error = %v", err)
		}

		if !strings.Contains(string(result), "MCP Test Bean") {
			t.Errorf("expected result to contain 'MCP Test Bean', got %s", string(result))
		}
	})

	t.Run("invalid query returns error", func(t *testing.T) {
		_, err := executeMCPQuery(`{ invalid }`, nil, "")
		if err == nil {
			t.Fatal("expected error for invalid query, got nil")
		}
	})
}

func TestGenerateMCPToolDescription(t *testing.T) {
	// Set up a minimal core for schema generation
	_, cleanup := setupMCPTestCore(t)
	defer cleanup()

	// Also set up the global core since GetGraphQLSchema uses it
	oldCore := core
	core = mcpCore
	defer func() { core = oldCore }()

	description, err := generateMCPToolDescription()
	if err != nil {
		t.Fatalf("generateMCPToolDescription() error = %v", err)
	}

	// Verify description contains expected content
	expectedContent := []string{
		"GraphQL",
		"beans",
		"query",
		"mutation",
		"type Bean",
	}

	for _, expected := range expectedContent {
		if !strings.Contains(description, expected) {
			t.Errorf("description missing expected content: %s", expected)
		}
	}
}

func TestGraphQLInput(t *testing.T) {
	// Test that GraphQLInput struct has the expected fields
	input := GraphQLInput{
		Query:         "{ beans { id } }",
		Variables:     map[string]any{"foo": "bar"},
		OperationName: "GetBeans",
	}

	if input.Query != "{ beans { id } }" {
		t.Errorf("unexpected Query value")
	}
	if input.Variables["foo"] != "bar" {
		t.Errorf("unexpected Variables value")
	}
	if input.OperationName != "GetBeans" {
		t.Errorf("unexpected OperationName value")
	}
}
