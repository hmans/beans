package agent

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// store handles JSONL persistence for agent conversations.
// Each bean gets a file at <beansDir>/.conversations/<beanID>.jsonl.
type store struct {
	dir string // .beans/.conversations/
}

// entry is a single line in the JSONL file.
type entry struct {
	Type      string `json:"type"`                // "message" or "meta"
	Role      string `json:"role,omitempty"`       // for messages: "user" or "assistant"
	Content   string `json:"content,omitempty"`    // for messages
	SessionID string `json:"session_id,omitempty"` // for meta
}

// newStore creates the conversations directory and .gitignore if needed.
func newStore(beansDir string) (*store, error) {
	dir := filepath.Join(beansDir, ".conversations")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create conversations dir: %w", err)
	}

	// Write .gitignore to exclude all conversation files
	gitignore := filepath.Join(dir, ".gitignore")
	if _, err := os.Stat(gitignore); os.IsNotExist(err) {
		if err := os.WriteFile(gitignore, []byte("*\n!.gitignore\n"), 0o644); err != nil {
			return nil, fmt.Errorf("create .gitignore: %w", err)
		}
	}

	return &store{dir: dir}, nil
}

// load reads the JSONL file for a bean and returns the messages and session ID.
func (s *store) load(beanID string) ([]Message, string, error) {
	path := s.path(beanID)
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", fmt.Errorf("open conversation file: %w", err)
	}
	defer f.Close()

	var messages []Message
	var sessionID string

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 256*1024), 256*1024)
	for scanner.Scan() {
		var e entry
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			continue // skip malformed lines
		}
		switch e.Type {
		case "message":
			messages = append(messages, Message{
				Role:    MessageRole(e.Role),
				Content: e.Content,
			})
		case "meta":
			if e.SessionID != "" {
				sessionID = e.SessionID
			}
		}
	}

	return messages, sessionID, scanner.Err()
}

// appendMessage appends a message entry to the JSONL file.
func (s *store) appendMessage(beanID string, msg Message) error {
	return s.appendEntry(beanID, entry{
		Type:    "message",
		Role:    string(msg.Role),
		Content: msg.Content,
	})
}

// saveSessionID appends a meta entry with the session ID.
func (s *store) saveSessionID(beanID, sessionID string) error {
	return s.appendEntry(beanID, entry{
		Type:      "meta",
		SessionID: sessionID,
	})
}

// appendEntry appends a single JSON line to the JSONL file.
func (s *store) appendEntry(beanID string, e entry) error {
	data, err := json.Marshal(e)
	if err != nil {
		return err
	}
	data = append(data, '\n')

	f, err := os.OpenFile(s.path(beanID), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open conversation file for append: %w", err)
	}
	defer f.Close()

	_, err = f.Write(data)
	return err
}

// clear deletes the JSONL file for a bean, removing all persisted conversation history.
func (s *store) clear(beanID string) error {
	err := os.Remove(s.path(beanID))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// path returns the JSONL file path for a bean.
func (s *store) path(beanID string) string {
	return filepath.Join(s.dir, beanID+".jsonl")
}
