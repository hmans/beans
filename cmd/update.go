package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"hmans.dev/beans/internal/bean"
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
  --link         Add a relationship (format: type:id)
  --unlink       Remove a relationship (format: type:id)

Relationship types: blocks, duplicates, parent, relates-to`,
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
			for _, link := range updateLink {
				linkType, targetID, err := parseLink(link)
				if err != nil {
					if updateJSON {
						return output.Error(output.ErrValidation, err.Error())
					}
					return err
				}
				if !isKnownLinkType(linkType) {
					errMsg := fmt.Sprintf("unknown link type: %s (must be blocks, duplicates, parent, relates-to)", linkType)
					if updateJSON {
						return output.Error(output.ErrValidation, errMsg)
					}
					return fmt.Errorf("%s", errMsg)
				}
				b.Links = b.Links.Add(linkType, targetID)
			}
			changes = append(changes, "links")
		}

		// Remove links
		if len(updateUnlink) > 0 {
			for _, link := range updateUnlink {
				linkType, targetID, err := parseLink(link)
				if err != nil {
					if updateJSON {
						return output.Error(output.ErrValidation, err.Error())
					}
					return err
				}
				b.Links = b.Links.Remove(linkType, targetID)
			}
			changes = append(changes, "links")
		}

		// Check if anything was changed
		if len(changes) == 0 {
			if updateJSON {
				return output.Error(output.ErrValidation, "no changes specified")
			}
			return fmt.Errorf("no changes specified (use --status, --type, --title, --body, --link, or --unlink)")
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
	updateCmd.Flags().StringVarP(&updateBody, "body", "d", "", "New body (use '-' to read from stdin)")
	updateCmd.Flags().StringVar(&updateBodyFile, "body-file", "", "Read body from file")
	updateCmd.Flags().StringArrayVar(&updateLink, "link", nil, "Add relationship (format: type:id)")
	updateCmd.Flags().StringArrayVar(&updateUnlink, "unlink", nil, "Remove relationship (format: type:id)")
	updateCmd.Flags().BoolVar(&updateNoEdit, "no-edit", false, "Skip opening $EDITOR")
	updateCmd.Flags().BoolVar(&updateJSON, "json", false, "Output as JSON")
	updateCmd.MarkFlagsMutuallyExclusive("body", "body-file")
	rootCmd.AddCommand(updateCmd)
}

// parseLink parses a link in the format "type:id".
func parseLink(s string) (linkType, targetID string, err error) {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid link format: %q (expected type:id)", s)
	}
	return parts[0], parts[1], nil
}

// isKnownLinkType checks if a link type is recognized.
func isKnownLinkType(linkType string) bool {
	for _, t := range bean.KnownLinkTypes {
		if t == linkType {
			return true
		}
	}
	return false
}
