package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

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

	// Create PTY session
	cols, rows := initMsg.Cols, initMsg.Rows
	if cols == 0 {
		cols = 80
	}
	if rows == 0 {
		rows = 24
	}

	sess, err := termMgr.Create(initMsg.SessionID, workDir, cols, rows)
	if err != nil {
		_ = conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseInternalServerErr, "failed to create PTY"))
		return
	}

	// When this handler exits, clean up the session
	defer termMgr.Close(initMsg.SessionID)

	// PTY → WebSocket (binary frames)
	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 4096)
		for {
			n, err := sess.Read(buf)
			if n > 0 {
				if writeErr := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); writeErr != nil {
					return
				}
			}
			if err != nil {
				if err != io.EOF {
					// Shell exited — send close frame
					_ = conn.WriteMessage(websocket.CloseMessage,
						websocket.FormatCloseMessage(websocket.CloseNormalClosure, "shell exited"))
				}
				return
			}
		}
	}()

	// WebSocket → PTY (JSON text frames)
	go func() {
		for {
			_, raw, err := conn.ReadMessage()
			if err != nil {
				// Client disconnected
				sess.Close()
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

	// Wait for PTY to close
	<-done
}

// resolveTerminalWorkDir maps a session ID to a filesystem path.
// "__central__" maps to the project root; other IDs are looked up as worktree bean IDs.
func resolveTerminalWorkDir(sessionID string, wtMgr *worktree.Manager, projectRoot string) (string, error) {
	if sessionID == "__central__" {
		return projectRoot, nil
	}

	worktrees, err := wtMgr.List()
	if err != nil {
		return "", fmt.Errorf("failed to list worktrees: %w", err)
	}

	for _, wt := range worktrees {
		if wt.ID == sessionID {
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
