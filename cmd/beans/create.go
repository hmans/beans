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
	createStatus          string
	createDescription     string
	createDescriptionFile string
	createNoEdit          bool
	createPath            string
	createJSON            bool
)

var createCmd = &cobra.Command{
	Use:   "create [title]",
	Short: "Create a new bean",
	Long:  `Creates a new bean (issue) with a generated ID and optional title.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		title := strings.Join(args, " ")
		status := createStatus

		// Validate status if provided
		if status != "" && !cfg.IsValidStatus(status) {
			if createJSON {
				return output.Error(output.ErrInvalidStatus, fmt.Sprintf("invalid status: %s (must be %s)", status, cfg.StatusList()))
			}
			return fmt.Errorf("invalid status: %s (must be %s)", status, cfg.StatusList())
		}
		if status == "" {
			status = cfg.GetDefaultStatus()
		}

		// Determine description content
		description, err := resolveContent(createDescription, createDescriptionFile)
		if err != nil {
			if createJSON {
				return output.Error(output.ErrFileError, err.Error())
			}
			return err
		}

		// Check if we're in scripting mode (any flag that suggests non-interactive use)
		scriptingMode := createDescription != "" || createDescriptionFile != "" || createJSON || createNoEdit || cmd.Flags().Changed("status")

		// If no title provided and not in scripting mode, show interactive form
		if title == "" && !scriptingMode {
			// Build status options
			var statusOptions []huh.Option[string]
			for _, s := range cfg.StatusNames() {
				statusOptions = append(statusOptions, huh.NewOption(formatStatusLabel(s), s))
			}

			form := huh.NewForm(
				huh.NewGroup(
					huh.NewInput().
						Title("Title").
						Description("What's this bean about?").
						Placeholder("Bug: login fails on Safari").
						Value(&title),
					huh.NewSelect[string]().
						Title("Status").
						Options(statusOptions...).
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
			ID:     bean.NewID(cfg.Beans.Prefix, cfg.Beans.IDLength),
			Slug:   bean.Slugify(title),
			Title:  title,
			Status: status,
			Body:   description,
		}

		// Set path if provided
		if createPath != "" {
			b.Path = createPath + "/" + bean.BuildFilename(b.ID, b.Slug)
		}

		if err := store.Save(b); err != nil {
			if createJSON {
				return output.Error(output.ErrFileError, err.Error())
			}
			return fmt.Errorf("failed to create bean: %w", err)
		}

		// Output result
		if createJSON {
			return output.Success(b, "Bean created")
		}

		fmt.Println(ui.Success.Render("Created ") + ui.ID.Render(b.ID) + " " + ui.Muted.Render(b.Path))

		// Open in editor unless --no-edit or --json
		if !createNoEdit && !createJSON {
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

// resolveContent returns content from a direct value or file flag.
// If value is "-", reads from stdin.
func resolveContent(value, file string) (string, error) {
	if value != "" && file != "" {
		return "", fmt.Errorf("cannot use both --description and --description-file")
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

// formatStatusLabel converts a status value to a display label.
// e.g., "in-progress" -> "In Progress", "open" -> "Open"
func formatStatusLabel(status string) string {
	words := strings.Split(status, "-")
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}
	return strings.Join(words, " ")
}

func init() {
	createCmd.Flags().StringVarP(&createStatus, "status", "s", "", "Initial status (open, in-progress, done)")
	createCmd.Flags().StringVarP(&createDescription, "description", "d", "", "Description content (use '-' to read from stdin)")
	createCmd.Flags().StringVar(&createDescriptionFile, "description-file", "", "Read description from file")
	createCmd.Flags().BoolVar(&createNoEdit, "no-edit", false, "Skip opening $EDITOR")
	createCmd.Flags().StringVarP(&createPath, "path", "p", "", "Subdirectory within .beans/")
	createCmd.Flags().BoolVar(&createJSON, "json", false, "Output as JSON")
	createCmd.MarkFlagsMutuallyExclusive("description", "description-file")
	rootCmd.AddCommand(createCmd)
}
