package cmd

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"os"
	"sync"
	"text/template"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
	"github.com/hmans/beans/internal/beancore"
	"github.com/hmans/beans/internal/config"
)

//go:embed mcp.tmpl
var mcpToolTemplate string

// GraphQLInput is the input schema for the beans_graphql tool.
type GraphQLInput struct {
	Query         string         `json:"query" jsonschema:"The GraphQL query or mutation to execute"`
	Variables     map[string]any `json:"variables,omitempty" jsonschema:"Optional variables for the query"`
	OperationName string         `json:"operationName,omitempty" jsonschema:"Optional operation name for multi-operation documents"`
}

// mcpCore holds the lazily-initialized core for MCP operations.
var (
	mcpCore     *beancore.Core
	mcpCoreOnce sync.Once
	mcpCoreErr  error
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Run MCP server for AI assistant integration",
	Long: `Run an MCP (Model Context Protocol) server that exposes beans functionality
to AI assistants like Claude Code.

The server provides a single 'beans_graphql' tool that accepts GraphQL queries
and mutations, giving AI assistants full access to query and manage beans.

To use with Claude Code, add to your MCP configuration:
  {
    "mcpServers": {
      "beans": {
        "command": "beans",
        "args": ["mcp"]
      }
    }
  }`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMCPServer()
	},
}

func runMCPServer() error {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "beans",
		Version: "0.1.0",
	}, nil)

	// Generate tool description with schema and usage docs
	toolDescription, err := generateMCPToolDescription()
	if err != nil {
		return fmt.Errorf("generating tool description: %w", err)
	}

	// Register the single beans_graphql tool
	mcp.AddTool(server, &mcp.Tool{
		Name:        "beans_graphql",
		Description: toolDescription,
	}, handleGraphQLTool)

	// Run the server on stdio
	return server.Run(context.Background(), &mcp.StdioTransport{})
}

// handleGraphQLTool handles calls to the beans_graphql tool.
func handleGraphQLTool(ctx context.Context, req *mcp.CallToolRequest, args GraphQLInput) (*mcp.CallToolResult, any, error) {
	// Lazily initialize core on first tool call
	if err := initMCPCore(); err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
		}, nil, nil
	}

	// Execute the GraphQL query using the MCP-specific core
	result, err := executeMCPQuery(args.Query, args.Variables, args.OperationName)
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
		}, nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(result)}},
	}, nil, nil
}

// initMCPCore initializes the core for MCP operations.
// It's called lazily on the first tool call.
func initMCPCore() error {
	mcpCoreOnce.Do(func() {
		// Search upward for .beans.yml config
		cwd, err := os.Getwd()
		if err != nil {
			mcpCoreErr = fmt.Errorf("getting current directory: %w", err)
			return
		}

		mcpCfg, err := config.LoadFromDirectory(cwd)
		if err != nil {
			mcpCoreErr = fmt.Errorf("no beans project found in %s or parent directories. Run 'beans init' to create one", cwd)
			return
		}

		// Use path from config
		root := mcpCfg.ResolveBeansPath()

		// Verify it exists
		if info, statErr := os.Stat(root); statErr != nil || !info.IsDir() {
			mcpCoreErr = fmt.Errorf("no .beans directory found at %s. Run 'beans init' to create one", root)
			return
		}

		mcpCore = beancore.New(root, mcpCfg)
		if err := mcpCore.Load(); err != nil {
			mcpCoreErr = fmt.Errorf("loading beans: %w", err)
			return
		}
	})

	return mcpCoreErr
}

// executeMCPQuery runs a GraphQL query using the MCP-specific core.
// This is separate from ExecuteQuery to use the lazily-initialized mcpCore.
func executeMCPQuery(query string, variables map[string]any, operationName string) ([]byte, error) {
	es := newExecutableSchemaForCore(mcpCore)
	return executeQueryWithSchema(es, query, variables, operationName)
}

// generateMCPToolDescription generates the tool description from the template.
func generateMCPToolDescription() (string, error) {
	tmpl, err := template.New("mcp").Parse(mcpToolTemplate)
	if err != nil {
		return "", err
	}

	data := promptData{
		GraphQLSchema: GetGraphQLSchema(),
		Types:         config.DefaultTypes,
		Statuses:      config.DefaultStatuses,
		Priorities:    config.DefaultPriorities,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}
