package cmd

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/graph"
	"github.com/hmans/beans/internal/output"
	"github.com/hmans/beans/internal/ui"
	"github.com/spf13/cobra"
)

var (
	showJSON     bool
	showRaw      bool
	showBodyOnly bool
)

var showCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show a bean's contents",
	Long:  `Displays the full contents of a bean, including front matter and body.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Query bean via GraphQL resolver
		resolver := &graph.Resolver{Core: core}
		b, err := resolver.Query().Bean(context.Background(), args[0])
		if err != nil {
			if showJSON {
				return output.Error(output.ErrNotFound, err.Error())
			}
			return fmt.Errorf("failed to find bean: %w", err)
		}
		if b == nil {
			if showJSON {
				return output.Error(output.ErrNotFound, "bean not found")
			}
			return fmt.Errorf("bean not found: %s", args[0])
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

		// Body only (no header, no styling)
		if showBodyOnly {
			fmt.Print(b.Body)
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
		if b.Priority != "" {
			priorityCfg := cfg.GetPriority(b.Priority)
			priorityColor := "gray"
			if priorityCfg != nil {
				priorityColor = priorityCfg.Color
			}
			header.WriteString(" ")
			header.WriteString(ui.RenderPriorityWithColor(b.Priority, priorityColor))
		}
		if len(b.Tags) > 0 {
			header.WriteString("  ")
			header.WriteString(ui.Muted.Render(strings.Join(b.Tags, ", ")))
		}
		header.WriteString("\n")
		header.WriteString(ui.Title.Render(b.Title))

		// Display relationships
		if len(b.Links) > 0 {
			header.WriteString("\n")
			header.WriteString(ui.Muted.Render(strings.Repeat("─", 50)))
			header.WriteString("\n")
			header.WriteString(formatLinks(b.Links))
		}

		header.WriteString("\n")
		header.WriteString(ui.Muted.Render(strings.Repeat("─", 50)))

		headerBox := lipgloss.NewStyle().
			MarginBottom(1).
			Render(header.String())

		fmt.Println(headerBox)

		// Render the body with Glamour
		if b.Body != "" {
			renderer, err := glamour.NewTermRenderer(
				glamour.WithAutoStyle(),
				glamour.WithWordWrap(80),
			)
			if err != nil {
				return fmt.Errorf("failed to create renderer: %w", err)
			}

			rendered, err := renderer.Render(b.Body)
			if err != nil {
				return fmt.Errorf("failed to render markdown: %w", err)
			}

			fmt.Print(rendered)
		}

		return nil
	},
}

// formatLinks formats links for display with consistent ordering.
func formatLinks(links bean.Links) string {
	if len(links) == 0 {
		return ""
	}

	// Group links by type, then sort types for deterministic output
	byType := make(map[string][]string)
	for _, link := range links {
		byType[link.Type] = append(byType[link.Type], link.Target)
	}

	types := make([]string, 0, len(byType))
	for t := range byType {
		types = append(types, t)
	}
	sort.Strings(types)

	var parts []string
	for _, linkType := range types {
		targets := byType[linkType]
		for _, target := range targets {
			parts = append(parts, fmt.Sprintf("%s %s",
				ui.Muted.Render(linkType+":"),
				ui.ID.Render(target)))
		}
	}
	return strings.Join(parts, "\n")
}

func init() {
	showCmd.Flags().BoolVar(&showJSON, "json", false, "Output as JSON")
	showCmd.Flags().BoolVar(&showRaw, "raw", false, "Output raw markdown without styling")
	showCmd.Flags().BoolVar(&showBodyOnly, "body-only", false, "Output only the body content")
	showCmd.MarkFlagsMutuallyExclusive("json", "raw", "body-only")
	rootCmd.AddCommand(showCmd)
}
