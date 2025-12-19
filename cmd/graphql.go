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
	"github.com/spf13/cobra"
	"github.com/tidwall/pretty"
	"github.com/vektah/gqlparser/v2/formatter"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"github.com/hmans/beans/internal/graph"
)

var (
	queryJSON       bool
	queryVariables  string
	queryOperation  string
	querySchemaOnly bool
)

var graphqlCmd = &cobra.Command{
	Use:     "graphql <query>",
	Aliases: []string{"query"},
	Short:   "Execute a GraphQL query or mutation",
	Long: `Execute a GraphQL query or mutation against the beans data.

The argument should be a valid GraphQL query or mutation string.

Examples:
  # List all beans
  beans graphql '{ beans { id title status } }'

  # Get a specific bean
  beans graphql '{ bean(id: "abc") { title status body } }'

  # Filter beans by status
  beans graphql '{ beans(filter: { status: ["todo", "in-progress"] }) { id title } }'

  # Get beans with relationships
  beans graphql '{ beans { id title blockedBy { id title } children { id title } } }'

  # Use variables
  beans graphql -v '{"id": "abc"}' 'query GetBean($id: ID!) { bean(id: $id) { title } }'

  # Read from stdin (useful for complex queries or escaping issues)
  echo '{ beans { id title } }' | beans graphql
  cat query.graphql | beans graphql

  # Print the schema
  beans graphql --schema`,
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
// On success, it returns just the data portion of the response.
// On error, it returns an error so the CLI can handle it appropriately.
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
		return nil, formatGraphQLErrors(errs)
	}

	ctx = graphql.WithOperationContext(ctx, opCtx)
	handler, ctx := exec.DispatchOperation(ctx, opCtx)
	resp := handler(ctx)

	if len(resp.Errors) > 0 {
		return nil, formatGraphQLErrors(resp.Errors)
	}

	return resp.Data, nil
}

// formatGraphQLErrors formats GraphQL errors into a single error.
func formatGraphQLErrors(errs gqlerror.List) error {
	if len(errs) == 0 {
		return nil
	}
	if len(errs) == 1 {
		return fmt.Errorf("graphql: %s", errs[0].Message)
	}
	var msgs []string
	for _, e := range errs {
		msgs = append(msgs, e.Message)
	}
	return fmt.Errorf("graphql errors:\n  %s", strings.Join(msgs, "\n  "))
}

// prettyPrint outputs the JSON with colors and indentation.
func prettyPrint(data []byte) {
	fmt.Println(string(pretty.Color(pretty.Pretty(data), nil)))
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
	graphqlCmd.Flags().BoolVar(&queryJSON, "json", false, "Output raw JSON (no formatting)")
	graphqlCmd.Flags().StringVarP(&queryVariables, "variables", "v", "", "Query variables as JSON string")
	graphqlCmd.Flags().StringVarP(&queryOperation, "operation", "o", "", "Operation name (for multi-operation documents)")
	graphqlCmd.Flags().BoolVar(&querySchemaOnly, "schema", false, "Print the GraphQL schema and exit")
	rootCmd.AddCommand(graphqlCmd)
}
