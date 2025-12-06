package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"hmans.dev/beans/internal/bean"
	"hmans.dev/beans/internal/output"
	"hmans.dev/beans/internal/ui"
)

var (
	listJSON   bool
	listStatus []string
	listQuiet  bool
	listSort   string
	listFull   bool
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all beans",
	Long:    `Lists all beans in the .beans directory.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		beans, err := store.FindAll()
		if err != nil {
			if listJSON {
				return output.Error(output.ErrFileError, err.Error())
			}
			return fmt.Errorf("failed to list beans: %w", err)
		}

		// Apply filters
		beans = filterBeans(beans, listStatus)

		// Sort beans
		sortBeans(beans, listSort, cfg.StatusNames())

		// JSON output
		if listJSON {
			if !listFull {
				for _, b := range beans {
					b.Description = ""
				}
			}
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

		// Calculate max ID width
		maxIDWidth := 2 // minimum for "ID" header
		for _, b := range beans {
			if len(b.ID) > maxIDWidth {
				maxIDWidth = len(b.ID)
			}
		}
		maxIDWidth += 2 // padding

		// Column styles with widths for alignment
		idStyle := lipgloss.NewStyle().Width(maxIDWidth)
		statusStyle := lipgloss.NewStyle().Width(14)
		typeStyle := lipgloss.NewStyle().Width(12)
		titleStyle := lipgloss.NewStyle()

		// Header style
		headerCol := lipgloss.NewStyle().Foreground(ui.ColorMuted)

		// Header
		header := lipgloss.JoinHorizontal(lipgloss.Top,
			idStyle.Render(headerCol.Render("ID")),
			statusStyle.Render(headerCol.Render("STATUS")),
			typeStyle.Render(headerCol.Render("TYPE")),
			titleStyle.Render(headerCol.Render("TITLE")),
		)
		fmt.Println(header)
		fmt.Println(ui.Muted.Render(strings.Repeat("─", maxIDWidth+14+12+30)))

		for _, b := range beans {
			// Get status color from config
			statusCfg := cfg.GetStatus(b.Status)
			statusColor := "gray"
			if statusCfg != nil {
				statusColor = statusCfg.Color
			}
			isArchive := cfg.IsArchiveStatus(b.Status)

			// Get type color from config
			typeColor := ""
			if typeCfg := cfg.GetType(b.Type); typeCfg != nil {
				typeColor = typeCfg.Color
			}

			row := lipgloss.JoinHorizontal(lipgloss.Top,
				idStyle.Render(ui.ID.Render(b.ID)),
				statusStyle.Render(ui.RenderStatusTextWithColor(b.Status, statusColor, isArchive)),
				typeStyle.Render(ui.RenderTypeText(b.Type, typeColor)),
				titleStyle.Render(truncate(b.Title, 50)),
			)
			fmt.Println(row)
		}

		return nil
	},
}

func sortBeans(beans []*bean.Bean, sortBy string, statusNames []string) {
	switch sortBy {
	case "created":
		sort.Slice(beans, func(i, j int) bool {
			if beans[i].CreatedAt == nil && beans[j].CreatedAt == nil {
				return beans[i].ID < beans[j].ID
			}
			if beans[i].CreatedAt == nil {
				return false
			}
			if beans[j].CreatedAt == nil {
				return true
			}
			return beans[i].CreatedAt.After(*beans[j].CreatedAt)
		})
	case "updated":
		sort.Slice(beans, func(i, j int) bool {
			if beans[i].UpdatedAt == nil && beans[j].UpdatedAt == nil {
				return beans[i].ID < beans[j].ID
			}
			if beans[i].UpdatedAt == nil {
				return false
			}
			if beans[j].UpdatedAt == nil {
				return true
			}
			return beans[i].UpdatedAt.After(*beans[j].UpdatedAt)
		})
	case "status":
		// Build status order from configured statuses
		statusOrder := make(map[string]int)
		for i, s := range statusNames {
			statusOrder[s] = i
		}
		sort.Slice(beans, func(i, j int) bool {
			oi, oj := statusOrder[beans[i].Status], statusOrder[beans[j].Status]
			if oi != oj {
				return oi < oj
			}
			return beans[i].ID < beans[j].ID
		})
	default:
		// Default: sort by ID
		sort.Slice(beans, func(i, j int) bool {
			return beans[i].ID < beans[j].ID
		})
	}
}

func filterBeans(beans []*bean.Bean, statuses []string) []*bean.Bean {
	if len(statuses) == 0 {
		return beans
	}

	var filtered []*bean.Bean
	for _, b := range beans {
		// Filter by status
		matched := false
		for _, s := range statuses {
			if b.Status == s {
				matched = true
				break
			}
		}
		if matched {
			filtered = append(filtered, b)
		}
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
	listCmd.Flags().BoolVarP(&listQuiet, "quiet", "q", false, "Only output IDs (one per line)")
	listCmd.Flags().StringVar(&listSort, "sort", "status", "Sort by: created, updated, status, id (default: status)")
	listCmd.Flags().BoolVar(&listFull, "full", false, "Include bean description in JSON output")
	listCmd.MarkFlagsMutuallyExclusive("json", "quiet")
	rootCmd.AddCommand(listCmd)
}
