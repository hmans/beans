package agent

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/hmans/beans/pkg/safepath"
)

// maxImageSize is the maximum allowed image size (5 MB, matching Anthropic API limits).
const maxImageSize = 5 * 1024 * 1024

// allowedImageTypes lists the accepted MIME types for image uploads.
var allowedImageTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

// store handles JSONL persistence for agent conversations.
// Each bean gets a file at <beansDir>/.conversations/<beanID>.jsonl.
type store struct {
	dir string // .beans/.conversations/
}

// entryImage is the JSON representation of an image reference in a JSONL entry.
type entryImage struct {
	ID        string `json:"id"`
	MediaType string `json:"media_type"`
}

// entry is a single line in the JSONL file.
type entry struct {
	Type        string       `json:"type"`                  // "message" or "meta"
	Role        string       `json:"role,omitempty"`         // for messages: "user" or "assistant"
	Content     string       `json:"content,omitempty"`      // for messages
	Images      []entryImage `json:"images,omitempty"`       // for messages with image attachments
	Diff        string       `json:"diff,omitempty"`         // for tool messages: unified diff output
	Attachments []string     `json:"attachments,omitempty"`  // file paths from @-mentions
	SessionID   string       `json:"session_id,omitempty"`   // for meta
}

// newStore creates the conversations directory if needed.
func newStore(beansDir string) (*store, error) {
	dir := filepath.Join(beansDir, ".conversations")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create conversations dir: %w", err)
	}

	return &store{dir: dir}, nil
}

// load reads the JSONL file for a bean and returns the messages and session ID.
func (s *store) load(beanID string) ([]Message, string, error) {
	path, err := s.path(beanID)
	if err != nil {
		return nil, "", err
	}
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
			msg := Message{
				Role:        MessageRole(e.Role),
				Content:     e.Content,
				Diff:        e.Diff,
				Attachments: e.Attachments,
			}
			for _, img := range e.Images {
				msg.Images = append(msg.Images, ImageRef{ID: img.ID, MediaType: img.MediaType})
			}
			messages = append(messages, msg)
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
	e := entry{
		Type:        "message",
		Role:        string(msg.Role),
		Content:     msg.Content,
		Diff:        msg.Diff,
		Attachments: msg.Attachments,
	}
	for _, img := range msg.Images {
		e.Images = append(e.Images, entryImage{ID: img.ID, MediaType: img.MediaType})
	}
	return s.appendEntry(beanID, e)
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

	path, err := s.path(beanID)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open conversation file for append: %w", err)
	}
	defer f.Close()

	_, err = f.Write(data)
	return err
}

// clear deletes the JSONL file and all attachments for a bean.
func (s *store) clear(beanID string) error {
	path, err := s.path(beanID)
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return s.clearAttachments(beanID)
}

// path returns the JSONL file path for a bean.
// Returns an error if the beanID would cause path traversal.
func (s *store) path(beanID string) (string, error) {
	if err := safepath.ValidateBeanID(beanID); err != nil {
		return "", fmt.Errorf("invalid bean ID for conversation path: %w", err)
	}
	return safepath.SafeJoin(s.dir, beanID+".jsonl")
}

// attachmentDir returns the directory for a bean's image attachments, creating it if needed.
func (s *store) attachmentDir(beanID string) (string, error) {
	if err := safepath.ValidateBeanID(beanID); err != nil {
		return "", fmt.Errorf("invalid bean ID for attachment dir: %w", err)
	}
	dir, err := safepath.SafeJoin(s.dir, filepath.Join("attachments", beanID))
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create attachment dir: %w", err)
	}
	return dir, nil
}

// saveImage stores an image file to disk and returns a reference to it.
// Validates media type and file size.
func (s *store) saveImage(beanID, mediaType string, data []byte) (ImageRef, error) {
	ext, ok := allowedImageTypes[mediaType]
	if !ok {
		return ImageRef{}, fmt.Errorf("unsupported image type %q (allowed: jpeg, png, gif, webp)", mediaType)
	}
	if len(data) > maxImageSize {
		return ImageRef{}, fmt.Errorf("image too large (%d bytes, max %d)", len(data), maxImageSize)
	}

	dir, err := s.attachmentDir(beanID)
	if err != nil {
		return ImageRef{}, err
	}

	id := uuid.New().String() + ext
	path := filepath.Join(dir, id)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return ImageRef{}, fmt.Errorf("write image file: %w", err)
	}

	return ImageRef{ID: id, MediaType: mediaType}, nil
}

// attachmentPath returns the filesystem path for a stored image.
// Validates both beanID and imageID to prevent path traversal.
func (s *store) attachmentPath(beanID, imageID string) (string, error) {
	if err := safepath.ValidateBeanID(beanID); err != nil {
		return "", fmt.Errorf("invalid bean ID for attachment: %w", err)
	}
	// Validate imageID: must not contain path separators or traversal sequences
	if strings.ContainsAny(imageID, "/\\") || strings.Contains(imageID, "..") || imageID == "" {
		return "", fmt.Errorf("invalid image ID %q", imageID)
	}
	dir := filepath.Join(s.dir, "attachments", beanID)
	return safepath.SafeJoin(dir, imageID)
}

// clearAttachments removes all stored images for a bean.
func (s *store) clearAttachments(beanID string) error {
	if err := safepath.ValidateBeanID(beanID); err != nil {
		return fmt.Errorf("invalid bean ID for attachment cleanup: %w", err)
	}
	dir := filepath.Join(s.dir, "attachments", beanID)
	err := os.RemoveAll(dir)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// pruneAttachments deletes any attachment files for a bean that are NOT in keepIDs.
func (s *store) pruneAttachments(beanID string, keepIDs []string) error {
	if err := safepath.ValidateBeanID(beanID); err != nil {
		return fmt.Errorf("invalid bean ID for attachment prune: %w", err)
	}
	dir := filepath.Join(s.dir, "attachments", beanID)
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read attachment dir: %w", err)
	}

	keep := make(map[string]bool, len(keepIDs))
	for _, id := range keepIDs {
		keep[id] = true
	}

	for _, e := range entries {
		if e.IsDir() || keep[e.Name()] {
			continue
		}
		path := filepath.Join(dir, e.Name())
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove orphaned attachment %s: %w", e.Name(), err)
		}
	}
	return nil
}
