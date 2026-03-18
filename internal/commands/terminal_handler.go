package commands

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/hmans/beans/internal/terminal"
	"github.com/hmans/beans/internal/worktree"
)

// terminalInitMsg is the initial message sent by the client to start a PTY session.
type terminalInitMsg struct {
	Type      string `json:"type"` // "init"
	SessionID string `json:"sessionId"`
	Cols      uint16 `json:"cols"`
	Rows      uint16 `json:"rows"`
}

// terminalInputMsg is sent by the client to write to the PTY.
type terminalInputMsg struct {
	Type string `json:"type"` // "input"
	Data string `json:"data"`
}

// terminalResizeMsg is sent by the client to resize the PTY.
type terminalResizeMsg struct {
	Type string `json:"type"` // "resize"
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
}

// handleTerminalWS upgrades an HTTP connection to a WebSocket and bridges it to a PTY session.
// Sessions persist across WebSocket reconnections — the PTY stays alive when the client disconnects,
// and scrollback is replayed when a new client attaches.
func handleTerminalWS(c *gin.Context, termMgr *terminal.Manager, wtMgr *worktree.Manager, upgrader websocket.Upgrader, projectRoot string) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	// Read the init message
	_, raw, err := conn.ReadMessage()
	if err != nil {
		return
	}

	var initMsg terminalInitMsg
	if err := json.Unmarshal(raw, &initMsg); err != nil || initMsg.Type != "init" {
		_ = conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseProtocolError, "expected init message"))
		return
	}

	// Resolve working directory
	workDir, err := resolveTerminalWorkDir(initMsg.SessionID, wtMgr, projectRoot)
	if err != nil {
		_ = conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseProtocolError, err.Error()))
		return
	}

	// Get existing session or create new one
	cols, rows := initMsg.Cols, initMsg.Rows
	if cols == 0 {
		cols = 80
	}
	if rows == 0 {
		rows = 24
	}

	sess, reconnected, err := termMgr.GetOrCreate(initMsg.SessionID, workDir, cols, rows)
	if err != nil {
		_ = conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseInternalServerErr, "failed to create PTY"))
		return
	}

	// Attach to session — get scrollback and live output channel
	scrollback, output := sess.Attach()
	defer sess.Detach(output)

	// Replay scrollback for reconnecting clients
	if reconnected && len(scrollback) > 0 {
		if err := conn.WriteMessage(websocket.BinaryMessage, scrollback); err != nil {
			return
		}
	}

	// Channel closed when the WebSocket read loop exits (client disconnected)
	clientDone := make(chan struct{})

	// PTY output → WebSocket (reads from the output channel populated by the session's readLoop)
	ptyDone := make(chan struct{})
	go func() {
		defer close(ptyDone)
		for {
			select {
			case data := <-output:
				if err := conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
					return
				}
			case <-sess.Done():
				// Shell exited — drain any remaining buffered output
				for {
					select {
					case data := <-output:
						_ = conn.WriteMessage(websocket.BinaryMessage, data)
					default:
						_ = conn.WriteMessage(websocket.CloseMessage,
							websocket.FormatCloseMessage(websocket.CloseNormalClosure, "shell exited"))
						return
					}
				}
			case <-clientDone:
				return
			}
		}
	}()

	// WebSocket → PTY (JSON text frames)
	go func() {
		defer close(clientDone)
		for {
			_, raw, err := conn.ReadMessage()
			if err != nil {
				return
			}

			var baseMsg struct {
				Type string `json:"type"`
			}
			if err := json.Unmarshal(raw, &baseMsg); err != nil {
				continue
			}

			switch baseMsg.Type {
			case "input":
				var msg terminalInputMsg
				if err := json.Unmarshal(raw, &msg); err == nil {
					_, _ = sess.Write([]byte(msg.Data))
				}
			case "resize":
				var msg terminalResizeMsg
				if err := json.Unmarshal(raw, &msg); err == nil {
					_ = sess.Resize(msg.Cols, msg.Rows)
				}
			}
		}
	}()

	// Wait for either PTY exit or client disconnect
	select {
	case <-ptyDone:
	case <-clientDone:
	}
}

// resolveTerminalWorkDir maps a session ID to a filesystem path.
// "__central__" maps to the project root; other IDs are looked up as worktree bean IDs.
// The "__run" suffix is stripped before lookup, so run sessions resolve to the same
// directory as their parent workspace.
func resolveTerminalWorkDir(sessionID string, wtMgr *worktree.Manager, projectRoot string) (string, error) {
	// Strip __run suffix so run sessions resolve to the same workspace directory
	baseID := strings.TrimSuffix(sessionID, "__run")

	if baseID == "__central__" {
		return projectRoot, nil
	}

	worktrees, err := wtMgr.List()
	if err != nil {
		return "", fmt.Errorf("failed to list worktrees: %w", err)
	}

	for _, wt := range worktrees {
		if wt.ID == baseID {
			return wt.Path, nil
		}
	}

	return "", fmt.Errorf("unknown session: %s", sessionID)
}

// RegisterTerminalRoute adds the /api/terminal WebSocket endpoint to the Gin router.
func RegisterTerminalRoute(router *gin.Engine, termMgr *terminal.Manager, wtMgr *worktree.Manager, checkOrigin func(r *http.Request) bool, projectRoot string) {
	upgrader := websocket.Upgrader{
		CheckOrigin: checkOrigin,
	}

	router.GET("/api/terminal", func(c *gin.Context) {
		handleTerminalWS(c, termMgr, wtMgr, upgrader, projectRoot)
	})
}
