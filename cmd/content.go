package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/output"
)

// resolveContent returns content from a direct value or file flag.
// If value is "-", reads from stdin.
func resolveContent(value, file string) (string, error) {
	if value != "" && file != "" {
		return "", fmt.Errorf("cannot use both --body and --body-file")
	}

	if value == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("reading stdin: %w", err)
		}
		return string(data), nil
	}

	if value != "" {
		return value, nil
	}

	if file != "" {
		data, err := os.ReadFile(file)
		if err != nil {
			return "", fmt.Errorf("reading file: %w", err)
		}
		return string(data), nil
	}

	return "", nil
}

// applyTags adds tags to a bean, returning an error if any tag is invalid.
func applyTags(b *bean.Bean, tags []string) error {
	for _, tag := range tags {
		if err := b.AddTag(tag); err != nil {
			return err
		}
	}
	return nil
}

// formatCycle formats a cycle path for display.
func formatCycle(path []string) string {
	return strings.Join(path, " → ")
}

// cmdError returns an appropriate error for JSON or text mode.
// Note: Use %v instead of %w for error arguments - wrapping is not preserved in JSON mode.
func cmdError(jsonMode bool, code string, format string, args ...any) error {
	if jsonMode {
		return output.Error(code, fmt.Sprintf(format, args...))
	}
	return fmt.Errorf(format, args...)
}

// mergeTags combines existing tags with additions and removals.
func mergeTags(existing, add, remove []string) []string {
	tags := make(map[string]bool)
	for _, t := range existing {
		tags[t] = true
	}
	for _, t := range add {
		tags[t] = true
	}
	for _, t := range remove {
		delete(tags, t)
	}
	result := make([]string, 0, len(tags))
	for t := range tags {
		result = append(result, t)
	}
	return result
}

// applyBodyReplace replaces exactly one occurrence of old with new.
// Returns an error if old is not found or found multiple times.
func applyBodyReplace(body, old, new string) (string, error) {
	if old == "" {
		return "", fmt.Errorf("--old cannot be empty")
	}
	count := strings.Count(body, old)
	if count == 0 {
		return "", fmt.Errorf("text not found in body")
	}
	if count > 1 {
		return "", fmt.Errorf("text found %d times in body (must be unique)", count)
	}
	return strings.Replace(body, old, new, 1), nil
}

// applyBodyAppend appends text to the body with a newline separator.
func applyBodyAppend(body, text string) string {
	if body == "" {
		return text
	}
	// Ensure single newline separator
	body = strings.TrimRight(body, "\n")
	return body + "\n\n" + text
}

// resolveAppendContent handles --append value, supporting stdin with "-".
func resolveAppendContent(value string) (string, error) {
	if value == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("reading stdin: %w", err)
		}
		return strings.TrimRight(string(data), "\n"), nil
	}
	return value, nil
}
