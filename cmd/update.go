package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"hmans.dev/beans/internal/output"
	"hmans.dev/beans/internal/ui"
)

var (
	updateStatus          string
	updateType            string
	updateTitle           string
	updateDescription     string
	updateDescriptionFile string
	updateNoEdit          bool
	updateJSON            bool
)

var updateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a bean's properties",
	Long: `Updates one or more properties of an existing bean.

Use flags to specify which properties to update:
  --status        Change the status
  --type          Change the type
  --title         Change the title
  --description   Change the description (use '-' to read from stdin)`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		// Find the bean
		b, err := store.FindByID(id)
		if err != nil {
			if updateJSON {
				return output.Error(output.ErrNotFound, err.Error())
			}
			return fmt.Errorf("failed to find bean: %w", err)
		}

		// Track what changed for output message
		var changes []string

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

		// Update description if provided
		if cmd.Flags().Changed("description") || cmd.Flags().Changed("description-file") {
			body, err := resolveContent(updateDescription, updateDescriptionFile)
			if err != nil {
				if updateJSON {
					return output.Error(output.ErrFileError, err.Error())
				}
				return err
			}
			b.Body = body
			changes = append(changes, "description")
		}

		// Check if anything was changed
		if len(changes) == 0 {
			if updateJSON {
				return output.Error(output.ErrValidation, "no changes specified")
			}
			return fmt.Errorf("no changes specified (use --status, --type, --title, or --description)")
		}

		// Save the bean
		if err := store.Save(b); err != nil {
			if updateJSON {
				return output.Error(output.ErrFileError, err.Error())
			}
			return fmt.Errorf("failed to save bean: %w", err)
		}

		// Output result
		if updateJSON {
			return output.Success(b, "Bean updated")
		}

		fmt.Println(ui.Success.Render("Updated ") + ui.ID.Render(b.ID) + " " + ui.Muted.Render(b.Path))

		// Open in editor unless --no-edit or --json
		if !updateNoEdit && !updateJSON {
			editor := os.Getenv("EDITOR")
			if editor != "" {
				path := store.FullPath(b)
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
	updateCmd.Flags().StringVarP(&updateStatus, "status", "s", "", "New status")
	updateCmd.Flags().StringVar(&updateType, "type", "", "New type (e.g., task, bug, epic)")
	updateCmd.Flags().StringVarP(&updateTitle, "title", "t", "", "New title")
	updateCmd.Flags().StringVarP(&updateDescription, "description", "d", "", "New description (use '-' to read from stdin)")
	updateCmd.Flags().StringVar(&updateDescriptionFile, "description-file", "", "Read description from file")
	updateCmd.Flags().BoolVar(&updateNoEdit, "no-edit", false, "Skip opening $EDITOR")
	updateCmd.Flags().BoolVar(&updateJSON, "json", false, "Output as JSON")
	updateCmd.MarkFlagsMutuallyExclusive("description", "description-file")
	rootCmd.AddCommand(updateCmd)
}
