package bean

import (
	"bytes"
	"fmt"
	"io"
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
	CreatedAt *time.Time `yaml:"created_at,omitempty" json:"created_at,omitempty"`
	UpdatedAt *time.Time `yaml:"updated_at,omitempty" json:"updated_at,omitempty"`

	// Body is the markdown content after the front matter.
	Body string `yaml:"-" json:"body,omitempty"`

	// Links are relationships to other beans (e.g., "blocks", "parent").
	Links Links `yaml:"links,omitempty" json:"links,omitempty"`
}

// frontMatter is the subset of Bean that gets serialized to YAML front matter.
// Uses []interface{} for Links to handle flexible YAML input via yaml.v2 (used by frontmatter lib).
type frontMatter struct {
	Title     string      `yaml:"title"`
	Status    string      `yaml:"status"`
	Type      string      `yaml:"type,omitempty"`
	CreatedAt *time.Time  `yaml:"created_at,omitempty"`
	UpdatedAt *time.Time  `yaml:"updated_at,omitempty"`
	Links     interface{} `yaml:"links,omitempty"`
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

	return &Bean{
		Title:     fm.Title,
		Status:    fm.Status,
		Type:      fm.Type,
		CreatedAt: fm.CreatedAt,
		UpdatedAt: fm.UpdatedAt,
		Body:      string(body),
		Links:     convertLinks(fm.Links),
	}, nil
}

// renderFrontMatter is used for YAML output with yaml.v3 (supports custom marshalers).
type renderFrontMatter struct {
	Title     string     `yaml:"title"`
	Status    string     `yaml:"status"`
	Type      string     `yaml:"type,omitempty"`
	CreatedAt *time.Time `yaml:"created_at,omitempty"`
	UpdatedAt *time.Time `yaml:"updated_at,omitempty"`
	Links     Links      `yaml:"links,omitempty"`
}

// Render serializes the bean back to markdown with YAML front matter.
func (b *Bean) Render() ([]byte, error) {
	fm := renderFrontMatter{
		Title:     b.Title,
		Status:    b.Status,
		Type:      b.Type,
		CreatedAt: b.CreatedAt,
		UpdatedAt: b.UpdatedAt,
		Links:     b.Links,
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
		buf.WriteString("\n")
		buf.WriteString(b.Body)
	}

	return buf.Bytes(), nil
}
