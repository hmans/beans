package cmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"hmans.dev/beans/internal/output"
	"hmans.dev/beans/internal/ui"
)

var (
	showJSON            bool
	showRaw             bool
	showDescriptionOnly bool
)

var showCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show a bean's contents",
	Long:  `Displays the full contents of a bean, including front matter and description.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		b, err := store.FindByID(args[0])
		if err != nil {
			if showJSON {
				return output.Error(output.ErrNotFound, err.Error())
			}
			return fmt.Errorf("failed to find bean: %w", err)
		}

		// JSON output
		if showJSON {
			return output.Success(b, "")
		}

		// Raw markdown output (frontmatter + body)
		if showRaw {
			content, err := b.Render()
			if err != nil {
				return fmt.Errorf("failed to render bean: %w", err)
			}
			fmt.Print(string(content))
			return nil
		}

		// Description only (no header, no styling)
		if showDescriptionOnly {
			fmt.Print(b.Description)
			return nil
		}

		// Default: styled human-friendly output
		statusCfg := cfg.GetStatus(b.Status)
		statusColor := "gray"
		if statusCfg != nil {
			statusColor = statusCfg.Color
		}
		isArchive := cfg.IsArchiveStatus(b.Status)

		var header strings.Builder
		header.WriteString(ui.ID.Render(b.ID))
		header.WriteString(" ")
		header.WriteString(ui.RenderStatusWithColor(b.Status, statusColor, isArchive))
		header.WriteString("\n")
		header.WriteString(ui.Title.Render(b.Title))
		header.WriteString("\n")
		header.WriteString(ui.Muted.Render(strings.Repeat("─", 50)))

		headerBox := lipgloss.NewStyle().
			MarginBottom(1).
			Render(header.String())

		fmt.Println(headerBox)

		// Render the description with Glamour
		if b.Description != "" {
			renderer, err := glamour.NewTermRenderer(
				glamour.WithAutoStyle(),
				glamour.WithWordWrap(80),
			)
			if err != nil {
				return fmt.Errorf("failed to create renderer: %w", err)
			}

			rendered, err := renderer.Render(b.Description)
			if err != nil {
				return fmt.Errorf("failed to render markdown: %w", err)
			}

			fmt.Print(rendered)
		}

		return nil
	},
}

func init() {
	showCmd.Flags().BoolVar(&showJSON, "json", false, "Output as JSON")
	showCmd.Flags().BoolVar(&showRaw, "raw", false, "Output raw markdown without styling")
	showCmd.Flags().BoolVar(&showDescriptionOnly, "description-only", false, "Output only the description content")
	showCmd.MarkFlagsMutuallyExclusive("json", "raw", "description-only")
	rootCmd.AddCommand(showCmd)
}
