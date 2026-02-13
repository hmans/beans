package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/graph"
	"github.com/hmans/beans/internal/output"
	"github.com/hmans/beans/internal/ui"
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
	return bean.ReplaceOnce(body, old, new)
}

// applyBodyAppend appends text to the body with a newline separator.
func applyBodyAppend(body, text string) string {
	return bean.AppendWithSeparator(body, text)
}

// resolveBean finds a bean by ID, checking the archive if needed.
// Returns the bean and whether it was unarchived.
func resolveBean(resolver *graph.Resolver, id string, jsonMode bool) (*bean.Bean, bool, error) {
	b, err := resolver.Query().Bean(context.Background(), id)
	if err != nil {
		return nil, false, cmdError(jsonMode, output.ErrNotFound, "failed to find bean: %v", err)
	}

	if b == nil {
		unarchived, unarchiveErr := core.LoadAndUnarchive(id)
		if unarchiveErr != nil {
			return nil, false, cmdError(jsonMode, output.ErrNotFound, "bean not found: %s", id)
		}
		b, err = resolver.Query().Bean(context.Background(), unarchived.ID)
		if err != nil || b == nil {
			return nil, false, cmdError(jsonMode, output.ErrNotFound, "bean not found: %s", id)
		}
		return b, true, nil
	}

	return b, false, nil
}

// result holds the outcome of a workflow command for a single bean.
type result struct {
	bean    *bean.Bean
	warning string
}

// outputResults handles JSON and human output for multi-ID workflow commands.
func outputResults(results []*result, errs []error, jsonMode bool, pastTense, pastTenseCapitalized string) error {
	beans := make([]*bean.Bean, 0, len(results))
	var warnings []string
	for _, r := range results {
		beans = append(beans, r.bean)
		if r.warning != "" {
			warnings = append(warnings, fmt.Sprintf("%s: %s", r.bean.ID, r.warning))
		}
	}
	for _, e := range errs {
		warnings = append(warnings, e.Error())
	}

	if jsonMode {
		if len(beans) == 1 && len(warnings) == 0 {
			return output.Success(beans[0], "Bean "+pastTense)
		}
		resp := output.Response{
			Success: true,
			Beans:   beans,
			Count:   len(beans),
			Message: fmt.Sprintf("%d bean(s) %s", len(beans), pastTense),
		}
		if len(warnings) > 0 {
			resp.Warnings = warnings
		}
		return output.JSON(resp)
	}

	for _, r := range results {
		if r.warning == "" {
			fmt.Println(ui.Success.Render(pastTenseCapitalized+" ") + ui.ID.Render(r.bean.ID))
		}
	}
	return nil
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
