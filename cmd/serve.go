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
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"

	"github.com/hmans/beans/internal/config"
	"github.com/hmans/beans/internal/graph"
	"github.com/hmans/beans/internal/web"
)

var (
	servePort int
)

var serveCmd = &cobra.Command{
	Use:     "serve",
	Aliases: []string{"server", "s"},
	Short:   "Start the web server",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Determine the port: CLI flag > config > default
		port := servePort
		if !cmd.Flags().Changed("port") {
			// Flag not explicitly set, use config value
			port = cfg.GetServerPort()
		}
		return runServer(port)
	},
}

func runServer(port int) error {
	// Start file watcher for subscriptions
	if err := core.StartWatching(); err != nil {
		return fmt.Errorf("failed to start file watcher: %w", err)
	}
	defer core.Unwatch()

	// Set Gin to release mode for cleaner output
	gin.SetMode(gin.ReleaseMode)

	// Create Gin router
	router := gin.New()

	// Add middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// CORS middleware for development
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Create GraphQL server with explicit transports
	es := graph.NewExecutableSchema(graph.Config{
		Resolvers: &graph.Resolver{Core: core},
	})
	gqlHandler := handler.New(es)

	// Add transports in order (WebSocket first for upgrade handling)
	gqlHandler.AddTransport(transport.Websocket{
		KeepAlivePingInterval: 10 * time.Second,
		Upgrader: websocket.Upgrader{
			CheckOrigin:  func(r *http.Request) bool { return true },
			Subprotocols: []string{"graphql-transport-ws"},
		},
	})
	gqlHandler.AddTransport(transport.Options{})
	gqlHandler.AddTransport(transport.GET{})
	gqlHandler.AddTransport(transport.POST{})

	// GraphQL API endpoint (handle all methods for WebSocket upgrade)
	router.Any("/api/graphql", gin.WrapH(gqlHandler))

	// GraphQL Playground
	router.GET("/playground", gin.WrapH(playground.Handler("Beans GraphQL", "/api/graphql")))

	// Serve the embedded frontend SPA
	router.NoRoute(gin.WrapH(web.Handler()))

	// Create HTTP server
	addr := fmt.Sprintf(":%d", port)
	server := &http.Server{
		Addr:         addr,
		Handler:      router,
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
		fmt.Printf("[beans] Starting server at http://localhost:%d/\n", port)
		fmt.Printf("[beans] GraphQL Playground: http://localhost:%d/playground\n", port)
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
			fmt.Printf("Graceful shutdown timed out: %v\n", err)
			fmt.Println("Forcing exit...")
			server.Close() // Force close all connections
		}
		fmt.Println("Server stopped")
	}

	return nil
}

func init() {
	serveCmd.Flags().IntVarP(&servePort, "port", "p", config.DefaultServerPort, "Port to listen on")
	rootCmd.AddCommand(serveCmd)
}
