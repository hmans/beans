package commands

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"

	"github.com/hmans/beans/internal/agent"
	"github.com/hmans/beans/internal/cors"
	"github.com/hmans/beans/internal/graph"
	"github.com/hmans/beans/internal/web"
	"github.com/hmans/beans/internal/worktree"
	"github.com/hmans/beans/pkg/config"
)

var (
	servePort    int
	corsOrigins  []string
)

const centralAgentPrompt = `You are the planning agent for this project. Your primary role is to help manage and organize work through beans (issues).

Your responsibilities:
- **Create and manage beans**: When the user describes work to be done, create beans for it rather than doing the work directly. Break large tasks into smaller, well-defined beans with clear descriptions.
- **Organize work**: Help prioritize, categorize, and structure beans. Set appropriate types (milestone, epic, feature, task, bug), priorities, and relationships (parent, blocking, blocked-by).
- **Start work on beans**: When the user wants to begin working on a specific bean, use the GraphQL startWork mutation to create a worktree for it: mutation { startWork(beanId: "<id>") { path } }
- **Nudge towards beans**: If the user asks you to implement something directly, suggest creating a bean for it instead. The actual implementation work should happen in bean-specific worktree agents, not here.
- **Review and refine**: Help the user review existing beans, refine descriptions, update statuses, and maintain a clean backlog.

You have access to the beans CLI and can use GraphQL queries to inspect and modify beans. Focus on planning and coordination — leave implementation to the worktree agents.`

var serveCmd = &cobra.Command{
	Use:     "serve",
	Aliases: []string{"server", "s"},
	Short:   "Start the web server",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Determine the port: CLI flag > config > default
		port := servePort
		if !cmd.Flags().Changed("port") {
			port = cfg.GetServerPort()
		}

		// Determine CORS origins: CLI flag > config > default
		origins := corsOrigins
		if !cmd.Flags().Changed("cors-origin") {
			origins = cfg.GetCORSOrigins()
		}

		return runServer(port, origins)
	},
}

func runServer(port int, origins []string) error {
	// Start file watcher for subscriptions
	if err := core.StartWatching(); err != nil {
		return fmt.Errorf("failed to start file watcher: %w", err)
	}
	defer core.Unwatch()

	// Set up origin checker for CORS and WebSocket
	checker := cors.NewChecker(origins)

	// Set Gin to release mode for cleaner output
	gin.SetMode(gin.ReleaseMode)

	// Create Gin router
	router := gin.New()

	// Add middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// CORS middleware
	router.Use(func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if allowed := checker.CORSOrigin(origin); allowed != "" {
			c.Header("Access-Control-Allow-Origin", allowed)
			if allowed != "*" {
				c.Header("Vary", "Origin")
			}
		}
		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Create worktree manager (worktrees stored inside .beans/worktrees/)
	wtManager := worktree.NewManager(cfg.ConfigDir(), core.Root(), cfg.GetWorktreeBaseRef())

	// Create agent session manager (with conversation persistence)
	agentMgr := agent.NewManager(core.Root(), func(beanID string) string {
		// Central/planning agent gets a planning-focused prompt
		if beanID == graph.CentralSessionID {
			return centralAgentPrompt
		}

		// Bean-specific agents get context about the bean they're working on
		b, err := core.Get(beanID)
		if err != nil {
			return ""
		}
		var sb strings.Builder
		fmt.Fprintf(&sb, "You are working on bean %s: %q\n", b.ID, b.Title)
		fmt.Fprintf(&sb, "Type: %s | Status: %s", b.Type, b.Status)
		if b.Priority != "" {
			fmt.Fprintf(&sb, " | Priority: %s", b.Priority)
		}
		sb.WriteString("\n")
		if b.Body != "" {
			fmt.Fprintf(&sb, "\nDescription:\n%s", b.Body)
		}
		return sb.String()
	}, agent.DefaultMode(cfg.GetDefaultMode()))
	defer agentMgr.Shutdown()

	// Create GraphQL server with explicit transports
	es := graph.NewExecutableSchema(graph.Config{
		Resolvers: &graph.Resolver{
			Core:        core,
			WorktreeMgr: wtManager,
			AgentMgr:    agentMgr,
			ProjectRoot: filepath.Dir(core.Root()),
		},
	})
	gqlHandler := handler.New(es)

	// Add transports in order (WebSocket first for upgrade handling)
	gqlHandler.AddTransport(transport.Websocket{
		KeepAlivePingInterval: 10 * time.Second,
		Upgrader: websocket.Upgrader{
			CheckOrigin:  checker.CheckOriginFunc(),
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
		fmt.Printf("[beans] Allowed origins: %s\n", strings.Join(origins, ", "))
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

func RegisterServeCmd(root *cobra.Command) {
	serveCmd.Flags().IntVarP(&servePort, "port", "p", config.DefaultServerPort, "Port to listen on")
	serveCmd.Flags().StringSliceVar(&corsOrigins, "cors-origin", cors.DefaultOrigins, "Allowed CORS origins (use * to allow all)")
	root.AddCommand(serveCmd)
}

