package beans

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"hmans.dev/beans/internal/bean"
	"hmans.dev/beans/internal/output"
	"hmans.dev/beans/internal/ui"
)

var (
	listJSON     bool
	listStatus   []string
	listPath     string
	listQuiet    bool
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all beans",
	Long:    `Lists all beans in the .beans directory, grouped by path.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		beans, err := store.FindAll()
		if err != nil {
			if listJSON {
				return output.Error(output.ErrFileError, err.Error())
			}
			return fmt.Errorf("failed to list beans: %w", err)
		}

		// Apply filters
		beans = filterBeans(beans, listStatus, listPath)

		// Sort by path for grouping
		sort.Slice(beans, func(i, j int) bool {
			return beans[i].Path < beans[j].Path
		})

		// JSON output
		if listJSON {
			return output.SuccessMultiple(beans)
		}

		// Quiet mode: just IDs
		if listQuiet {
			for _, b := range beans {
				fmt.Println(b.ID)
			}
			return nil
		}

		// Human-friendly output
		if len(beans) == 0 {
			fmt.Println(ui.Muted.Render("No beans found. Create one with: beans new <title>"))
			return nil
		}

		// Column styles with fixed widths for alignment
		idStyle := lipgloss.NewStyle().Width(10)
		statusStyle := lipgloss.NewStyle().Width(14)
		pathStyle := lipgloss.NewStyle().Width(20).Foreground(ui.ColorMuted)
		titleStyle := lipgloss.NewStyle()

		// Header style
		headerCol := lipgloss.NewStyle().Foreground(ui.ColorMuted)

		// Header
		header := lipgloss.JoinHorizontal(lipgloss.Top,
			idStyle.Render(headerCol.Render("ID")),
			statusStyle.Render(headerCol.Render("STATUS")),
			pathStyle.Render(headerCol.Render("PATH")),
			titleStyle.Render(headerCol.Render("TITLE")),
		)
		fmt.Println(header)
		fmt.Println(ui.Muted.Render(strings.Repeat("─", 70)))

		for _, b := range beans {
			dir := filepath.Dir(b.Path)
			if dir == "." {
				dir = ""
			}

			row := lipgloss.JoinHorizontal(lipgloss.Top,
				idStyle.Render(ui.ID.Render(b.ID)),
				statusStyle.Render(ui.RenderStatusText(b.Status)),
				pathStyle.Render(truncate(dir, 18)),
				titleStyle.Render(truncate(b.Title, 50)),
			)
			fmt.Println(row)
		}

		return nil
	},
}

func filterBeans(beans []*bean.Bean, statuses []string, pathPrefix string) []*bean.Bean {
	if len(statuses) == 0 && pathPrefix == "" {
		return beans
	}

	var filtered []*bean.Bean
	for _, b := range beans {
		// Filter by status
		if len(statuses) > 0 {
			matched := false
			for _, s := range statuses {
				if b.Status == s {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		// Filter by path prefix
		if pathPrefix != "" && !strings.HasPrefix(b.Path, pathPrefix) {
			continue
		}

		filtered = append(filtered, b)
	}
	return filtered
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func init() {
	listCmd.Flags().BoolVar(&listJSON, "json", false, "Output as JSON")
	listCmd.Flags().StringArrayVarP(&listStatus, "status", "s", nil, "Filter by status (can be repeated)")
	listCmd.Flags().StringVarP(&listPath, "path", "p", "", "Filter by path prefix")
	listCmd.Flags().BoolVarP(&listQuiet, "quiet", "q", false, "Only output IDs (one per line)")
	listCmd.MarkFlagsMutuallyExclusive("json", "quiet")
	rootCmd.AddCommand(listCmd)
}
