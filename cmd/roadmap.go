package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"hmans.dev/beans/internal/bean"
)

var (
	roadmapJSON        bool
	roadmapIncludeDone bool
	roadmapStatus      []string
	roadmapNoStatus    []string
	roadmapNoLinks     bool
	roadmapLinkPrefix  string
)

// roadmapData holds the structured roadmap for JSON output.
type roadmapData struct {
	Milestones  []milestoneGroup `json:"milestones"`
	Unscheduled []epicGroup      `json:"unscheduled,omitempty"`
}

// milestoneGroup represents a milestone and its contents.
type milestoneGroup struct {
	Milestone *bean.Bean   `json:"milestone"`
	Epics     []epicGroup  `json:"epics,omitempty"`
	Other     []*bean.Bean `json:"other,omitempty"`
}

// epicGroup represents an epic and its child items.
type epicGroup struct {
	Epic  *bean.Bean   `json:"epic"`
	Items []*bean.Bean `json:"items,omitempty"`
}

var roadmapCmd = &cobra.Command{
	Use:   "roadmap",
	Short: "Generate a Markdown roadmap from milestones and epics",
	RunE: func(cmd *cobra.Command, args []string) error {
		allBeans := core.All()

		// Build the roadmap
		data := buildRoadmap(allBeans, roadmapIncludeDone, roadmapStatus, roadmapNoStatus)

		// JSON output
		if roadmapJSON {
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(data)
		}

		// Markdown output
		links := !roadmapNoLinks
		md := renderRoadmapMarkdown(data, links, roadmapLinkPrefix)
		fmt.Print(md)
		return nil
	},
}

// buildRoadmap constructs the roadmap data structure from beans.
func buildRoadmap(allBeans []*bean.Bean, includeDone bool, statusFilter, noStatusFilter []string) *roadmapData {
	// Index all beans by ID for lookups
	byID := make(map[string]*bean.Bean)
	for _, b := range allBeans {
		byID[b.ID] = b
	}

	// Build children index: parent ID -> children
	// This maps each bean ID to the beans that have it as a parent
	children := make(map[string][]*bean.Bean)
	for _, b := range allBeans {
		for _, parentID := range b.Links.Targets("parent") {
			children[parentID] = append(children[parentID], b)
		}
	}

	// Find milestones, applying status filters
	var milestones []*bean.Bean
	for _, b := range allBeans {
		if b.Type != "milestone" {
			continue
		}
		// Apply status filters to milestones
		if len(statusFilter) > 0 && !containsStatus(statusFilter, b.Status) {
			continue
		}
		if len(noStatusFilter) > 0 && containsStatus(noStatusFilter, b.Status) {
			continue
		}
		milestones = append(milestones, b)
	}

	// Sort milestones by status order, then by created date
	sortByStatusThenCreated(milestones, cfg)

	// Build milestone groups
	var milestoneGroups []milestoneGroup
	for _, m := range milestones {
		group := buildMilestoneGroup(m, children, byID, includeDone)
		// Only include milestones that have visible content
		if len(group.Epics) > 0 || len(group.Other) > 0 {
			milestoneGroups = append(milestoneGroups, group)
		}
	}

	// Find unscheduled epics (epics with children but no milestone parent)
	var unscheduled []epicGroup
	for _, b := range allBeans {
		if b.Type != "epic" {
			continue
		}
		// Check if this epic has a milestone as parent
		if hasParentOfType(b, "milestone", byID) {
			continue
		}
		// Build epic group if it has visible children
		epicItems := filterChildren(children[b.ID], includeDone)
		if len(epicItems) > 0 {
			sortByStatusThenID(epicItems, cfg)
			unscheduled = append(unscheduled, epicGroup{Epic: b, Items: epicItems})
		}
	}

	return &roadmapData{
		Milestones:  milestoneGroups,
		Unscheduled: unscheduled,
	}
}

// buildMilestoneGroup builds a milestone group with its epics and other items.
func buildMilestoneGroup(m *bean.Bean, children map[string][]*bean.Bean, _ map[string]*bean.Bean, includeDone bool) milestoneGroup {
	group := milestoneGroup{Milestone: m}

	// Get direct children of this milestone
	directChildren := children[m.ID]

	// Separate epics from other items
	var epics []*bean.Bean

	for _, child := range directChildren {
		if child.Type == "epic" {
			epics = append(epics, child)
		}
	}

	// Track items that appear under epics to avoid duplicates in "Other"
	inEpic := make(map[string]bool)

	// Build epic groups
	for _, epic := range epics {
		epicItems := filterChildren(children[epic.ID], includeDone)
		// Only include epics that have visible children
		if len(epicItems) > 0 {
			sortByStatusThenID(epicItems, cfg)
			group.Epics = append(group.Epics, epicGroup{Epic: epic, Items: epicItems})
			// Mark these items as belonging to an epic
			for _, item := range epicItems {
				inEpic[item.ID] = true
			}
		}
	}

	// Build "Other" list: direct children that are not epics and not already in an epic
	var other []*bean.Bean
	for _, child := range directChildren {
		if child.Type == "epic" {
			continue
		}
		if inEpic[child.ID] {
			continue
		}
		if includeDone || !cfg.IsArchiveStatus(child.Status) {
			other = append(other, child)
		}
	}

	// Sort epics by their epic's title
	sort.Slice(group.Epics, func(i, j int) bool {
		return group.Epics[i].Epic.Title < group.Epics[j].Epic.Title
	})

	// Sort other items
	sortByStatusThenID(other, cfg)
	group.Other = other

	return group
}

// filterChildren filters children based on done status.
func filterChildren(children []*bean.Bean, includeDone bool) []*bean.Bean {
	if includeDone {
		// Return a copy to avoid modifying the original
		result := make([]*bean.Bean, len(children))
		copy(result, children)
		return result
	}

	var filtered []*bean.Bean
	for _, b := range children {
		if !cfg.IsArchiveStatus(b.Status) {
			filtered = append(filtered, b)
		}
	}
	return filtered
}

// hasParentOfType checks if a bean has a parent of the given type.
func hasParentOfType(b *bean.Bean, parentType string, byID map[string]*bean.Bean) bool {
	for _, parentID := range b.Links.Targets("parent") {
		if parent, ok := byID[parentID]; ok && parent.Type == parentType {
			return true
		}
	}
	return false
}

// containsStatus checks if a status is in the list.
func containsStatus(statuses []string, status string) bool {
	for _, s := range statuses {
		if s == status {
			return true
		}
	}
	return false
}

// sortByStatusThenCreated sorts beans by status order, then by created date.
func sortByStatusThenCreated(beans []*bean.Bean, cfg interface{ StatusNames() []string }) {
	statusOrder := make(map[string]int)
	for i, s := range cfg.StatusNames() {
		statusOrder[s] = i
	}

	sort.Slice(beans, func(i, j int) bool {
		oi, oj := statusOrder[beans[i].Status], statusOrder[beans[j].Status]
		if oi != oj {
			return oi < oj
		}
		// Then by created date (oldest first for milestones)
		if beans[i].CreatedAt != nil && beans[j].CreatedAt != nil {
			return beans[i].CreatedAt.Before(*beans[j].CreatedAt)
		}
		return beans[i].ID < beans[j].ID
	})
}

// sortByStatusThenID sorts beans by status order, then by ID.
func sortByStatusThenID(beans []*bean.Bean, cfg interface{ StatusNames() []string }) {
	statusOrder := make(map[string]int)
	for i, s := range cfg.StatusNames() {
		statusOrder[s] = i
	}

	sort.Slice(beans, func(i, j int) bool {
		oi, oj := statusOrder[beans[i].Status], statusOrder[beans[j].Status]
		if oi != oj {
			return oi < oj
		}
		return beans[i].ID < beans[j].ID
	})
}

// renderRoadmapMarkdown renders the roadmap as Markdown.
func renderRoadmapMarkdown(data *roadmapData, links bool, linkPrefix string) string {
	var sb strings.Builder

	sb.WriteString("# Roadmap\n")

	// Render milestones
	for _, mg := range data.Milestones {
		sb.WriteString("\n## Milestone: ")
		sb.WriteString(mg.Milestone.Title)
		sb.WriteString(" ")
		sb.WriteString(renderBeanRef(mg.Milestone, links, linkPrefix))
		sb.WriteString("\n")

		// Add milestone description (first paragraph of body)
		if desc := firstParagraph(mg.Milestone.Body); desc != "" {
			sb.WriteString("\n> ")
			sb.WriteString(desc)
			sb.WriteString("\n")
		}

		// Render epics
		for _, eg := range mg.Epics {
			sb.WriteString("\n### Epic: ")
			sb.WriteString(eg.Epic.Title)
			sb.WriteString(" ")
			sb.WriteString(renderBeanRef(eg.Epic, links, linkPrefix))
			sb.WriteString("\n")

			// Add epic description
			if desc := firstParagraph(eg.Epic.Body); desc != "" {
				sb.WriteString("\n> ")
				sb.WriteString(desc)
				sb.WriteString("\n")
			}

			sb.WriteString("\n")
			for _, item := range eg.Items {
				sb.WriteString(renderItem(item, links, linkPrefix))
			}
		}

		// Render other (ungrouped) items
		if len(mg.Other) > 0 {
			sb.WriteString("\n### Other\n\n")
			for _, item := range mg.Other {
				sb.WriteString(renderItem(item, links, linkPrefix))
			}
		}
	}

	// Render unscheduled epics
	if len(data.Unscheduled) > 0 {
		sb.WriteString("\n## Unscheduled\n")

		for _, eg := range data.Unscheduled {
			sb.WriteString("\n### Epic: ")
			sb.WriteString(eg.Epic.Title)
			sb.WriteString(" ")
			sb.WriteString(renderBeanRef(eg.Epic, links, linkPrefix))
			sb.WriteString("\n")

			if desc := firstParagraph(eg.Epic.Body); desc != "" {
				sb.WriteString("\n> ")
				sb.WriteString(desc)
				sb.WriteString("\n")
			}

			sb.WriteString("\n")
			for _, item := range eg.Items {
				sb.WriteString(renderItem(item, links, linkPrefix))
			}
		}
	}

	return sb.String()
}

// renderBeanRef renders a bean ID, optionally as a markdown link.
func renderBeanRef(b *bean.Bean, asLink bool, linkPrefix string) string {
	if !asLink {
		return "[#" + b.ID + "]"
	}
	if linkPrefix == "" {
		return fmt.Sprintf("[[#%s](%s)]", b.ID, b.Path)
	}
	// Ensure prefix ends with / for clean concatenation
	if !strings.HasSuffix(linkPrefix, "/") {
		linkPrefix += "/"
	}
	return fmt.Sprintf("[[#%s](%s%s)]", b.ID, linkPrefix, b.Path)
}

// renderItem renders a single item as a Markdown list item.
func renderItem(b *bean.Bean, asLink bool, linkPrefix string) string {
	return fmt.Sprintf("- %s %s\n", renderBeanRef(b, asLink, linkPrefix), b.Title)
}

// firstParagraph extracts the first paragraph from a body text.
func firstParagraph(body string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return ""
	}

	// Find the first blank line (paragraph separator)
	lines := strings.Split(body, "\n")
	var para []string
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			break
		}
		// Skip markdown headers
		if strings.HasPrefix(line, "#") {
			continue
		}
		para = append(para, strings.TrimSpace(line))
	}

	result := strings.Join(para, " ")
	// Truncate if too long
	if len(result) > 200 {
		result = result[:197] + "..."
	}
	return result
}

func init() {
	roadmapCmd.Flags().BoolVar(&roadmapJSON, "json", false, "Output as JSON")
	roadmapCmd.Flags().BoolVar(&roadmapIncludeDone, "include-done", false, "Include completed items")
	roadmapCmd.Flags().StringArrayVar(&roadmapStatus, "status", nil, "Filter milestones by status (can be repeated)")
	roadmapCmd.Flags().StringArrayVar(&roadmapNoStatus, "no-status", nil, "Exclude milestones by status (can be repeated)")
	roadmapCmd.Flags().BoolVar(&roadmapNoLinks, "no-links", false, "Don't render bean IDs as markdown links")
	roadmapCmd.Flags().StringVar(&roadmapLinkPrefix, "link-prefix", "", "URL prefix for links")
	rootCmd.AddCommand(roadmapCmd)
}
