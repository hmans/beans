package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/spf13/cobra"

	"github.com/hmans/beans/internal/graph"
)

var (
	servePort int
)

var serveCmd = &cobra.Command{
	Use:     "serve",
	Aliases: []string{"s"},
	Short:   "Start the web server",
	Long: `Start an HTTP server that serves the GraphQL API.

The server exposes:
  - GraphQL endpoint at /graphql (POST)
  - GraphQL Playground at /graphql (GET) for interactive queries

Examples:
  # Start server on default port 22880
  beans serve

  # Start server on a custom port
  beans serve --port 3000`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runServer()
	},
}

func runServer() error {
	// Create GraphQL server
	es := graph.NewExecutableSchema(graph.Config{
		Resolvers: &graph.Resolver{Core: core},
	})
	srv := handler.NewDefaultServer(es)

	// Set up routes
	mux := http.NewServeMux()

	// GraphQL endpoint - serves both the API and playground
	mux.Handle("/graphql", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Serve playground on GET requests
		if r.Method == http.MethodGet {
			playground.Handler("Beans GraphQL", "/graphql").ServeHTTP(w, r)
			return
		}
		// Handle GraphQL requests
		srv.ServeHTTP(w, r)
	}))

	// Create HTTP server
	addr := fmt.Sprintf(":%d", servePort)
	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Channel to listen for shutdown signals
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Channel to listen for server errors
	serverErr := make(chan error, 1)

	// Start server in goroutine
	go func() {
		fmt.Printf("Starting server at http://localhost:%d\n", servePort)
		fmt.Printf("GraphQL Playground: http://localhost:%d/graphql\n", servePort)
		serverErr <- server.ListenAndServe()
	}()

	// Wait for shutdown signal or server error
	select {
	case err := <-serverErr:
		if err != http.ErrServerClosed {
			return fmt.Errorf("server error: %w", err)
		}
	case sig := <-shutdown:
		fmt.Printf("\nReceived %v, shutting down...\n", sig)

		// Create context with timeout for graceful shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			return fmt.Errorf("graceful shutdown failed: %w", err)
		}
		fmt.Println("Server stopped")
	}

	return nil
}

func init() {
	serveCmd.Flags().IntVarP(&servePort, "port", "p", 22880, "Port to listen on")
	rootCmd.AddCommand(serveCmd)
}
