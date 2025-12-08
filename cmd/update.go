package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"hmans.dev/beans/internal/config"
	"hmans.dev/beans/internal/output"
	"hmans.dev/beans/internal/ui"
)

var (
	updateStatus   string
	updateType     string
	updateTitle    string
	updateBody     string
	updateBodyFile string
	updateLink     []string
	updateUnlink   []string
	updateTag      []string
	updateUntag    []string
	updateNoEdit   bool
	updateJSON     bool
)

var updateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a bean's properties",
	Long: `Updates one or more properties of an existing bean.

Use flags to specify which properties to update:
  --status       Change the status
  --type         Change the type
  --title        Change the title
  --body         Change the body (use '-' to read from stdin)
  --link         Add a relationship (format: type:id, can be repeated)
  --unlink       Remove a relationship (format: type:id, can be repeated)
  --tag          Add a tag (can be repeated)
  --untag        Remove a tag (can be repeated)

Relationship types: blocks, duplicates, parent, related`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		// Find the bean
		b, err := core.Get(id)
		if err != nil {
			if updateJSON {
				return output.Error(output.ErrNotFound, err.Error())
			}
			return fmt.Errorf("failed to find bean: %w", err)
		}

		// Track what changed for output message
		var changes []string
		var warnings []string

		// Update status if provided
		if cmd.Flags().Changed("status") {
			if !cfg.IsValidStatus(updateStatus) {
				if updateJSON {
					return output.Error(output.ErrInvalidStatus, fmt.Sprintf("invalid status: %s (must be %s)", updateStatus, cfg.StatusList()))
				}
				return fmt.Errorf("invalid status: %s (must be %s)", updateStatus, cfg.StatusList())
			}
			b.Status = updateStatus
			changes = append(changes, "status")
		}

		// Update type if provided
		if cmd.Flags().Changed("type") {
			if !cfg.IsValidType(updateType) {
				if updateJSON {
					return output.Error(output.ErrValidation, fmt.Sprintf("invalid type: %s (must be %s)", updateType, cfg.TypeList()))
				}
				return fmt.Errorf("invalid type: %s (must be %s)", updateType, cfg.TypeList())
			}
			b.Type = updateType
			changes = append(changes, "type")
		}

		// Update title if provided
		if cmd.Flags().Changed("title") {
			b.Title = updateTitle
			changes = append(changes, "title")
		}

		// Update body if provided
		if cmd.Flags().Changed("body") || cmd.Flags().Changed("body-file") {
			body, err := resolveContent(updateBody, updateBodyFile)
			if err != nil {
				if updateJSON {
					return output.Error(output.ErrFileError, err.Error())
				}
				return err
			}
			b.Body = body
			changes = append(changes, "body")
		}

		// Add links
		if len(updateLink) > 0 {
			linkWarnings, err := applyLinks(b, updateLink)
			if err != nil {
				if updateJSON {
					return output.Error(output.ErrValidation, err.Error())
				}
				return err
			}
			warnings = append(warnings, linkWarnings...)
			changes = append(changes, "links")
		}

		// Remove links
		if len(updateUnlink) > 0 {
			if err := removeLinks(b, updateUnlink); err != nil {
				if updateJSON {
					return output.Error(output.ErrValidation, err.Error())
				}
				return err
			}
			changes = append(changes, "links")
		}

		// Add tags
		if len(updateTag) > 0 {
			if err := applyTags(b, updateTag); err != nil {
				if updateJSON {
					return output.Error(output.ErrValidation, err.Error())
				}
				return err
			}
			changes = append(changes, "tags")
		}

		// Remove tags
		if len(updateUntag) > 0 {
			for _, tag := range updateUntag {
				b.RemoveTag(tag)
			}
			changes = append(changes, "tags")
		}

		// Check if anything was changed
		if len(changes) == 0 {
			if updateJSON {
				return output.Error(output.ErrValidation, "no changes specified")
			}
			return fmt.Errorf("no changes specified (use --status, --type, --title, --body, --link, --unlink, --tag, or --untag)")
		}

		// Save the bean
		if err := core.Update(b); err != nil {
			if updateJSON {
				return output.Error(output.ErrFileError, err.Error())
			}
			return fmt.Errorf("failed to save bean: %w", err)
		}

		// Output result
		if updateJSON {
			if len(warnings) > 0 {
				return output.SuccessWithWarnings(b, "Bean updated", warnings)
			}
			return output.Success(b, "Bean updated")
		}

		// Print warnings in text mode
		for _, w := range warnings {
			fmt.Println(ui.Warning.Render("Warning: ") + w)
		}

		fmt.Println(ui.Success.Render("Updated ") + ui.ID.Render(b.ID) + " " + ui.Muted.Render(b.Path))

		// Open in editor unless --no-edit or --json
		if !updateNoEdit && !updateJSON {
			editor := os.Getenv("EDITOR")
			if editor != "" {
				path := core.FullPath(b)
				editorCmd := exec.Command(editor, path)
				editorCmd.Stdin = os.Stdin
				editorCmd.Stdout = os.Stdout
				editorCmd.Stderr = os.Stderr
				if err := editorCmd.Run(); err != nil {
					return fmt.Errorf("editor failed: %w", err)
				}
			}
		}

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

	updateCmd.Flags().StringVarP(&updateStatus, "status", "s", "", "New status ("+strings.Join(statusNames, ", ")+")")
	updateCmd.Flags().StringVar(&updateType, "type", "", "New type ("+strings.Join(typeNames, ", ")+")")
	updateCmd.Flags().StringVarP(&updateTitle, "title", "t", "", "New title")
	updateCmd.Flags().StringVarP(&updateBody, "body", "d", "", "New body (use '-' to read from stdin)")
	updateCmd.Flags().StringVar(&updateBodyFile, "body-file", "", "Read body from file")
	updateCmd.Flags().StringArrayVar(&updateLink, "link", nil, "Add relationship (format: type:id, can be repeated)")
	updateCmd.Flags().StringArrayVar(&updateUnlink, "unlink", nil, "Remove relationship (format: type:id, can be repeated)")
	updateCmd.Flags().StringArrayVar(&updateTag, "tag", nil, "Add tag (can be repeated)")
	updateCmd.Flags().StringArrayVar(&updateUntag, "untag", nil, "Remove tag (can be repeated)")
	updateCmd.Flags().BoolVar(&updateNoEdit, "no-edit", false, "Skip opening $EDITOR")
	updateCmd.Flags().BoolVar(&updateJSON, "json", false, "Output as JSON")
	updateCmd.MarkFlagsMutuallyExclusive("body", "body-file")
	rootCmd.AddCommand(updateCmd)
}
