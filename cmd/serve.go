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
	"github.com/hmans/beans/internal/web"
)

var (
	servePort int
)

var serveCmd = &cobra.Command{
	Use:     "serve",
	Aliases: []string{"s"},
	Short:   "Start the web server",
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

	// GraphQL API endpoint with CORS support
	mux.Handle("/api/graphql", corsMiddleware(srv))

	// GraphQL Playground
	mux.Handle("/playground", playground.Handler("Beans GraphQL", "/api/graphql"))

	// Serve the embedded frontend SPA
	mux.Handle("/", web.Handler())

	// Create HTTP server
	addr := fmt.Sprintf(":%d", servePort)
	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Set up signal handling with context
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Channel to listen for server errors
	serverErr := make(chan error, 1)

	// Start server in goroutine
	go func() {
		fmt.Printf("Starting server at http://localhost:%d/\n", servePort)
		fmt.Printf("GraphQL Playground: http://localhost:%d/playground\n", servePort)
		serverErr <- server.ListenAndServe()
	}()

	// Wait for shutdown signal or server error
	select {
	case err := <-serverErr:
		if err != http.ErrServerClosed {
			return fmt.Errorf("server error: %w", err)
		}
	case <-ctx.Done():
		fmt.Printf("\nShutting down...\n")

		// Create context with timeout for graceful shutdown
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("graceful shutdown failed: %w", err)
		}
		fmt.Println("Server stopped")
	}

	return nil
}

// corsMiddleware adds CORS headers for cross-origin requests (e.g., Vite dev server)
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow requests from any origin during development
		// w.Header().Set("Access-Control-Allow-Origin", "*")
		// w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		// w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func init() {
	serveCmd.Flags().IntVarP(&servePort, "port", "p", 22880, "Port to listen on")
	rootCmd.AddCommand(serveCmd)
}
