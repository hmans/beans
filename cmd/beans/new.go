package beans

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"hmans.dev/beans/internal/bean"
	"hmans.dev/beans/internal/output"
	"hmans.dev/beans/internal/ui"
)

var (
	newStatus   string
	newBody     string
	newBodyFile string
	newNoEdit   bool
	newPath     string
	newJSON     bool
)

var validStatuses = map[string]bool{
	"open":        true,
	"in-progress": true,
	"closed":      true,
}

var newCmd = &cobra.Command{
	Use:   "new [title]",
	Short: "Create a new bean",
	Long:  `Creates a new bean (issue) with a generated ID and optional title.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		title := strings.Join(args, " ")
		status := newStatus

		// Validate status if provided
		if status != "" && !validStatuses[status] {
			if newJSON {
				return output.Error(output.ErrInvalidStatus, fmt.Sprintf("invalid status: %s (must be open, in-progress, or closed)", status))
			}
			return fmt.Errorf("invalid status: %s (must be open, in-progress, or closed)", status)
		}
		if status == "" {
			status = "open"
		}

		// Determine body content
		body, err := resolveBody(newBody, newBodyFile)
		if err != nil {
			if newJSON {
				return output.Error(output.ErrFileError, err.Error())
			}
			return err
		}

		// Check if we're in scripting mode (any flag that suggests non-interactive use)
		scriptingMode := newBody != "" || newBodyFile != "" || newJSON || newNoEdit || cmd.Flags().Changed("status")

		// If no title provided and not in scripting mode, show interactive form
		if title == "" && !scriptingMode {
			form := huh.NewForm(
				huh.NewGroup(
					huh.NewInput().
						Title("Title").
						Description("What's this bean about?").
						Placeholder("Bug: login fails on Safari").
						Value(&title),
					huh.NewSelect[string]().
						Title("Status").
						Options(
							huh.NewOption("Open", "open"),
							huh.NewOption("In Progress", "in-progress"),
							huh.NewOption("Closed", "closed"),
						).
						Value(&status),
				),
			)

			if err := form.Run(); err != nil {
				return err
			}
		}

		if title == "" {
			title = "Untitled"
		}

		b := &bean.Bean{
			ID:     bean.NewID(),
			Slug:   bean.Slugify(title),
			Title:  title,
			Status: status,
			Body:   body,
		}

		// Set path if provided
		if newPath != "" {
			b.Path = newPath + "/" + b.ID + "-" + b.Slug + ".md"
		}

		if err := store.Save(b); err != nil {
			if newJSON {
				return output.Error(output.ErrFileError, err.Error())
			}
			return fmt.Errorf("failed to create bean: %w", err)
		}

		// Output result
		if newJSON {
			return output.Success(b, "Bean created")
		}

		fmt.Println(ui.Success.Render("Created ") + ui.ID.Render(b.ID) + " " + ui.Muted.Render(b.Path))

		// Open in editor unless --no-edit or --json
		if !newNoEdit && !newJSON {
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

// resolveBody returns the body content from --body or --body-file flags.
// If --body is "-", reads from stdin.
func resolveBody(body, bodyFile string) (string, error) {
	if body != "" && bodyFile != "" {
		return "", fmt.Errorf("cannot use both --body and --body-file")
	}

	if body == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("reading stdin: %w", err)
		}
		return string(data), nil
	}

	if body != "" {
		return body, nil
	}

	if bodyFile != "" {
		data, err := os.ReadFile(bodyFile)
		if err != nil {
			return "", fmt.Errorf("reading body file: %w", err)
		}
		return string(data), nil
	}

	return "", nil
}

func init() {
	newCmd.Flags().StringVarP(&newStatus, "status", "s", "", "Initial status (open, in-progress, closed)")
	newCmd.Flags().StringVarP(&newBody, "body", "b", "", "Body content (use '-' to read from stdin)")
	newCmd.Flags().StringVar(&newBodyFile, "body-file", "", "Read body from file")
	newCmd.Flags().BoolVar(&newNoEdit, "no-edit", false, "Skip opening $EDITOR")
	newCmd.Flags().StringVarP(&newPath, "path", "p", "", "Subdirectory within .beans/")
	newCmd.Flags().BoolVar(&newJSON, "json", false, "Output as JSON")
	newCmd.MarkFlagsMutuallyExclusive("body", "body-file")
	rootCmd.AddCommand(newCmd)
}
