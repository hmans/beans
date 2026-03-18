package agent

import (
	"context"
	"log"
	"os/exec"
	"strings"
	"time"
)

const quickRepliesPrompt = `You are given the last message from an AI coding assistant. Suggest 3-4 short replies (under 10 words each) that the user might want to send next. Focus on the most likely actions: approving work, asking for changes, requesting more detail, etc.

Output one reply per line, nothing else. No numbering, no bullets, no quotes.

Examples of good replies:
Yes, implement this
Show me the code first
What about error handling?
Let's skip this for now

Assistant message:`

// GenerateQuickReplies runs a lightweight Claude Haiku call to suggest
// follow-up replies based on the last assistant message.
// Returns a slice of suggestion strings, or nil on error.
func GenerateQuickReplies(message string) []string {
	prompt := quickRepliesPrompt + "\n\n" + truncate(message, 2000)

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

	replies := GenerateQuickReplies(lastAssistant)
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
