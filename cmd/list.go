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
	listJSON       bool
	listStatus     []string
	listLinks      []string
	listLinkedAs   []string
	listNoLinks    []string
	listNoLinkedAs []string
	listQuiet      bool
	listSort       string
	listFull       bool
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

		// Apply filters (positive first, then exclusions)
		beans = filterBeans(beans, listStatus)
		beans = filterByLinks(beans, listLinks)
		beans = filterByLinkedAs(beans, listLinkedAs)
		beans = excludeByLinks(beans, listNoLinks)
		beans = excludeByLinkedAs(beans, listNoLinkedAs)

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

// filterByLinks filters beans by outgoing relationship.
// Supports two formats:
//   - "type:id" - Returns beans that have id in their links[type]
//   - "type" - Returns beans that have ANY link of this type
//
// Use repeated flags for multiple values (OR logic).
//
// Examples:
//   - --links blocks:A returns beans that block A
//   - --links blocks returns all beans that block something
//   - --links blocks --links parent returns beans that block something OR have a parent link
func filterByLinks(beans []*bean.Bean, link []string) []*bean.Bean {
	if len(link) == 0 {
		return beans
	}

	var filtered []*bean.Bean
	for _, b := range beans {
		matched := false
		for _, l := range link {
			parts := strings.SplitN(l, ":", 2)
			linkType := parts[0]

			if len(parts) == 1 {
				// Type-only: check if this bean has ANY link of this type
				if b.Links.HasType(linkType) {
					matched = true
				}
			} else {
				// Type:ID: check if this bean links to the specific target
				targetID := parts[1]
				if b.Links.HasLink(linkType, targetID) {
					matched = true
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

// filterByLinkedAs filters beans by incoming relationship.
// Supports two formats:
//   - "type:id" - Returns beans that the specified bean (id) has in its links[type]
//   - "type" - Returns beans that ANY bean has in its links[type]
//
// Use repeated flags for multiple values (OR logic).
//
// Examples:
//   - --linked-as blocks:A returns beans that A blocks
//   - --linked-as blocks returns all beans that are blocked by something
//   - --linked-as blocks --linked-as parent returns beans that are blocked OR have a parent
func filterByLinkedAs(beans []*bean.Bean, linked []string) []*bean.Bean {
	if len(linked) == 0 {
		return beans
	}

	// Build ID → Bean lookup for source beans
	byID := make(map[string]*bean.Bean)
	for _, b := range beans {
		byID[b.ID] = b
	}

	// Build set of all beans targeted by each link type (for type-only queries)
	targetedBy := make(map[string]map[string]bool) // linkType -> set of target IDs
	for _, b := range beans {
		for _, link := range b.Links {
			if targetedBy[link.Type] == nil {
				targetedBy[link.Type] = make(map[string]bool)
			}
			targetedBy[link.Type][link.Target] = true
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

				if source.Links.HasLink(linkType, b.ID) {
					matched = true
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

// excludeByLinks excludes beans by outgoing relationship.
// Inverse of filterByLinks: returns beans that DON'T match the criteria.
//
// Examples:
//   - --no-links blocks returns beans that don't block anything
//   - --no-links parent returns beans without a parent link
func excludeByLinks(beans []*bean.Bean, exclude []string) []*bean.Bean {
	if len(exclude) == 0 {
		return beans
	}

	var filtered []*bean.Bean
	for _, b := range beans {
		excluded := false
		for _, l := range exclude {
			parts := strings.SplitN(l, ":", 2)
			linkType := parts[0]

			if len(parts) == 1 {
				// Type-only: exclude if this bean has ANY link of this type
				if b.Links.HasType(linkType) {
					excluded = true
				}
			} else {
				// Type:ID: exclude if this bean links to the specific target
				targetID := parts[1]
				if b.Links.HasLink(linkType, targetID) {
					excluded = true
				}
			}

			if excluded {
				break
			}
		}
		if !excluded {
			filtered = append(filtered, b)
		}
	}
	return filtered
}

// excludeByLinkedAs excludes beans by incoming relationship.
// Inverse of filterByLinkedAs: returns beans that DON'T match the criteria.
//
// Examples:
//   - --no-linked-as blocks returns beans not blocked by anything (actionable work)
//   - --no-linked-as parent:epic-123 returns beans that are not children of epic-123
func excludeByLinkedAs(beans []*bean.Bean, exclude []string) []*bean.Bean {
	if len(exclude) == 0 {
		return beans
	}

	// Build ID → Bean lookup for source beans
	byID := make(map[string]*bean.Bean)
	for _, b := range beans {
		byID[b.ID] = b
	}

	// Build set of all beans targeted by each link type (for type-only queries)
	targetedBy := make(map[string]map[string]bool) // linkType -> set of target IDs
	for _, b := range beans {
		for _, link := range b.Links {
			if targetedBy[link.Type] == nil {
				targetedBy[link.Type] = make(map[string]bool)
			}
			targetedBy[link.Type][link.Target] = true
		}
	}

	var filtered []*bean.Bean
	for _, b := range beans {
		excluded := false
		for _, link := range exclude {
			parts := strings.SplitN(link, ":", 2)
			linkType := parts[0]

			if len(parts) == 1 {
				// Type-only: exclude if this bean is targeted by ANY bean with this link type
				if targets, ok := targetedBy[linkType]; ok && targets[b.ID] {
					excluded = true
				}
			} else {
				// Type:ID: exclude if specific source bean has this bean in its links
				sourceID := parts[1]
				source, exists := byID[sourceID]
				if !exists {
					continue // Source bean not found, can't exclude
				}

				if source.Links.HasLink(linkType, b.ID) {
					excluded = true
				}
			}

			if excluded {
				break
			}
		}
		if !excluded {
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
	listCmd.Flags().StringArrayVar(&listLinks, "links", nil, "Filter by outgoing relationship (format: type or type:id)")
	listCmd.Flags().StringArrayVar(&listLinkedAs, "linked-as", nil, "Filter by incoming relationship (format: type or type:id)")
	listCmd.Flags().StringArrayVar(&listNoLinks, "no-links", nil, "Exclude beans with outgoing relationship (format: type or type:id)")
	listCmd.Flags().StringArrayVar(&listNoLinkedAs, "no-linked-as", nil, "Exclude beans with incoming relationship (format: type or type:id)")
	listCmd.Flags().BoolVarP(&listQuiet, "quiet", "q", false, "Only output IDs (one per line)")
	listCmd.Flags().StringVar(&listSort, "sort", "status", "Sort by: created, updated, status, id (default: status)")
	listCmd.Flags().BoolVar(&listFull, "full", false, "Include bean body in JSON output")
	listCmd.MarkFlagsMutuallyExclusive("json", "quiet")
	rootCmd.AddCommand(listCmd)
}
