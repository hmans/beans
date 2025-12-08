package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/executor"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/vektah/gqlparser/v2/formatter"
	"hmans.dev/beans/internal/graph"
)

var (
	queryJSON       bool
	queryVariables  string
	queryOperation  string
	querySchemaOnly bool
)

var queryCmd = &cobra.Command{
	Use:   "query <graphql>",
	Short: "Execute a GraphQL query",
	Long: `Execute a GraphQL query against the beans data.

The query argument should be a valid GraphQL query string.

Examples:
  # List all beans
  beans query '{ beans { id title status } }'

  # Get a specific bean
  beans query '{ bean(id: "abc") { title status body } }'

  # Filter beans by status
  beans query '{ beans(filter: { status: ["todo", "in-progress"] }) { id title } }'

  # Get beans with relationships
  beans query '{ beans { id title blockedBy { id title } children { id title } } }'

  # Use variables
  beans query -v '{"id": "abc"}' 'query GetBean($id: ID!) { bean(id: $id) { title } }'

  # Read query from stdin (useful for complex queries or escaping issues)
  echo '{ beans { id title } }' | beans query
  cat query.graphql | beans query

  # Print the schema
  beans query --schema`,
	Args: func(cmd *cobra.Command, args []string) error {
		if querySchemaOnly {
			return nil
		}
		// Allow 0 args if stdin has data, or exactly 1 arg
		if len(args) > 1 {
			return fmt.Errorf("accepts at most 1 argument (the GraphQL query)")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Schema-only mode
		if querySchemaOnly {
			return printSchema()
		}

		var query string
		if len(args) == 1 {
			query = args[0]
		} else {
			// Try to read from stdin
			stdinQuery, err := readFromStdin()
			if err != nil {
				return err
			}
			if stdinQuery == "" {
				return fmt.Errorf("no query provided (pass as argument or pipe to stdin)")
			}
			query = stdinQuery
		}

		// Parse variables if provided
		var variables map[string]any
		if queryVariables != "" {
			if err := json.Unmarshal([]byte(queryVariables), &variables); err != nil {
				return fmt.Errorf("invalid variables JSON: %w", err)
			}
		}

		// Execute the query
		result, err := executeQuery(query, variables, queryOperation)
		if err != nil {
			return err
		}

		// Output
		if queryJSON {
			fmt.Println(string(result))
		} else {
			prettyPrint(result)
		}

		return nil
	},
}

// readFromStdin reads the query from stdin if data is available.
func readFromStdin() (string, error) {
	// Check if stdin has data (is a pipe or file, not a terminal)
	stat, err := os.Stdin.Stat()
	if err != nil {
		return "", fmt.Errorf("checking stdin: %w", err)
	}

	// If stdin is a terminal (no pipe), return empty
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		return "", nil
	}

	// Read all data from stdin
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", fmt.Errorf("reading stdin: %w", err)
	}

	return strings.TrimSpace(string(data)), nil
}

// executeQuery runs a GraphQL query against the beans core.
func executeQuery(query string, variables map[string]any, operationName string) ([]byte, error) {
	es := graph.NewExecutableSchema(graph.Config{
		Resolvers: &graph.Resolver{Core: core},
	})

	exec := executor.New(es)

	ctx := graphql.StartOperationTrace(context.Background())
	params := &graphql.RawParams{
		Query:         query,
		Variables:     variables,
		OperationName: operationName,
	}

	opCtx, errs := exec.CreateOperationContext(ctx, params)
	if errs != nil {
		return json.Marshal(graphql.Response{Errors: errs})
	}

	ctx = graphql.WithOperationContext(ctx, opCtx)
	handler, ctx := exec.DispatchOperation(ctx, opCtx)
	resp := handler(ctx)

	return json.Marshal(resp)
}

// prettyPrint outputs the JSON with colors and indentation.
func prettyPrint(data []byte) {
	var response struct {
		Data   json.RawMessage `json:"data"`
		Errors []struct {
			Message string `json:"message"`
			Path    []any  `json:"path,omitempty"`
		} `json:"errors,omitempty"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		// Fallback to raw output
		fmt.Println(string(data))
		return
	}

	// Print errors if any
	if len(response.Errors) > 0 {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
		for _, e := range response.Errors {
			pathStr := ""
			if len(e.Path) > 0 {
				parts := make([]string, len(e.Path))
				for i, p := range e.Path {
					parts[i] = fmt.Sprint(p)
				}
				pathStr = " at " + strings.Join(parts, ".")
			}
			fmt.Println(errorStyle.Render("Error" + pathStr + ": " + e.Message))
		}
		if response.Data == nil {
			return
		}
		fmt.Println()
	}

	// Pretty-print data
	if response.Data != nil {
		var pretty bytes.Buffer
		if err := json.Indent(&pretty, response.Data, "", "  "); err != nil {
			fmt.Println(string(response.Data))
			return
		}
		fmt.Println(pretty.String())
	}
}

// printSchema outputs the GraphQL schema.
func printSchema() error {
	fmt.Print(GetGraphQLSchema())
	return nil
}

// GetGraphQLSchema returns the GraphQL schema as a string.
// This is exported so it can be used by other commands like prompt.
func GetGraphQLSchema() string {
	es := graph.NewExecutableSchema(graph.Config{
		Resolvers: &graph.Resolver{Core: core},
	})

	var buf bytes.Buffer
	f := formatter.NewFormatter(&buf, formatter.WithIndent("  "))
	f.FormatSchema(es.Schema())

	return buf.String()
}

func init() {
	queryCmd.Flags().BoolVar(&queryJSON, "json", false, "Output raw JSON (no formatting)")
	queryCmd.Flags().StringVarP(&queryVariables, "variables", "v", "", "Query variables as JSON string")
	queryCmd.Flags().StringVarP(&queryOperation, "operation", "o", "", "Operation name (for multi-operation documents)")
	queryCmd.Flags().BoolVar(&querySchemaOnly, "schema", false, "Print the GraphQL schema and exit")
	rootCmd.AddCommand(queryCmd)
}
