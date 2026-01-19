package bean

import (
	"fmt"
	"strings"
)

// ReplaceOnce replaces exactly one occurrence of old with new in text.
// Returns an error if old is empty, not found, or found multiple times.
// The new string can be empty to delete the matched text.
func ReplaceOnce(text, old, new string) (string, error) {
	if old == "" {
		return "", fmt.Errorf("old text cannot be empty")
	}
	count := strings.Count(text, old)
	if count == 0 {
		return "", fmt.Errorf("text not found in body")
	}
	if count > 1 {
		return "", fmt.Errorf("text found %d times in body (must be unique)", count)
	}
	return strings.Replace(text, old, new, 1), nil
}

// AppendWithSeparator appends addition to text with a blank line separator.
// If text is empty, returns addition without separator.
// If addition is empty, returns text unchanged (no-op).
func AppendWithSeparator(text, addition string) string {
	if addition == "" {
		return text
	}
	if text == "" {
		return addition
	}
	// Ensure single newline separator
	text = strings.TrimRight(text, "\n")
	return text + "\n\n" + addition
}
