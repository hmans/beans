package bean

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/adrg/frontmatter"
	"gopkg.in/yaml.v3"
)

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
	Links map[string][]string `yaml:"links,omitempty" json:"links,omitempty"`
}

// frontMatter is the subset of Bean that gets serialized to YAML front matter.
// Uses interface{} for Links to handle flexible YAML input via yaml.v2 (used by frontmatter lib).
type frontMatter struct {
	Title     string                 `yaml:"title"`
	Status    string                 `yaml:"status"`
	Type      string                 `yaml:"type,omitempty"`
	CreatedAt *time.Time             `yaml:"created_at,omitempty"`
	UpdatedAt *time.Time             `yaml:"updated_at,omitempty"`
	Links     map[string]interface{} `yaml:"links,omitempty"`
}

// convertLinks converts flexible YAML links (string or []interface{}) to map[string][]string.
func convertLinks(raw map[string]interface{}) map[string][]string {
	if raw == nil {
		return nil
	}

	result := make(map[string][]string)
	for key, val := range raw {
		switch v := val.(type) {
		case string:
			result[key] = []string{v}
		case []interface{}:
			ids := make([]string, 0, len(v))
			for _, item := range v {
				if s, ok := item.(string); ok {
					ids = append(ids, s)
				}
			}
			result[key] = ids
		}
	}
	return result
}

// linksToInterface converts map[string][]string to map[string]interface{} for YAML output.
func linksToInterface(links map[string][]string) map[string]interface{} {
	if len(links) == 0 {
		return nil
	}

	result := make(map[string]interface{})
	for key, ids := range links {
		if len(ids) == 0 {
			continue
		}
		result[key] = ids
	}
	if len(result) == 0 {
		return nil
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

// Render serializes the bean back to markdown with YAML front matter.
func (b *Bean) Render() ([]byte, error) {
	fm := frontMatter{
		Title:     b.Title,
		Status:    b.Status,
		Type:      b.Type,
		CreatedAt: b.CreatedAt,
		UpdatedAt: b.UpdatedAt,
		Links:     linksToInterface(b.Links),
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
