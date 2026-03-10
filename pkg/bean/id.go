package bean

import (
	"regexp"
	"strings"
	"unicode"

	gonanoid "github.com/matoous/go-nanoid/v2"
)

const idAlphabet = "0123456789abcdefghijklmnopqrstuvwxyz"

// NewID generates a new NanoID for a bean with an optional prefix and configurable length.
func NewID(prefix string, length int) string {
	id, err := gonanoid.Generate(idAlphabet, length)
	if err != nil {
		panic(err) // should never happen with valid alphabet
	}
	return prefix + id
}

// ParseFilename extracts the ID and optional slug from a bean filename.
// Supports multiple formats for backward compatibility:
//   - New format: "f7g--user-registration.md" -> ("f7g", "user-registration")
//   - Dot format: "f7g.user-registration.md" -> ("f7g", "user-registration")
//   - Legacy format: "f7g-user-registration.md" -> ("f7g", "user-registration")
//   - ID only: "f7g.md" -> ("f7g", "")
func ParseFilename(name string) (id, slug string) {
	// Remove .md extension
	name = strings.TrimSuffix(name, ".md")

	// Try new format first (double-dash separator): id--slug
	if idx := strings.Index(name, "--"); idx > 0 {
		return name[:idx], name[idx+2:]
	}

	// Try dot format: id.slug
	if idx := strings.Index(name, "."); idx > 0 {
		return name[:idx], name[idx+1:]
	}

	// Fall back to original legacy format (single dash separator): id-slug
	parts := strings.SplitN(name, "-", 2)
	id = parts[0]
	if len(parts) > 1 {
		slug = parts[1]
	}
	return id, slug
}

// BuildFilename constructs a filename from ID and optional slug.
// Uses double-dash separator: id--slug.md
func BuildFilename(id, slug string) string {
	if slug == "" {
		return id + ".md"
	}
	return id + "--" + slug + ".md"
}

// Slugify converts a title to a URL-friendly slug.
func Slugify(title string) string {
	// Convert to lowercase
	s := strings.ToLower(title)

	// Replace spaces and underscores with dashes
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")

	// Remove non-alphanumeric characters (except dashes)
	var result strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' {
			result.WriteRune(r)
		}
	}
	s = result.String()

	// Collapse multiple dashes
	re := regexp.MustCompile(`-+`)
	s = re.ReplaceAllString(s, "-")

	// Trim dashes from ends
	s = strings.Trim(s, "-")

	// Truncate to reasonable length
	if len(s) > 50 {
		s = s[:50]
		// Don't end with a dash
		s = strings.TrimRight(s, "-")
	}

	return s
}
