package beans

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"hmans.dev/beans/internal/bean"
	"hmans.dev/beans/internal/output"
	"hmans.dev/beans/internal/ui"
)

var (
	editSetTitle    string
	editSetStatus   string
	editSetBody     string
	editSetBodyFile string
	editAppendBody  string
	editJSON        bool
)

var editCmd = &cobra.Command{
	Use:   "edit <id>",
	Short: "Edit a bean in your editor",
	Long: `Opens a bean in your $EDITOR for editing.

With --set-* flags, modifies the bean programmatically without opening an editor.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		b, err := store.FindByID(args[0])
		if err != nil {
			if editJSON {
				return output.Error(output.ErrNotFound, err.Error())
			}
			return fmt.Errorf("failed to find bean: %w", err)
		}

		// Check if any --set-* flag was provided
		hasSetFlags := editSetTitle != "" || editSetStatus != "" ||
			editSetBody != "" || editSetBodyFile != "" || editAppendBody != ""

		if hasSetFlags {
			return editProgrammatically(b)
		}

		// Default: open in editor
		editor := os.Getenv("EDITOR")
		if editor == "" {
			if editJSON {
				return output.Error(output.ErrValidation, "$EDITOR not set")
			}
			return fmt.Errorf("$EDITOR not set")
		}

		path := store.FullPath(b)
		editorCmd := exec.Command(editor, path)
		editorCmd.Stdin = os.Stdin
		editorCmd.Stdout = os.Stdout
		editorCmd.Stderr = os.Stderr

		if err := editorCmd.Run(); err != nil {
			return fmt.Errorf("editor failed: %w", err)
		}

		return nil
	},
}

func editProgrammatically(b *bean.Bean) error {
	var changes []string

	// Apply title change
	if editSetTitle != "" {
		b.Title = editSetTitle
		b.Slug = bean.Slugify(editSetTitle)
		changes = append(changes, "title")
	}

	// Apply status change
	if editSetStatus != "" {
		if !validStatuses[editSetStatus] {
			if editJSON {
				return output.Error(output.ErrInvalidStatus,
					fmt.Sprintf("invalid status: %s (must be open, in-progress, or closed)", editSetStatus))
			}
			return fmt.Errorf("invalid status: %s (must be open, in-progress, or closed)", editSetStatus)
		}
		b.Status = editSetStatus
		changes = append(changes, "status")
	}

	// Apply body change
	if editSetBody != "" || editSetBodyFile != "" {
		body, err := resolveBody(editSetBody, editSetBodyFile)
		if err != nil {
			if editJSON {
				return output.Error(output.ErrFileError, err.Error())
			}
			return err
		}
		b.Body = body
		changes = append(changes, "body")
	}

	// Apply body append
	if editAppendBody != "" {
		appendContent := editAppendBody
		if appendContent == "-" {
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				if editJSON {
					return output.Error(output.ErrFileError, fmt.Sprintf("reading stdin: %s", err))
				}
				return fmt.Errorf("reading stdin: %w", err)
			}
			appendContent = string(data)
		}

		if b.Body != "" && !strings.HasSuffix(b.Body, "\n") {
			b.Body += "\n"
		}
		b.Body += appendContent
		changes = append(changes, "body")
	}

	// Save the bean
	if err := store.Save(b); err != nil {
		if editJSON {
			return output.Error(output.ErrFileError, err.Error())
		}
		return fmt.Errorf("failed to save bean: %w", err)
	}

	// Output result
	if editJSON {
		return output.Success(b, fmt.Sprintf("Updated %s", strings.Join(changes, ", ")))
	}

	fmt.Println(ui.Success.Render("Updated ") + ui.ID.Render(b.ID) +
		" " + ui.Muted.Render("("+strings.Join(changes, ", ")+")"))
	return nil
}

func init() {
	editCmd.Flags().StringVar(&editSetTitle, "set-title", "", "Set new title")
	editCmd.Flags().StringVar(&editSetStatus, "set-status", "", "Set new status (open, in-progress, closed)")
	editCmd.Flags().StringVar(&editSetBody, "set-body", "", "Replace body content (use '-' for stdin)")
	editCmd.Flags().StringVar(&editSetBodyFile, "set-body-file", "", "Replace body from file")
	editCmd.Flags().StringVar(&editAppendBody, "append-body", "", "Append to body (use '-' for stdin)")
	editCmd.Flags().BoolVar(&editJSON, "json", false, "Output as JSON")
	editCmd.MarkFlagsMutuallyExclusive("set-body", "set-body-file", "append-body")
	rootCmd.AddCommand(editCmd)
}
