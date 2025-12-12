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

// sliceContains checks if a slice contains a value.
func sliceContains(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}

// sliceAdd adds a value to a slice if not already present.
func sliceAdd(slice []string, val string) []string {
	if sliceContains(slice, val) {
		return slice
	}
	return append(slice, val)
}

// sliceRemove removes a value from a slice.
func sliceRemove(slice []string, val string) []string {
	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if s != val {
			result = append(result, s)
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

	// Hierarchy links (single target each)
	Milestone string `yaml:"milestone,omitempty" json:"milestone,omitempty"`
	Epic      string `yaml:"epic,omitempty" json:"epic,omitempty"`
	Feature   string `yaml:"feature,omitempty" json:"feature,omitempty"`

	// Relationship links (multiple targets)
	Blocks []string `yaml:"blocks,omitempty" json:"blocks,omitempty"`
}

// Link helper methods for Blocks

// HasBlock returns true if this bean blocks the given target.
func (b *Bean) HasBlock(target string) bool {
	return sliceContains(b.Blocks, target)
}

// AddBlock adds a block relationship to the given target.
func (b *Bean) AddBlock(target string) {
	b.Blocks = sliceAdd(b.Blocks, target)
}

// RemoveBlock removes a block relationship to the given target.
func (b *Bean) RemoveBlock(target string) {
	b.Blocks = sliceRemove(b.Blocks, target)
}

// frontMatter is the subset of Bean that gets serialized to YAML front matter.
// Used for parsing via yaml.v2 (from frontmatter lib).
type frontMatter struct {
	Title     string     `yaml:"title"`
	Status    string     `yaml:"status"`
	Type      string     `yaml:"type,omitempty"`
	Priority  string     `yaml:"priority,omitempty"`
	Tags      []string   `yaml:"tags,omitempty"`
	CreatedAt *time.Time `yaml:"created_at,omitempty"`
	UpdatedAt *time.Time `yaml:"updated_at,omitempty"`

	// Hierarchy links
	Milestone string `yaml:"milestone,omitempty"`
	Epic      string `yaml:"epic,omitempty"`
	Feature   string `yaml:"feature,omitempty"`

	// Relationship links
	Blocks []string `yaml:"blocks,omitempty"`
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
		Priority:  fm.Priority,
		Tags:      fm.Tags,
		CreatedAt: fm.CreatedAt,
		UpdatedAt: fm.UpdatedAt,
		Body:      string(body),
		// Hierarchy links
		Milestone: fm.Milestone,
		Epic:      fm.Epic,
		Feature:   fm.Feature,
		// Relationship links
		Blocks: fm.Blocks,
	}, nil
}

// renderFrontMatter is used for YAML output with yaml.v3.
type renderFrontMatter struct {
	Title     string     `yaml:"title"`
	Status    string     `yaml:"status"`
	Type      string     `yaml:"type,omitempty"`
	Priority  string     `yaml:"priority,omitempty"`
	Tags      []string   `yaml:"tags,omitempty"`
	CreatedAt *time.Time `yaml:"created_at,omitempty"`
	UpdatedAt *time.Time `yaml:"updated_at,omitempty"`

	// Hierarchy links
	Milestone string `yaml:"milestone,omitempty"`
	Epic      string `yaml:"epic,omitempty"`
	Feature   string `yaml:"feature,omitempty"`

	// Relationship links
	Blocks []string `yaml:"blocks,omitempty"`
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
		// Hierarchy links
		Milestone: b.Milestone,
		Epic:      b.Epic,
		Feature:   b.Feature,
		// Relationship links
		Blocks: b.Blocks,
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
