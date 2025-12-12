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
	createStatus    string
	createType      string
	createPriority  string
	createBody      string
	createBodyFile  string
	createTag       []string
	createMilestone string
	createEpic      string
	createFeature   string
	createBlock     []string
	createRelated   []string
	createDuplicate []string
	createJSON      bool
)

var createCmd = &cobra.Command{
	Use:   "create [title]",
	Short: "Create a new bean",
	Long:  `Creates a new bean (issue) with a generated ID and optional title.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		title := strings.Join(args, " ")
		if title == "" {
			title = "Untitled"
		}

		// Validate inputs
		if createStatus != "" && !cfg.IsValidStatus(createStatus) {
			return cmdError(createJSON, output.ErrInvalidStatus, "invalid status: %s (must be %s)", createStatus, cfg.StatusList())
		}
		if createType != "" && !cfg.IsValidType(createType) {
			return cmdError(createJSON, output.ErrValidation, "invalid type: %s (must be %s)", createType, cfg.TypeList())
		}
		if createPriority != "" && !cfg.IsValidPriority(createPriority) {
			return cmdError(createJSON, output.ErrValidation, "invalid priority: %s (must be %s)", createPriority, cfg.PriorityList())
		}

		body, err := resolveContent(createBody, createBodyFile)
		if err != nil {
			return cmdError(createJSON, output.ErrFileError, "%s", err)
		}

		// Build GraphQL input
		input := model.CreateBeanInput{Title: title}
		if createStatus != "" {
			input.Status = &createStatus
		} else {
			defaultStatus := cfg.GetDefaultStatus()
			input.Status = &defaultStatus
		}
		if createType != "" {
			input.Type = &createType
		} else {
			defaultType := cfg.GetDefaultType()
			input.Type = &defaultType
		}
		if createPriority != "" {
			input.Priority = &createPriority
		}
		if body != "" {
			input.Body = &body
		}
		if len(createTag) > 0 {
			input.Tags = createTag
		}

		// Create via GraphQL mutation
		resolver := &graph.Resolver{Core: core}
		mutation := resolver.Mutation()
		ctx := context.Background()

		b, err := mutation.CreateBean(ctx, input)
		if err != nil {
			return cmdError(createJSON, output.ErrFileError, "failed to create bean: %v", err)
		}

		// Apply hierarchy links via separate mutations
		if createMilestone != "" {
			if b, err = mutation.SetMilestone(ctx, b.ID, &createMilestone); err != nil {
				return cmdError(createJSON, output.ErrFileError, "failed to set milestone: %v", err)
			}
		}
		if createEpic != "" {
			if b, err = mutation.SetEpic(ctx, b.ID, &createEpic); err != nil {
				return cmdError(createJSON, output.ErrFileError, "failed to set epic: %v", err)
			}
		}
		if createFeature != "" {
			if b, err = mutation.SetFeature(ctx, b.ID, &createFeature); err != nil {
				return cmdError(createJSON, output.ErrFileError, "failed to set feature: %v", err)
			}
		}

		// Apply relationship links via separate mutations
		for _, target := range createBlock {
			if b, err = mutation.AddBlock(ctx, b.ID, target); err != nil {
				return cmdError(createJSON, output.ErrFileError, "failed to add block: %v", err)
			}
		}
		for _, target := range createRelated {
			if b, err = mutation.AddRelated(ctx, b.ID, target); err != nil {
				return cmdError(createJSON, output.ErrFileError, "failed to add related: %v", err)
			}
		}
		for _, target := range createDuplicate {
			if b, err = mutation.AddDuplicate(ctx, b.ID, target); err != nil {
				return cmdError(createJSON, output.ErrFileError, "failed to add duplicate: %v", err)
			}
		}

		if createJSON {
			return output.Success(b, "Bean created")
		}

		fmt.Println(ui.Success.Render("Created ") + ui.ID.Render(b.ID) + " " + ui.Muted.Render(b.Path))
		return nil
	},
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

	createCmd.Flags().StringVarP(&createStatus, "status", "s", "", "Initial status ("+strings.Join(statusNames, ", ")+")")
	createCmd.Flags().StringVarP(&createType, "type", "t", "", "Bean type ("+strings.Join(typeNames, ", ")+")")
	createCmd.Flags().StringVarP(&createPriority, "priority", "p", "", "Priority level ("+strings.Join(priorityNames, ", ")+")")
	createCmd.Flags().StringVarP(&createBody, "body", "d", "", "Body content (use '-' to read from stdin)")
	createCmd.Flags().StringVar(&createBodyFile, "body-file", "", "Read body from file")
	createCmd.Flags().StringArrayVar(&createTag, "tag", nil, "Add tag (can be repeated)")

	// Hierarchy link flags
	createCmd.Flags().StringVar(&createMilestone, "milestone", "", "Set milestone (bean ID)")
	createCmd.Flags().StringVar(&createEpic, "epic", "", "Set epic (bean ID)")
	createCmd.Flags().StringVar(&createFeature, "feature", "", "Set feature (bean ID, requires task/bug type)")

	// Relationship link flags
	createCmd.Flags().StringArrayVar(&createBlock, "block", nil, "Add block relationship (can be repeated)")
	createCmd.Flags().StringArrayVar(&createRelated, "related", nil, "Add related relationship (can be repeated)")
	createCmd.Flags().StringArrayVar(&createDuplicate, "duplicate", nil, "Add duplicate relationship (can be repeated)")

	createCmd.Flags().BoolVar(&createJSON, "json", false, "Output as JSON")
	createCmd.MarkFlagsMutuallyExclusive("body", "body-file")
	rootCmd.AddCommand(createCmd)
}
