package cmd

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// parsePropertyFlags parses --set flags in the form "key=value" and returns
// a map with YAML-inferred types (e.g., "3" → int, "true" → bool).
func parsePropertyFlags(flags []string) (map[string]interface{}, error) {
	result := make(map[string]interface{}, len(flags))
	for _, flag := range flags {
		key, value, ok := strings.Cut(flag, "=")
		if !ok {
			return nil, fmt.Errorf("invalid property format %q: expected key=value", flag)
		}
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("invalid property format %q: key cannot be empty", flag)
		}

		// Use YAML unmarshaling for automatic type detection
		var parsed any
		if err := yaml.Unmarshal([]byte(value), &parsed); err != nil {
			// Fall back to raw string if YAML parsing fails
			parsed = value
		}
		// yaml.Unmarshal returns nil for empty string — preserve as empty string
		if parsed == nil {
			parsed = ""
		}
		result[key] = parsed
	}
	return result, nil
}
