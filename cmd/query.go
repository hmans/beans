package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
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
		var variables map[string]interface{}
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

// graphqlRequest represents a GraphQL request body.
type graphqlRequest struct {
	Query         string                 `json:"query"`
	Variables     map[string]interface{} `json:"variables,omitempty"`
	OperationName string                 `json:"operationName,omitempty"`
}

// executeQuery runs a GraphQL query against the beans core.
func executeQuery(query string, variables map[string]interface{}, operationName string) ([]byte, error) {
	// Create the GraphQL server
	srv := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{
		Resolvers: &graph.Resolver{Core: core},
	}))

	// Build request body
	reqBody := graphqlRequest{
		Query:         query,
		Variables:     variables,
		OperationName: operationName,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	// Create HTTP request
	req := httptest.NewRequest(http.MethodPost, "/graphql", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	// Execute via httptest
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	// Read response
	resp := rec.Result()
	defer resp.Body.Close()

	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	return result, nil
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
	es := graph.NewExecutableSchema(graph.Config{
		Resolvers: &graph.Resolver{Core: core},
	})
	schema := es.Schema()

	// Build schema string from the AST
	var sb strings.Builder

	// Print types
	for _, t := range schema.Types {
		// Skip built-in types
		if strings.HasPrefix(t.Name, "__") {
			continue
		}
		// Skip built-in scalars
		if t.BuiltIn {
			continue
		}

		switch t.Kind {
		case "SCALAR":
			sb.WriteString(fmt.Sprintf("scalar %s\n\n", t.Name))
		case "ENUM":
			sb.WriteString(fmt.Sprintf("enum %s {\n", t.Name))
			for _, v := range t.EnumValues {
				sb.WriteString(fmt.Sprintf("  %s\n", v.Name))
			}
			sb.WriteString("}\n\n")
		case "INPUT_OBJECT":
			sb.WriteString(fmt.Sprintf("input %s {\n", t.Name))
			for _, f := range t.Fields {
				sb.WriteString(fmt.Sprintf("  %s: %s\n", f.Name, f.Type.String()))
			}
			sb.WriteString("}\n\n")
		case "OBJECT":
			sb.WriteString(fmt.Sprintf("type %s {\n", t.Name))
			for _, f := range t.Fields {
				// Add description as comment if present
				if f.Description != "" {
					sb.WriteString(fmt.Sprintf("  # %s\n", f.Description))
				}
				if len(f.Arguments) > 0 {
					args := make([]string, len(f.Arguments))
					for i, a := range f.Arguments {
						args[i] = fmt.Sprintf("%s: %s", a.Name, a.Type.String())
					}
					sb.WriteString(fmt.Sprintf("  %s(%s): %s\n", f.Name, strings.Join(args, ", "), f.Type.String()))
				} else {
					sb.WriteString(fmt.Sprintf("  %s: %s\n", f.Name, f.Type.String()))
				}
			}
			sb.WriteString("}\n\n")
		}
	}

	fmt.Print(sb.String())
	return nil
}

func init() {
	queryCmd.Flags().BoolVar(&queryJSON, "json", false, "Output raw JSON (no formatting)")
	queryCmd.Flags().StringVarP(&queryVariables, "variables", "v", "", "Query variables as JSON string")
	queryCmd.Flags().StringVarP(&queryOperation, "operation", "o", "", "Operation name (for multi-operation documents)")
	queryCmd.Flags().BoolVar(&querySchemaOnly, "schema", false, "Print the GraphQL schema and exit")
	rootCmd.AddCommand(queryCmd)
}
