package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"hmans.dev/beans/internal/bean"
	"hmans.dev/beans/internal/config"
	"hmans.dev/beans/internal/output"
	"hmans.dev/beans/internal/ui"
)

var (
	listJSON       bool
	listStatus     []string
	listNoStatus   []string
	listLinks      []string
	listLinkedAs   []string
	listNoLinks    []string
	listNoLinkedAs []string
	listQuiet      bool
	listSort       string
	listFull       bool
)

// linkFilter represents a pre-parsed link filter criterion.
type linkFilter struct {
	linkType string
	targetID string // empty means "any target"
}

// parseLinkFilters parses filter strings like "blocks" or "blocks:id" into linkFilter structs.
func parseLinkFilters(filters []string) []linkFilter {
	result := make([]linkFilter, len(filters))
	for i, f := range filters {
		parts := strings.SplitN(f, ":", 2)
		result[i].linkType = parts[0]
		if len(parts) == 2 {
			result[i].targetID = parts[1]
		}
	}
	return result
}

// linkIndex holds precomputed data structures for efficient link filtering.
type linkIndex struct {
	byID       map[string]*bean.Bean      // ID -> Bean lookup
	targetedBy map[string]map[string]bool // linkType -> set of target IDs
}

// buildLinkIndex creates a linkIndex from a slice of beans.
// This should be called once before any filtering to capture all relationships.
func buildLinkIndex(beans []*bean.Bean) *linkIndex {
	idx := &linkIndex{
		byID:       make(map[string]*bean.Bean),
		targetedBy: make(map[string]map[string]bool),
	}
	for _, b := range beans {
		idx.byID[b.ID] = b
		for _, link := range b.Links {
			if idx.targetedBy[link.Type] == nil {
				idx.targetedBy[link.Type] = make(map[string]bool)
			}
			idx.targetedBy[link.Type][link.Target] = true
		}
	}
	return idx
}

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

		// Parse filter criteria once (avoid repeated string splitting)
		linksFilters := parseLinkFilters(listLinks)
		linkedAsFilters := parseLinkFilters(listLinkedAs)
		noLinksFilters := parseLinkFilters(listNoLinks)
		noLinkedAsFilters := parseLinkFilters(listNoLinkedAs)

		// Build link index once from all beans (before status filtering)
		// This ensures relationships are captured even if source bean is filtered out
		idx := buildLinkIndex(beans)

		// Apply filters (positive first, then exclusions)
		beans = filterBeans(beans, listStatus)
		beans = excludeByStatus(beans, listNoStatus)
		beans = filterByLinks(beans, linksFilters)
		beans = filterByLinkedAs(beans, linkedAsFilters, idx)
		beans = excludeByLinks(beans, noLinksFilters)
		beans = excludeByLinkedAs(beans, noLinkedAsFilters, idx)

		// Sort beans
		sortBeans(beans, listSort, cfg)

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

func sortBeans(beans []*bean.Bean, sortBy string, cfg *config.Config) {
	statusNames := cfg.StatusNames()
	typeNames := cfg.TypeNames()

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
	case "id":
		sort.Slice(beans, func(i, j int) bool {
			return beans[i].ID < beans[j].ID
		})
	default:
		// Default: sort by archive status (not done first), then by type order
		typeOrder := make(map[string]int)
		for i, t := range typeNames {
			typeOrder[t] = i
		}

		sort.Slice(beans, func(i, j int) bool {
			// First: sort by archive status (non-archive/not-done first)
			iArchive := cfg.IsArchiveStatus(beans[i].Status)
			jArchive := cfg.IsArchiveStatus(beans[j].Status)
			if iArchive != jArchive {
				return !iArchive // non-archive (not done) comes first
			}

			// Second: sort by type order from config
			ti, tj := typeOrder[beans[i].Type], typeOrder[beans[j].Type]
			if ti != tj {
				return ti < tj
			}

			// Finally: sort by ID for stable ordering
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

// excludeByStatus excludes beans that match any of the given statuses.
// Inverse of filterBeans: returns beans that DON'T match the criteria.
//
// Examples:
//   - --no-status done returns beans that are not done
//   - --no-status done --no-status archived returns beans that are neither done nor archived
func excludeByStatus(beans []*bean.Bean, statuses []string) []*bean.Bean {
	if len(statuses) == 0 {
		return beans
	}

	var filtered []*bean.Bean
	for _, b := range beans {
		excluded := false
		for _, s := range statuses {
			if b.Status == s {
				excluded = true
				break
			}
		}
		if !excluded {
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
func filterByLinks(beans []*bean.Bean, filters []linkFilter) []*bean.Bean {
	if len(filters) == 0 {
		return beans
	}

	var filtered []*bean.Bean
	for _, b := range beans {
		matched := false
		for _, f := range filters {
			if f.targetID == "" {
				// Type-only: check if this bean has ANY link of this type
				if b.Links.HasType(f.linkType) {
					matched = true
				}
			} else {
				// Type:ID: check if this bean links to the specific target
				if b.Links.HasLink(f.linkType, f.targetID) {
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
func filterByLinkedAs(beans []*bean.Bean, filters []linkFilter, idx *linkIndex) []*bean.Bean {
	if len(filters) == 0 {
		return beans
	}

	var filtered []*bean.Bean
	for _, b := range beans {
		matched := false
		for _, f := range filters {
			if f.targetID == "" {
				// Type-only: check if this bean is targeted by ANY bean with this link type
				if targets, ok := idx.targetedBy[f.linkType]; ok && targets[b.ID] {
					matched = true
				}
			} else {
				// Type:ID: check if specific source bean has this bean in its links
				source, exists := idx.byID[f.targetID]
				if !exists {
					continue // Source bean not found
				}

				if source.Links.HasLink(f.linkType, b.ID) {
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
func excludeByLinks(beans []*bean.Bean, filters []linkFilter) []*bean.Bean {
	if len(filters) == 0 {
		return beans
	}

	var filtered []*bean.Bean
	for _, b := range beans {
		excluded := false
		for _, f := range filters {
			if f.targetID == "" {
				// Type-only: exclude if this bean has ANY link of this type
				if b.Links.HasType(f.linkType) {
					excluded = true
				}
			} else {
				// Type:ID: exclude if this bean links to the specific target
				if b.Links.HasLink(f.linkType, f.targetID) {
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
func excludeByLinkedAs(beans []*bean.Bean, filters []linkFilter, idx *linkIndex) []*bean.Bean {
	if len(filters) == 0 {
		return beans
	}

	var filtered []*bean.Bean
	for _, b := range beans {
		excluded := false
		for _, f := range filters {
			if f.targetID == "" {
				// Type-only: exclude if this bean is targeted by ANY bean with this link type
				if targets, ok := idx.targetedBy[f.linkType]; ok && targets[b.ID] {
					excluded = true
				}
			} else {
				// Type:ID: exclude if specific source bean has this bean in its links
				source, exists := idx.byID[f.targetID]
				if !exists {
					continue // Source bean not found, can't exclude
				}

				if source.Links.HasLink(f.linkType, b.ID) {
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
	listCmd.Flags().StringArrayVar(&listNoStatus, "no-status", nil, "Exclude by status (can be repeated)")
	listCmd.Flags().StringArrayVar(&listLinks, "links", nil, "Filter by outgoing relationship (format: type or type:id)")
	listCmd.Flags().StringArrayVar(&listLinkedAs, "linked-as", nil, "Filter by incoming relationship (format: type or type:id)")
	listCmd.Flags().StringArrayVar(&listNoLinks, "no-links", nil, "Exclude beans with outgoing relationship (format: type or type:id)")
	listCmd.Flags().StringArrayVar(&listNoLinkedAs, "no-linked-as", nil, "Exclude beans with incoming relationship (format: type or type:id)")
	listCmd.Flags().BoolVarP(&listQuiet, "quiet", "q", false, "Only output IDs (one per line)")
	listCmd.Flags().StringVar(&listSort, "sort", "", "Sort by: created, updated, status, id (default: not-done/done, then type)")
	listCmd.Flags().BoolVar(&listFull, "full", false, "Include bean body in JSON output")
	listCmd.MarkFlagsMutuallyExclusive("json", "quiet")
	rootCmd.AddCommand(listCmd)
}
