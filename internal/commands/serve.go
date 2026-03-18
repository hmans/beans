package commands

import (
	"context"
	"fmt"
	"log"
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
	"github.com/hmans/beans/internal/portalloc"
	"github.com/hmans/beans/internal/terminal"
	"github.com/hmans/beans/internal/web"
	"github.com/hmans/beans/internal/worktree"
	"github.com/hmans/beans/pkg/config"
	"github.com/hmans/beans/pkg/forge"
)

var (
	servePort    int
	corsOrigins  []string
)

const centralAgentPrompt = `You are the planning agent for this project. Your primary role is to help manage and organize work through beans (issues).

IMPORTANT: Do NOT use Claude Code's built-in worktree system (EnterWorktree tool). This project has its own worktree management. To start work on a bean, use the GraphQL startWork mutation instead: mutation { startWork(beanId: "<id>") { path } }

CRITICAL — ASKING QUESTIONS:
You are running inside a web UI, NOT an interactive terminal. The user CANNOT see or respond to plain-text questions in your output. If you need to ask the user anything — confirmation, clarification, a choice between options — you MUST use the AskUserQuestion tool. Every single time. No exceptions. Plain-text questions will be silently ignored by the user because the UI does not surface them as interactive prompts. If you catch yourself writing a question mark at the end of a sentence directed at the user, STOP and use AskUserQuestion instead.

Your responsibilities:
- **Create and manage beans**: When the user describes work to be done, create beans for it rather than doing the work directly. Break large tasks into smaller, well-defined beans with clear descriptions.
- **Organize work**: Help prioritize, categorize, and structure beans. Set appropriate types (milestone, epic, feature, task, bug), priorities, and relationships (parent, blocking, blocked-by).
- **Start work on beans**: When the user wants to begin working on a specific bean, use the startWork GraphQL mutation (see above) to create a worktree for it. NEVER use the EnterWorktree tool.
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
	// Resolve worktree root directory (default: ~/.beans/worktrees/<project>/)
	projectName := cfg.GetProjectName()
	if projectName == "" {
		projectName = filepath.Base(cfg.ConfigDir())
	}
	worktreeRoot, err := cfg.ResolveWorktreePath(projectName)
	if err != nil {
		return fmt.Errorf("failed to resolve worktree path: %w", err)
	}
	if err := os.MkdirAll(worktreeRoot, 0755); err != nil {
		return fmt.Errorf("failed to create worktree directory %s: %w", worktreeRoot, err)
	}
	log.Printf("[beans] worktrees directory: %s", worktreeRoot)

	// Warn if old in-repo worktrees directory exists
	oldWorktreeDir := filepath.Join(core.Root(), ".worktrees")
	if entries, err := os.ReadDir(oldWorktreeDir); err == nil && len(entries) > 0 {
		log.Printf("[beans] WARNING: found old worktrees in %s — worktrees are now created in %s. You may want to recreate existing worktrees and remove the old directory.", oldWorktreeDir, worktreeRoot)
	}

	wtManager := worktree.NewManager(cfg.ConfigDir(), worktreeRoot, cfg.GetWorktreeBaseRef(), cfg.GetWorktreeSetup())

	// Watch existing worktrees for bean changes
	if existingWTs, err := wtManager.List(); err == nil {
		for _, wt := range existingWTs {
			if err := core.WatchWorktreeBeans(wt.Path); err != nil {
				fmt.Printf("[beans] warning: failed to watch worktree %s: %v\n", wt.ID, err)
			}
		}
	}

	// Create workspace port allocator and allocate port for central workspace
	portAlloc := portalloc.NewDefault()
	portAlloc.Allocate(graph.CentralSessionID)

	// Allocate ports for existing worktrees
	if existingWTs, err := wtManager.List(); err == nil {
		for _, wt := range existingWTs {
			portAlloc.Allocate(wt.ID)
		}
	}

	// Create terminal session manager with workspace port injection.
	// Run sessions use "${workspaceId}__run" as session ID — strip the
	// suffix so they share the same port as the workspace's shell session.
	termMgr := terminal.NewManager(func(sessionID string) []string {
		workspaceID := strings.TrimSuffix(sessionID, "__run")
		port, err := portAlloc.Get(workspaceID)
		if err != nil {
			return nil
		}
		return []string{fmt.Sprintf("BEANS_WORKSPACE_PORT=%d", port)}
	})
	defer termMgr.Shutdown()

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
		sb.WriteString("IMPORTANT: Do NOT use Claude Code's built-in worktree system (EnterWorktree tool). You are already working inside a beans-managed worktree.\n\n")
		sb.WriteString("CRITICAL — ASKING QUESTIONS:\nYou are running inside a web UI, NOT an interactive terminal. The user CANNOT see or respond to plain-text questions in your output. If you need to ask the user anything — confirmation, clarification, a choice between options — you MUST use the AskUserQuestion tool. Every single time. No exceptions. Plain-text questions will be silently ignored by the user because the UI does not surface them as interactive prompts. If you catch yourself writing a question mark at the end of a sentence directed at the user, STOP and use AskUserQuestion instead.\n\n")
		sb.WriteString("IMPORTANT: You MUST only create or modify files within this worktree directory. NEVER make changes to files in the main repository or any other worktree. All your file operations (reads are fine anywhere, but writes, edits, and deletions) must be scoped to your current working directory.\n\n")
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
	if effort := cfg.GetDefaultEffort(); config.IsValidEffortLevel(effort) {
		agentMgr.SetDefaultEffort(agent.EffortLevel(effort))
	}
	defer agentMgr.Shutdown()

	// Inject a system prompt that tells the agent which worktree/directory it's in.
	// This is separate from context (which goes in the first user message) because
	// the system prompt persists across --resume, ensuring the agent always knows
	// its workspace identity.
	agentMgr.SetSystemPromptProvider(func(beanID string) string {
		if beanID == graph.CentralSessionID {
			return fmt.Sprintf("You are the central planning agent for the beans project. Your working directory is the main repository at: %s", filepath.Dir(core.Root()))
		}
		wtPath := wtManager.WorktreePath(beanID)
		if wtPath == "" {
			return ""
		}
		return fmt.Sprintf("You are a workspace agent working in a git worktree. Your worktree ID is %q and your working directory is: %s\nAll file modifications MUST be within this directory. NEVER modify files in the main repository or other worktrees.\nNEVER force-push to main. NEVER push to origin/main. The Integrate action is local-only.", beanID, wtPath)
	})

	// Post an info message to the workspace's agent chat when setup finishes.
	wtManager.SetOnSetupDone(func(worktreeID string, success bool, output string) {
		if success {
			agentMgr.AddInfoMessage(worktreeID, "Workspace setup completed successfully.")
		} else {
			agentMgr.AddInfoMessage(worktreeID, fmt.Sprintf("Workspace setup failed:\n```\n%s\n```", output))
		}
	})

	// Auto-generate workspace descriptions when the first user message is sent.
	// Runs a cheap Claude call (Haiku) in the background to summarize what
	// the workspace is doing, then stores it as worktree metadata.
	agentMgr.SetOnFirstUserMessage(func(beanID string, message string) {
		// Only generate for worktree agents, not central
		if beanID == graph.CentralSessionID {
			return
		}
		// Only if this worktree exists and doesn't already have a description
		if wts, err := wtManager.List(); err == nil {
			for _, wt := range wts {
				if wt.ID == beanID && wt.Description == "" {
					desc := agent.GenerateDescription(message)
					if desc != "" {
						if err := wtManager.UpdateDescription(beanID, desc); err != nil {
							log.Printf("[beans] failed to save workspace description for %s: %v", beanID, err)
						} else {
							log.Printf("[beans] generated workspace description for %s: %q", beanID, desc)
						}
					}
					break
				}
			}
		}
	})

	// Update workspace activity timestamp when an agent completes a turn,
	// so the sidebar sorts workspaces by most recently active.
	agentMgr.SetOnTurnComplete(func(beanID string) {
		if beanID == graph.CentralSessionID || wtManager == nil {
			return
		}
		if err := wtManager.TouchLastActive(beanID); err != nil {
			log.Printf("[beans] failed to update last_active_at for %s: %v", beanID, err)
		}
	})

	// When bean files change in a worktree, also notify the worktree manager
	// so the worktree subscription re-emits with updated detected bean IDs.
	if wtManager != nil {
		core.SetOnWorktreeBeansChanged(wtManager.Notify)
	}

	// Detect git forge (GitHub, GitLab, etc.) for PR integration
	projectRoot := filepath.Dir(core.Root())
	forgeProvider := forge.Detect(projectRoot)
	if forgeProvider != nil {
		fmt.Printf("[beans] detected forge: %s (using %s CLI)\n", forgeProvider.Name(), forgeProvider.CLIName())
	}

	// Create GraphQL server with explicit transports
	es := graph.NewExecutableSchema(graph.Config{
		Resolvers: &graph.Resolver{
			Core:        core,
			WorktreeMgr: wtManager,
			AgentMgr:    agentMgr,
			TerminalMgr: termMgr,
			PortAlloc:   portAlloc,
			Forge:       forgeProvider,
			ProjectRoot: projectRoot,
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

	// Serve agent chat image attachments
	router.GET("/api/attachments/:beanId/:filename", func(c *gin.Context) {
		beanID := c.Param("beanId")
		filename := c.Param("filename")
		path, err := agentMgr.AttachmentPath(beanID, filename)
		if err != nil {
			c.Status(http.StatusBadRequest)
			return
		}
		if _, err := os.Stat(path); err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		c.Header("Content-Disposition", "inline")
		c.Header("X-Content-Type-Options", "nosniff")
		c.File(path)
	})

	// Terminal WebSocket endpoint
	RegisterTerminalRoute(router, termMgr, wtManager, checker.CheckOriginFunc(), filepath.Dir(core.Root()))

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

		// Hard deadline: if graceful shutdown takes too long, force exit.
		// This prevents zombie processes when cleanup hangs (e.g. a claude
		// process ignores SIGINT, or a WebSocket handler blocks).
		go func() {
			time.Sleep(10 * time.Second)
			fmt.Fprintf(os.Stderr, "Shutdown deadline exceeded, forcing exit\n")
			os.Exit(1)
		}()

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

