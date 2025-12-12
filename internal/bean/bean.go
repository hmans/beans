package bean

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/adrg/frontmatter"
	"gopkg.in/yaml.v3"
)

// Link represents a relationship from this bean to another.
type Link struct {
	Type   string `json:"type"`
	Target string `json:"target"`
}

// Links is a slice of Link with custom YAML marshaling.
// YAML format: array of single-key maps, e.g., [{parent: abc}, {blocks: foo}]
type Links []Link

// MarshalYAML implements yaml.Marshaler for the array-of-single-key-maps format.
func (l Links) MarshalYAML() (interface{}, error) {
	if len(l) == 0 {
		return nil, nil
	}

	result := make([]map[string]string, 0, len(l))
	for _, link := range l {
		result = append(result, map[string]string{link.Type: link.Target})
	}
	return result, nil
}

// UnmarshalYAML implements yaml.Unmarshaler for the array-of-single-key-maps format.
// This handles yaml.v3 format (used by Render).
func (l *Links) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.SequenceNode {
		return fmt.Errorf("links must be a sequence, got %v", node.Kind)
	}

	*l = nil
	for _, item := range node.Content {
		if item.Kind != yaml.MappingNode || len(item.Content) != 2 {
			return fmt.Errorf("each link must be a single-key map")
		}
		link := Link{
			Type:   item.Content[0].Value,
			Target: item.Content[1].Value,
		}
		*l = append(*l, link)
	}
	return nil
}

// HasType returns true if any link has the given type.
func (l Links) HasType(linkType string) bool {
	for _, link := range l {
		if link.Type == linkType {
			return true
		}
	}
	return false
}

// HasLink returns true if a link with the given type and target exists.
func (l Links) HasLink(linkType, target string) bool {
	for _, link := range l {
		if link.Type == linkType && link.Target == target {
			return true
		}
	}
	return false
}

// Targets returns all target IDs for a specific link type.
func (l Links) Targets(linkType string) []string {
	var result []string
	for _, link := range l {
		if link.Type == linkType {
			result = append(result, link.Target)
		}
	}
	return result
}

// Add adds a link if it doesn't already exist, returns modified Links.
func (l Links) Add(linkType, target string) Links {
	if l.HasLink(linkType, target) {
		return l
	}
	return append(l, Link{Type: linkType, Target: target})
}

// Remove removes a link, returns modified Links.
func (l Links) Remove(linkType, target string) Links {
	result := make(Links, 0, len(l))
	for _, link := range l {
		if !(link.Type == linkType && link.Target == target) {
			result = append(result, link)
		}
	}
	return result
}

// tagPattern matches valid tags: lowercase letters, numbers, and hyphens.
// Must start with a letter, can contain hyphens but not consecutively or at the end.
var tagPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$`)

// ValidateTag checks if a tag is valid (lowercase, URL-safe, single word).
func ValidateTag(tag string) error {
	if tag == "" {
		return fmt.Errorf("tag cannot be empty")
	}
	if !tagPattern.MatchString(tag) {
		return fmt.Errorf("invalid tag %q: must be lowercase, start with a letter, and contain only letters, numbers, and hyphens", tag)
	}
	return nil
}

// NormalizeTag converts a tag to its canonical form (lowercase).
func NormalizeTag(tag string) string {
	return strings.ToLower(strings.TrimSpace(tag))
}

// HasTag returns true if the bean has the specified tag.
func (b *Bean) HasTag(tag string) bool {
	normalized := NormalizeTag(tag)
	for _, t := range b.Tags {
		if t == normalized {
			return true
		}
	}
	return false
}

// AddTag adds a tag to the bean if it doesn't already exist.
// Returns an error if the tag is invalid.
func (b *Bean) AddTag(tag string) error {
	normalized := NormalizeTag(tag)
	if err := ValidateTag(normalized); err != nil {
		return err
	}
	if !b.HasTag(normalized) {
		b.Tags = append(b.Tags, normalized)
	}
	return nil
}

// RemoveTag removes a tag from the bean.
func (b *Bean) RemoveTag(tag string) {
	normalized := NormalizeTag(tag)
	result := make([]string, 0, len(b.Tags))
	for _, t := range b.Tags {
		if t != normalized {
			result = append(result, t)
		}
	}
	b.Tags = result
}

// HasParent returns true if the bean has a parent.
func (b *Bean) HasParent() bool {
	return b.Parent != ""
}

// BlocksBean returns true if this bean blocks the given bean ID.
func (b *Bean) BlocksBean(id string) bool {
	for _, target := range b.Blocks {
		if target == id {
			return true
		}
	}
	return false
}

// AddBlock adds a bean ID to the blocks list if not already present.
func (b *Bean) AddBlock(id string) {
	if !b.BlocksBean(id) {
		b.Blocks = append(b.Blocks, id)
	}
}

// RemoveBlock removes a bean ID from the blocks list.
func (b *Bean) RemoveBlock(id string) {
	result := make([]string, 0, len(b.Blocks))
	for _, target := range b.Blocks {
		if target != id {
			result = append(result, target)
		}
	}
	b.Blocks = result
}

// Bean represents an issue stored as a markdown file with front matter.
type Bean struct {
	// ID is the unique NanoID identifier (from filename).
	ID string `yaml:"-" json:"id"`
	// Slug is the optional human-readable part of the filename.
	Slug string `yaml:"-" json:"slug,omitempty"`
	// Path is the relative path from .beans/ root (e.g., "epic-auth/abc123-login.md").
	Path string `yaml:"-" json:"path"`

	// Front matter fields
	Title     string     `yaml:"title" json:"title"`
	Status    string     `yaml:"status" json:"status"`
	Type      string     `yaml:"type,omitempty" json:"type,omitempty"`
	Priority  string     `yaml:"priority,omitempty" json:"priority,omitempty"`
	Tags      []string   `yaml:"tags,omitempty" json:"tags,omitempty"`
	CreatedAt *time.Time `yaml:"created_at,omitempty" json:"created_at,omitempty"`
	UpdatedAt *time.Time `yaml:"updated_at,omitempty" json:"updated_at,omitempty"`

	// Body is the markdown content after the front matter.
	Body string `yaml:"-" json:"body,omitempty"`

	// Parent is the optional parent bean ID (milestone, epic, or feature).
	Parent string `yaml:"parent,omitempty" json:"parent,omitempty"`

	// Blocks is a list of bean IDs that this bean blocks.
	Blocks []string `yaml:"blocks,omitempty" json:"blocks,omitempty"`
}

// frontMatter is the subset of Bean that gets serialized to YAML front matter.
// Uses []interface{} for Links to handle flexible YAML input via yaml.v2 (used by frontmatter lib).
type frontMatter struct {
	Title     string      `yaml:"title"`
	Status    string      `yaml:"status"`
	Type      string      `yaml:"type,omitempty"`
	Priority  string      `yaml:"priority,omitempty"`
	Tags      []string    `yaml:"tags,omitempty"`
	CreatedAt *time.Time  `yaml:"created_at,omitempty"`
	UpdatedAt *time.Time  `yaml:"updated_at,omitempty"`
	Parent    string      `yaml:"parent,omitempty"`
	Blocks    []string    `yaml:"blocks,omitempty"`
	// Links is kept for reading old format during migration
	Links interface{} `yaml:"links,omitempty"`
}

// convertLinks converts flexible YAML links from frontmatter lib (yaml.v2) to Links.
// Input format: []interface{} where each element is map[interface{}]interface{} with single key.
func convertLinks(raw interface{}) Links {
	if raw == nil {
		return nil
	}

	// Handle []interface{} from yaml.v2
	slice, ok := raw.([]interface{})
	if !ok {
		return nil
	}

	var result Links
	for _, item := range slice {
		// Each item should be a map with a single key
		m, ok := item.(map[interface{}]interface{})
		if !ok {
			continue
		}
		for k, v := range m {
			keyStr, ok1 := k.(string)
			valStr, ok2 := v.(string)
			if ok1 && ok2 {
				result = append(result, Link{Type: keyStr, Target: valStr})
			}
		}
	}
	return result
}

// Parse reads a bean from a reader (markdown with YAML front matter).
func Parse(r io.Reader) (*Bean, error) {
	var fm frontMatter
	body, err := frontmatter.Parse(r, &fm)
	if err != nil {
		return nil, fmt.Errorf("parsing front matter: %w", err)
	}

	b := &Bean{
		Title:     fm.Title,
		Status:    fm.Status,
		Type:      fm.Type,
		Priority:  fm.Priority,
		Tags:      fm.Tags,
		CreatedAt: fm.CreatedAt,
		UpdatedAt: fm.UpdatedAt,
		Body:      string(body),
		Parent:    fm.Parent,
		Blocks:    fm.Blocks,
	}

	// Migrate old links format if present
	if fm.Links != nil {
		oldLinks := convertLinks(fm.Links)
		for _, link := range oldLinks {
			switch link.Type {
			case "parent":
				if b.Parent == "" {
					b.Parent = link.Target
				}
			case "blocks":
				if !b.BlocksBean(link.Target) {
					b.Blocks = append(b.Blocks, link.Target)
				}
			// "duplicates" and "related" are silently dropped
			}
		}
	}

	return b, nil
}

// renderFrontMatter is used for YAML output with yaml.v3 (supports custom marshalers).
type renderFrontMatter struct {
	Title     string     `yaml:"title"`
	Status    string     `yaml:"status"`
	Type      string     `yaml:"type,omitempty"`
	Priority  string     `yaml:"priority,omitempty"`
	Tags      []string   `yaml:"tags,omitempty"`
	CreatedAt *time.Time `yaml:"created_at,omitempty"`
	UpdatedAt *time.Time `yaml:"updated_at,omitempty"`
	Parent    string     `yaml:"parent,omitempty"`
	Blocks    []string   `yaml:"blocks,omitempty"`
}

// Render serializes the bean back to markdown with YAML front matter.
func (b *Bean) Render() ([]byte, error) {
	fm := renderFrontMatter{
		Title:     b.Title,
		Status:    b.Status,
		Type:      b.Type,
		Priority:  b.Priority,
		Tags:      b.Tags,
		CreatedAt: b.CreatedAt,
		UpdatedAt: b.UpdatedAt,
		Parent:    b.Parent,
		Blocks:    b.Blocks,
	}

	fmBytes, err := yaml.Marshal(&fm)
	if err != nil {
		return nil, fmt.Errorf("marshaling front matter: %w", err)
	}

	var buf bytes.Buffer
	buf.WriteString("---\n")
	buf.Write(fmBytes)
	buf.WriteString("---\n")
	if b.Body != "" {
		// Only add newline separator if body doesn't already start with one
		if !strings.HasPrefix(b.Body, "\n") {
			buf.WriteString("\n")
		}
		buf.WriteString(b.Body)
	}

	return buf.Bytes(), nil
}
