package cmd

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hmans/beans/internal/config"
	"github.com/hmans/beans/internal/graph"
	"github.com/hmans/beans/internal/graph/model"
	"github.com/hmans/beans/internal/output"
	"github.com/hmans/beans/internal/ui"
	"github.com/spf13/cobra"
)

var (
	updateStatus         string
	updateType           string
	updatePriority       string
	updateTitle          string
	updateBody           string
	updateBodyFile       string
	updateParent         string
	updateRemoveParent   bool
	updateBlocking       []string
	updateRemoveBlocking []string
	updateTag            []string
	updateRemoveTag      []string
	updateIfMatch        string
	updateJSON           bool
)

var updateCmd = &cobra.Command{
	Use:     "update <id>",
	Aliases: []string{"u"},
	Short:   "Update a bean's properties",
	Long:    `Updates one or more properties of an existing bean.`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		resolver := &graph.Resolver{Core: core}

		// Find the bean
		b, err := resolver.Query().Bean(ctx, args[0])
		if err != nil {
			return cmdError(updateJSON, output.ErrNotFound, "failed to find bean: %v", err)
		}

		// If not found, check the archive and unarchive if present
		wasArchived := false
		if b == nil {
			unarchived, unarchiveErr := core.LoadAndUnarchive(args[0])
			if unarchiveErr != nil {
				return cmdError(updateJSON, output.ErrNotFound, "bean not found: %s", args[0])
			}
			// Re-query to get the model.Bean
			b, err = resolver.Query().Bean(ctx, unarchived.ID)
			if err != nil || b == nil {
				return cmdError(updateJSON, output.ErrNotFound, "bean not found: %s", args[0])
			}
			wasArchived = true
		}

		// Track changes for output
		var changes []string

		// Prepare ifMatch for GraphQL mutations
		var ifMatch *string
		if updateIfMatch != "" {
			ifMatch = &updateIfMatch
		}

		// Build and validate field updates
		input, fieldChanges, err := buildUpdateInput(cmd, b.Tags)
		if err != nil {
			return cmdError(updateJSON, output.ErrValidation, "%s", err)
		}
		changes = append(changes, fieldChanges...)

		// Add ifMatch to input if provided
		if ifMatch != nil {
			input.IfMatch = ifMatch
		}

		// Apply field updates
		if hasFieldUpdates(input) {
			b, err = resolver.Mutation().UpdateBean(ctx, b.ID, input)
			if err != nil {
				return mutationError(updateJSON, err)
			}
		}

		// Handle parent changes
		if cmd.Flags().Changed("parent") || updateRemoveParent {
			var parentID *string
			if !updateRemoveParent && updateParent != "" {
				parentID = &updateParent
			}
			b, err = resolver.Mutation().SetParent(ctx, b.ID, parentID, ifMatch)
			if err != nil {
				return mutationError(updateJSON, err)
			}
			changes = append(changes, "parent")
		}

		// Process blocking additions
		for _, targetID := range updateBlocking {
			b, err = resolver.Mutation().AddBlocking(ctx, b.ID, targetID, ifMatch)
			if err != nil {
				return mutationError(updateJSON, err)
			}
			changes = append(changes, "blocking")
		}

		// Process blocking removals
		for _, targetID := range updateRemoveBlocking {
			b, err = resolver.Mutation().RemoveBlocking(ctx, b.ID, targetID, ifMatch)
			if err != nil {
				return mutationError(updateJSON, err)
			}
			changes = append(changes, "blocking")
		}

		// Require at least one change
		if len(changes) == 0 {
			return cmdError(updateJSON, output.ErrValidation,
				"no changes specified (use --status, --type, --priority, --title, --body, --parent, --blocking, --tag, or their --remove-* variants)")
		}

		// Output result
		if updateJSON {
			msg := "Bean updated"
			if wasArchived {
				msg = "Bean unarchived and updated"
			}
			return output.Success(b, msg)
		}

		if wasArchived {
			fmt.Println(ui.Success.Render("Unarchived and updated ") + ui.ID.Render(b.ID) + " " + ui.Muted.Render(b.Path))
		} else {
			fmt.Println(ui.Success.Render("Updated ") + ui.ID.Render(b.ID) + " " + ui.Muted.Render(b.Path))
		}
		return nil
	},
}

// buildUpdateInput constructs the GraphQL input from flags and returns which fields changed.
func buildUpdateInput(cmd *cobra.Command, existingTags []string) (model.UpdateBeanInput, []string, error) {
	var input model.UpdateBeanInput
	var changes []string

	if cmd.Flags().Changed("status") {
		if !cfg.IsValidStatus(updateStatus) {
			return input, nil, fmt.Errorf("invalid status: %s (must be %s)", updateStatus, cfg.StatusList())
		}
		input.Status = &updateStatus
		changes = append(changes, "status")
	}

	if cmd.Flags().Changed("type") {
		if !cfg.IsValidType(updateType) {
			return input, nil, fmt.Errorf("invalid type: %s (must be %s)", updateType, cfg.TypeList())
		}
		input.Type = &updateType
		changes = append(changes, "type")
	}

	if cmd.Flags().Changed("priority") {
		if !cfg.IsValidPriority(updatePriority) {
			return input, nil, fmt.Errorf("invalid priority: %s (must be %s)", updatePriority, cfg.PriorityList())
		}
		input.Priority = &updatePriority
		changes = append(changes, "priority")
	}

	if cmd.Flags().Changed("title") {
		input.Title = &updateTitle
		changes = append(changes, "title")
	}

	// Handle body modifications
	if cmd.Flags().Changed("body") || cmd.Flags().Changed("body-file") {
		body, err := resolveContent(updateBody, updateBodyFile)
		if err != nil {
			return input, nil, err
		}
		input.Body = &body
		changes = append(changes, "body")
	}

	if len(updateTag) > 0 || len(updateRemoveTag) > 0 {
		input.Tags = mergeTags(existingTags, updateTag, updateRemoveTag)
		changes = append(changes, "tags")
	}

	return input, changes, nil
}

// hasFieldUpdates returns true if any field in the input is set.
func hasFieldUpdates(input model.UpdateBeanInput) bool {
	return input.Status != nil || input.Type != nil || input.Priority != nil ||
		input.Title != nil || input.Body != nil || input.Tags != nil
}

// isConflictError returns true if the error is an ETag-related conflict error.
func isConflictError(err error) bool {
	var mismatchErr *graph.ETagMismatchError
	var requiredErr *graph.ETagRequiredError
	return errors.As(err, &mismatchErr) || errors.As(err, &requiredErr)
}

// mutationError returns a cmdError with the appropriate error code based on the error type.
func mutationError(jsonOutput bool, err error) error {
	if isConflictError(err) {
		return cmdError(jsonOutput, output.ErrConflict, "%s", err)
	}
	return cmdError(jsonOutput, output.ErrValidation, "%s", err)
}

func init() {
	// Build help text with allowed values from hardcoded config
	statusNames := make([]string, len(config.DefaultStatuses))
	for i, s := range config.DefaultStatuses {
		statusNames[i] = s.Name
	}
	typeNames := make([]string, len(config.DefaultTypes))
	for i, t := range config.DefaultTypes {
		typeNames[i] = t.Name
	}
	priorityNames := make([]string, len(config.DefaultPriorities))
	for i, p := range config.DefaultPriorities {
		priorityNames[i] = p.Name
	}

	updateCmd.Flags().StringVarP(&updateStatus, "status", "s", "", "New status ("+strings.Join(statusNames, ", ")+")")
	updateCmd.Flags().StringVarP(&updateType, "type", "t", "", "New type ("+strings.Join(typeNames, ", ")+")")
	updateCmd.Flags().StringVarP(&updatePriority, "priority", "p", "", "New priority ("+strings.Join(priorityNames, ", ")+", or empty to clear)")
	updateCmd.Flags().StringVar(&updateTitle, "title", "", "New title")
	updateCmd.Flags().StringVarP(&updateBody, "body", "d", "", "New body (use '-' to read from stdin)")
	updateCmd.Flags().StringVar(&updateBodyFile, "body-file", "", "Read body from file")
	updateCmd.Flags().StringVar(&updateParent, "parent", "", "Set parent bean ID")
	updateCmd.Flags().BoolVar(&updateRemoveParent, "remove-parent", false, "Remove parent")
	updateCmd.Flags().StringArrayVar(&updateBlocking, "blocking", nil, "ID of bean this blocks (can be repeated)")
	updateCmd.Flags().StringArrayVar(&updateRemoveBlocking, "remove-blocking", nil, "ID of bean to unblock (can be repeated)")
	updateCmd.Flags().StringArrayVar(&updateTag, "tag", nil, "Add tag (can be repeated)")
	updateCmd.Flags().StringArrayVar(&updateRemoveTag, "remove-tag", nil, "Remove tag (can be repeated)")
	updateCmd.Flags().StringVar(&updateIfMatch, "if-match", "", "Only update if etag matches (optimistic locking)")
	updateCmd.MarkFlagsMutuallyExclusive("parent", "remove-parent")
	updateCmd.Flags().BoolVar(&updateJSON, "json", false, "Output as JSON")
	updateCmd.MarkFlagsMutuallyExclusive("body", "body-file")
	rootCmd.AddCommand(updateCmd)
}
