package cmd

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/config"
	"github.com/hmans/beans/internal/graph"
	"github.com/hmans/beans/internal/graph/model"
	"github.com/hmans/beans/internal/output"
	"github.com/hmans/beans/internal/ui"
)

var (
	listJSON       bool
	listSearch     string
	listStatus     []string
	listNoStatus   []string
	listType       []string
	listNoType     []string
	listPriority   []string
	listNoPriority []string
	listLinks      []string
	listLinkedAs   []string
	listNoLinks    []string
	listNoLinkedAs []string
	listTag        []string
	listNoTag      []string
	listQuiet      bool
	listSort       string
	listFull       bool
)

// stringToLinkType converts a CLI string to a LinkType enum value.
func stringToLinkType(s string) model.LinkType {
	switch strings.ToUpper(s) {
	case "MILESTONE":
		return model.LinkTypeMilestone
	case "EPIC":
		return model.LinkTypeEpic
	case "FEATURE":
		return model.LinkTypeFeature
	case "BLOCKS":
		return model.LinkTypeBlocks
	default:
		// Return the input as-is converted to LinkType (will fail validation if invalid)
		return model.LinkType(strings.ToUpper(s))
	}
}

// parseLinkFilters parses CLI link filter strings (e.g., "blocks" or "blocks:id")
// into GraphQL LinkFilter models.
func parseLinkFilters(filters []string) []*model.LinkFilter {
	if len(filters) == 0 {
		return nil
	}
	result := make([]*model.LinkFilter, len(filters))
	for i, f := range filters {
		parts := strings.SplitN(f, ":", 2)
		lf := &model.LinkFilter{Type: stringToLinkType(parts[0])}
		if len(parts) == 2 {
			target := parts[1]
			lf.Target = &target
		}
		result[i] = lf
	}
	return result
}

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all beans",
	Long: `Lists all beans in the .beans directory.

Search Syntax (--search/-S):
  The search flag supports Bleve query string syntax:

  login          Exact term match
  login~         Fuzzy match (1 edit distance, finds "loggin", "logins")
  login~2        Fuzzy match (2 edit distance)
  log*           Wildcard prefix match
  "user login"   Exact phrase match
  user AND login Both terms required
  user OR login  Either term matches
  slug:auth      Search only in slug field
  title:login    Search only in title field
  body:auth      Search only in body field`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Build GraphQL filter from CLI flags
		filter := &model.BeanFilter{
			Status:          listStatus,
			ExcludeStatus:   listNoStatus,
			Type:            listType,
			ExcludeType:     listNoType,
			Priority:        listPriority,
			ExcludePriority: listNoPriority,
			Tags:            listTag,
			ExcludeTags:     listNoTag,
			HasLinks:        parseLinkFilters(listLinks),
			LinkedAs:        parseLinkFilters(listLinkedAs),
			NoLinks:         parseLinkFilters(listNoLinks),
			NoLinkedAs:      parseLinkFilters(listNoLinkedAs),
		}

		// Add search filter if provided
		if listSearch != "" {
			filter.Search = &listSearch
		}

		// Execute query via GraphQL resolver
		resolver := &graph.Resolver{Core: core}
		beans, err := resolver.Query().Beans(context.Background(), filter)
		if err != nil {
			return fmt.Errorf("querying beans: %w", err)
		}

		// Sort beans
		sortBeans(beans, listSort, cfg)

		// JSON output (flat list)
		if listJSON {
			if !listFull {
				for _, b := range beans {
					b.Body = ""
				}
			}
			return output.SuccessMultiple(beans)
		}

		// Quiet mode: just IDs (flat)
		if listQuiet {
			for _, b := range beans {
				fmt.Println(b.ID)
			}
			return nil
		}

		// Default: tree view
		// We need all beans to find ancestors for context
		allBeans, err := resolver.Query().Beans(context.Background(), nil)
		if err != nil {
			return fmt.Errorf("querying all beans for tree: %w", err)
		}

		// Create sort function for tree building
		sortFn := func(b []*bean.Bean) {
			sortBeans(b, listSort, cfg)
		}

		// Build tree
		tree := ui.BuildTree(beans, allBeans, sortFn)

		if len(tree) == 0 {
			fmt.Println(ui.Muted.Render("No beans found. Create one with: beans new <title>"))
			return nil
		}

		// Calculate max ID width from all beans in tree
		maxIDWidth := 2
		for _, b := range allBeans {
			if len(b.ID) > maxIDWidth {
				maxIDWidth = len(b.ID)
			}
		}
		maxIDWidth += 2

		// Check if any beans have tags
		hasTags := false
		for _, b := range beans {
			if len(b.Tags) > 0 {
				hasTags = true
				break
			}
		}

		fmt.Print(ui.RenderTree(tree, cfg, maxIDWidth, hasTags))
		return nil
	},
}

func sortBeans(beans []*bean.Bean, sortBy string, cfg *config.Config) {
	statusNames := cfg.StatusNames()
	priorityNames := cfg.PriorityNames()
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
	case "priority":
		// Build priority order from configured priorities
		priorityOrder := make(map[string]int)
		for i, p := range priorityNames {
			priorityOrder[p] = i
		}
		// Find normal priority index for beans without priority
		normalIdx := len(priorityNames)
		for i, p := range priorityNames {
			if p == "normal" {
				normalIdx = i
				break
			}
		}
		sort.Slice(beans, func(i, j int) bool {
			pi := normalIdx
			if beans[i].Priority != "" {
				if order, ok := priorityOrder[beans[i].Priority]; ok {
					pi = order
				}
			}
			pj := normalIdx
			if beans[j].Priority != "" {
				if order, ok := priorityOrder[beans[j].Priority]; ok {
					pj = order
				}
			}
			if pi != pj {
				return pi < pj
			}
			return beans[i].ID < beans[j].ID
		})
	case "id":
		sort.Slice(beans, func(i, j int) bool {
			return beans[i].ID < beans[j].ID
		})
	default:
		// Default: sort by status order, then priority, then type order, then title (same as TUI)
		bean.SortByStatusPriorityAndType(beans, statusNames, priorityNames, typeNames)
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func init() {
	listCmd.Flags().BoolVar(&listJSON, "json", false, "Output as JSON")
	listCmd.Flags().StringVarP(&listSearch, "search", "S", "", "Full-text search in title and body")
	listCmd.Flags().StringArrayVarP(&listStatus, "status", "s", nil, "Filter by status (can be repeated)")
	listCmd.Flags().StringArrayVar(&listNoStatus, "no-status", nil, "Exclude by status (can be repeated)")
	listCmd.Flags().StringArrayVarP(&listType, "type", "t", nil, "Filter by type (can be repeated)")
	listCmd.Flags().StringArrayVar(&listNoType, "no-type", nil, "Exclude by type (can be repeated)")
	listCmd.Flags().StringArrayVarP(&listPriority, "priority", "p", nil, "Filter by priority (can be repeated)")
	listCmd.Flags().StringArrayVar(&listNoPriority, "no-priority", nil, "Exclude by priority (can be repeated)")
	listCmd.Flags().StringArrayVar(&listLinks, "links", nil, "Filter by outgoing relationship (format: type or type:id)")
	listCmd.Flags().StringArrayVar(&listLinkedAs, "linked-as", nil, "Filter by incoming relationship (format: type or type:id)")
	listCmd.Flags().StringArrayVar(&listNoLinks, "no-links", nil, "Exclude beans with outgoing relationship (format: type or type:id)")
	listCmd.Flags().StringArrayVar(&listNoLinkedAs, "no-linked-as", nil, "Exclude beans with incoming relationship (format: type or type:id)")
	listCmd.Flags().StringArrayVar(&listTag, "tag", nil, "Filter by tag (can be repeated, OR logic)")
	listCmd.Flags().StringArrayVar(&listNoTag, "no-tag", nil, "Exclude beans with tag (can be repeated)")
	listCmd.Flags().BoolVarP(&listQuiet, "quiet", "q", false, "Only output IDs (one per line)")
	listCmd.Flags().StringVar(&listSort, "sort", "", "Sort by: created, updated, status, priority, id (default: status, priority, type, title)")
	listCmd.Flags().BoolVar(&listFull, "full", false, "Include bean body in JSON output")
	rootCmd.AddCommand(listCmd)
}
