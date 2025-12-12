package cmd

import (
	"context"
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
	updateStatus   string
	updateType     string
	updatePriority string
	updateTitle    string
	updateBody     string
	updateBodyFile string
	updateParent   string
	updateNoParent   bool
	updateBlocking   []string
	updateUnblocking []string
	updateTag        []string
	updateUntag    []string
	updateJSON     bool
)

var updateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a bean's properties",
	Long: `Updates one or more properties of an existing bean.

Use flags to specify which properties to update:
  --status       Change the status
  --type         Change the type
  --priority     Change the priority
  --title        Change the title
  --body         Change the body (use '-' to read from stdin)
  --parent       Set parent bean ID
  --no-parent    Remove parent
  --blocking     Add to blocking list (can be repeated)
  --unblocking   Remove from blocking list (can be repeated)
  --tag          Add a tag (can be repeated)
  --untag        Remove a tag (can be repeated)`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		resolver := &graph.Resolver{Core: core}

		// Find the bean
		b, err := resolver.Query().Bean(ctx, args[0])
		if err != nil {
			return cmdError(updateJSON, output.ErrNotFound, "failed to find bean: %v", err)
		}
		if b == nil {
			return cmdError(updateJSON, output.ErrNotFound, "bean not found: %s", args[0])
		}

		// Track changes for output
		var changes []string

		// Build and validate field updates
		input, fieldChanges, err := buildUpdateInput(cmd, b.Tags)
		if err != nil {
			return cmdError(updateJSON, output.ErrValidation, "%s", err)
		}
		changes = append(changes, fieldChanges...)

		// Apply field updates
		if hasFieldUpdates(input) {
			b, err = resolver.Mutation().UpdateBean(ctx, b.ID, input)
			if err != nil {
				return cmdError(updateJSON, output.ErrFileError, "failed to save bean: %v", err)
			}
		}

		// Handle parent changes
		if cmd.Flags().Changed("parent") || updateNoParent {
			var parentID *string
			if !updateNoParent && updateParent != "" {
				parentID = &updateParent
			}
			b, err = resolver.Mutation().SetParent(ctx, b.ID, parentID)
			if err != nil {
				return cmdError(updateJSON, output.ErrValidation, "%s", err)
			}
			changes = append(changes, "parent")
		}

		// Process blocking additions
		for _, targetID := range updateBlocking {
			b, err = resolver.Mutation().AddBlocking(ctx, b.ID, targetID)
			if err != nil {
				return cmdError(updateJSON, output.ErrValidation, "%s", err)
			}
			changes = append(changes, "blocking")
		}

		// Process blocking removals
		for _, targetID := range updateUnblocking {
			b, err = resolver.Mutation().RemoveBlocking(ctx, b.ID, targetID)
			if err != nil {
				return cmdError(updateJSON, output.ErrValidation, "%s", err)
			}
			changes = append(changes, "blocking")
		}

		// Require at least one change
		if len(changes) == 0 {
			return cmdError(updateJSON, output.ErrValidation,
				"no changes specified (use --status, --type, --priority, --title, --body, --parent, --blocking, --unblocking, --tag, or --untag)")
		}

		// Output result
		if updateJSON {
			return output.Success(b, "Bean updated")
		}

		fmt.Println(ui.Success.Render("Updated ") + ui.ID.Render(b.ID) + " " + ui.Muted.Render(b.Path))
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

	if cmd.Flags().Changed("body") || cmd.Flags().Changed("body-file") {
		body, err := resolveContent(updateBody, updateBodyFile)
		if err != nil {
			return input, nil, err
		}
		input.Body = &body
		changes = append(changes, "body")
	}

	if len(updateTag) > 0 || len(updateUntag) > 0 {
		input.Tags = mergeTags(existingTags, updateTag, updateUntag)
		changes = append(changes, "tags")
	}

	return input, changes, nil
}

// hasFieldUpdates returns true if any field in the input is set.
func hasFieldUpdates(input model.UpdateBeanInput) bool {
	return input.Status != nil || input.Type != nil || input.Priority != nil ||
		input.Title != nil || input.Body != nil || input.Tags != nil
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
	updateCmd.Flags().BoolVar(&updateNoParent, "no-parent", false, "Remove parent")
	updateCmd.Flags().StringArrayVar(&updateBlocking, "blocking", nil, "ID of bean this blocks (can be repeated)")
	updateCmd.Flags().StringArrayVar(&updateUnblocking, "unblocking", nil, "ID of bean to unblock (can be repeated)")
	updateCmd.Flags().StringArrayVar(&updateTag, "tag", nil, "Add tag (can be repeated)")
	updateCmd.Flags().StringArrayVar(&updateUntag, "untag", nil, "Remove tag (can be repeated)")
	updateCmd.MarkFlagsMutuallyExclusive("parent", "no-parent")
	updateCmd.Flags().BoolVar(&updateJSON, "json", false, "Output as JSON")
	updateCmd.MarkFlagsMutuallyExclusive("body", "body-file")
	rootCmd.AddCommand(updateCmd)
}
