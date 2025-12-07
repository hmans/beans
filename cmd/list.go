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
	listLinked []string
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
		beans = filterByLinked(beans, listLinked)

		// Sort beans
		sortBeans(beans, listSort, cfg.StatusNames())

		// JSON output
		if listJSON {
			if !listFull {
				for _, b := range beans {
					b.Body = ""
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

// filterByLinked filters beans by incoming relationship.
// Supports two formats:
//   - "type:id" - Returns beans that the specified bean (id) has in its links[type]
//   - "type" - Returns beans that ANY bean has in its links[type]
//
// Multiple values can be comma-separated or specified via repeated flags.
//
// Examples:
//   - --linked blocks:A returns beans that A blocks
//   - --linked blocks returns all beans that are blocked by something
//   - --linked blocks,parent returns beans that are blocked OR have a parent
func filterByLinked(beans []*bean.Bean, linked []string) []*bean.Bean {
	if len(linked) == 0 {
		return beans
	}

	// Expand comma-separated values
	var expandedLinked []string
	for _, l := range linked {
		for _, part := range strings.Split(l, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				expandedLinked = append(expandedLinked, part)
			}
		}
	}
	linked = expandedLinked

	// Build ID → Bean lookup for source beans
	byID := make(map[string]*bean.Bean)
	for _, b := range beans {
		byID[b.ID] = b
	}

	// Build set of all beans targeted by each link type (for type-only queries)
	targetedBy := make(map[string]map[string]bool) // linkType -> set of target IDs
	for _, b := range beans {
		for linkType, ids := range b.Links {
			if targetedBy[linkType] == nil {
				targetedBy[linkType] = make(map[string]bool)
			}
			for _, id := range ids {
				targetedBy[linkType][id] = true
			}
		}
	}

	var filtered []*bean.Bean
	for _, b := range beans {
		matched := false
		for _, link := range linked {
			parts := strings.SplitN(link, ":", 2)
			linkType := parts[0]

			if len(parts) == 1 {
				// Type-only: check if this bean is targeted by ANY bean with this link type
				if targets, ok := targetedBy[linkType]; ok && targets[b.ID] {
					matched = true
				}
			} else {
				// Type:ID: check if specific source bean has this bean in its links
				sourceID := parts[1]
				source, exists := byID[sourceID]
				if !exists {
					continue // Source bean not found
				}

				if ids, ok := source.Links[linkType]; ok {
					for _, id := range ids {
						if id == b.ID {
							matched = true
							break
						}
					}
				}
			}

			if matched {
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
	listCmd.Flags().StringArrayVar(&listLinked, "linked", nil, "Filter by relationship (format: type:id)")
	listCmd.Flags().BoolVarP(&listQuiet, "quiet", "q", false, "Only output IDs (one per line)")
	listCmd.Flags().StringVar(&listSort, "sort", "status", "Sort by: created, updated, status, id (default: status)")
	listCmd.Flags().BoolVar(&listFull, "full", false, "Include bean body in JSON output")
	listCmd.MarkFlagsMutuallyExclusive("json", "quiet")
	rootCmd.AddCommand(listCmd)
}
