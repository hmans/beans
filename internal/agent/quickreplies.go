package agent

import (
	"context"
	"log"
	"os/exec"
	"strings"
	"time"
)

const quickRepliesPrompt = `You are given the last message from an AI coding assistant, along with the current workspace status. Suggest 3-4 short replies (under 10 words each) that the user might want to send next.

Consider the workspace status when making suggestions. For example:
- If there are uncommitted changes, suggest committing
- If there are unpushed commits and no PR exists, suggest creating a PR
- If a PR exists with passing checks, suggest merging
- If there are conflicts with the base branch, suggest rebasing

Focus on the most likely next actions: approving work, committing, creating/updating PRs, asking for changes, requesting more detail, etc.

Output one reply per line, nothing else. No numbering, no bullets, no quotes.

Examples of good replies:
Yes, implement this
Commit these changes
Create a PR
Show me the code first
What about error handling?
Let's skip this for now`

// GenerateQuickReplies runs a lightweight Claude Haiku call to suggest
// follow-up replies based on the last assistant message and workspace context.
// workspaceContext is optional — pass "" to omit it.
// Returns a slice of suggestion strings, or nil on error.
func GenerateQuickReplies(message string, workspaceContext string) []string {
	var sb strings.Builder
	sb.WriteString(quickRepliesPrompt)
	if workspaceContext != "" {
		sb.WriteString("\n\nWorkspace status:\n")
		sb.WriteString(workspaceContext)
	}
	sb.WriteString("\n\nAssistant message:\n\n")
	sb.WriteString(truncate(message, 2000))
	prompt := sb.String()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "claude",
		"--print", "--model", "haiku",
		"--no-session-persistence",
		"--disable-slash-commands",
		"--strict-mcp-config", "--mcp-config", `{"mcpServers":{}}`,
	)
	cmd.Env = buildClaudeEnv()
	cmd.Stdin = strings.NewReader(prompt)

	out, err := cmd.Output()
	if err != nil {
		log.Printf("[quickreplies] failed to generate quick replies: %v", err)
		return nil
	}

	return parseQuickReplies(string(out))
}

// generateQuickReplies extracts the last assistant message from the session
// and asynchronously generates quick reply suggestions. Updates the session
// and notifies subscribers when done.
func (m *Manager) generateQuickReplies(beanID string) {
	// Extract the last assistant message
	m.mu.RLock()
	s, ok := m.sessions[beanID]
	if !ok {
		m.mu.RUnlock()
		return
	}
	var lastAssistant string
	for i := len(s.Messages) - 1; i >= 0; i-- {
		if s.Messages[i].Role == RoleAssistant {
			lastAssistant = s.Messages[i].Content
			break
		}
	}
	m.mu.RUnlock()

	if lastAssistant == "" {
		return
	}

	var wsContext string
	if m.quickReplyContext != nil {
		wsContext = m.quickReplyContext(beanID)
	}

	replies := GenerateQuickReplies(lastAssistant, wsContext)
	if len(replies) == 0 {
		return
	}

	m.mu.Lock()
	s, ok = m.sessions[beanID]
	if ok {
		// Only set if the session is still idle (user hasn't started a new turn)
		if s.Status == StatusIdle {
			s.QuickReplies = replies
		}
	}
	m.mu.Unlock()

	if ok {
		m.notify(beanID)
	}
}

// parseQuickReplies splits the raw output into individual reply strings,
// filtering out empty lines and common formatting artifacts.
func parseQuickReplies(raw string) []string {
	lines := strings.Split(strings.TrimSpace(raw), "\n")
	var replies []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Strip common formatting: bullets, numbering, quotes
		line = strings.TrimLeft(line, "-•*0123456789.) ")
		line = strings.Trim(line, "\"'")
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		replies = append(replies, line)
	}
	// Cap at 4 replies
	if len(replies) > 4 {
		replies = replies[:4]
	}
	return replies
}
