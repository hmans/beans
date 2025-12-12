package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/config"
)

// TreeNode represents a node in the bean tree hierarchy.
type TreeNode struct {
	Bean     *bean.Bean
	Children []*TreeNode
	Matched  bool // true if this bean matched the filter (vs. shown for context)
}

// TreeNodeJSON is the JSON-serializable version of TreeNode.
type TreeNodeJSON struct {
	ID        string          `json:"id"`
	Slug      string          `json:"slug,omitempty"`
	Path      string          `json:"path"`
	Title     string          `json:"title"`
	Status    string          `json:"status"`
	Type      string          `json:"type,omitempty"`
	Priority  string          `json:"priority,omitempty"`
	Tags      []string        `json:"tags,omitempty"`
	Body      string          `json:"body,omitempty"`
	Matched   bool            `json:"matched"`
	Children  []*TreeNodeJSON `json:"children,omitempty"`
}

// ToJSON converts a TreeNode to its JSON-serializable form.
func (n *TreeNode) ToJSON(includeFull bool) *TreeNodeJSON {
	json := &TreeNodeJSON{
		ID:       n.Bean.ID,
		Slug:     n.Bean.Slug,
		Path:     n.Bean.Path,
		Title:    n.Bean.Title,
		Status:   n.Bean.Status,
		Type:     n.Bean.Type,
		Priority: n.Bean.Priority,
		Tags:     n.Bean.Tags,
		Matched:  n.Matched,
	}
	if includeFull {
		json.Body = n.Bean.Body
	}
	if len(n.Children) > 0 {
		json.Children = make([]*TreeNodeJSON, len(n.Children))
		for i, child := range n.Children {
			json.Children[i] = child.ToJSON(includeFull)
		}
	}
	return json
}

// BuildTree builds a tree structure from filtered beans, including ancestors for context.
// matchedBeans: beans that matched the filter
// allBeans: all beans (needed to find ancestors)
// sortFn: function to sort beans at each level
func BuildTree(matchedBeans []*bean.Bean, allBeans []*bean.Bean, sortFn func([]*bean.Bean)) []*TreeNode {
	// Build index of all beans by ID
	beanByID := make(map[string]*bean.Bean)
	for _, b := range allBeans {
		beanByID[b.ID] = b
	}

	// Build set of matched bean IDs
	matchedSet := make(map[string]bool)
	for _, b := range matchedBeans {
		matchedSet[b.ID] = true
	}

	// Find all ancestors needed for context
	// Start with matched beans, then walk up parent links
	neededBeans := make(map[string]*bean.Bean)
	for _, b := range matchedBeans {
		neededBeans[b.ID] = b
	}

	// Add ancestors of matched beans
	for _, b := range matchedBeans {
		addAncestors(b, beanByID, neededBeans)
	}

	// Build children index (parent ID -> children)
	children := make(map[string][]*bean.Bean)
	for _, b := range neededBeans {
		if b.Parent != "" {
			// Only add as child if parent is in our needed set
			if _, ok := neededBeans[b.Parent]; ok {
				children[b.Parent] = append(children[b.Parent], b)
			}
		}
	}

	// Sort children at each level
	for parentID := range children {
		sortFn(children[parentID])
	}

	// Find root beans (no parent or parent not in needed set)
	var roots []*bean.Bean
	for _, b := range neededBeans {
		if b.Parent == "" {
			roots = append(roots, b)
		} else {
			// Check if parent is in the tree
			if _, ok := neededBeans[b.Parent]; !ok {
				roots = append(roots, b)
			}
		}
	}
	sortFn(roots)

	// Build tree nodes recursively
	return buildNodes(roots, children, matchedSet)
}

// addAncestors recursively adds all ancestors of a bean to the needed set.
func addAncestors(b *bean.Bean, beanByID map[string]*bean.Bean, needed map[string]*bean.Bean) {
	if b.Parent == "" {
		return
	}
	parent, ok := beanByID[b.Parent]
	if !ok {
		return // parent doesn't exist (broken link)
	}
	if _, alreadyNeeded := needed[b.Parent]; alreadyNeeded {
		return // already processed
	}
	needed[b.Parent] = parent
	addAncestors(parent, beanByID, needed)
}

// buildNodes recursively builds TreeNodes from beans.
func buildNodes(beans []*bean.Bean, children map[string][]*bean.Bean, matchedSet map[string]bool) []*TreeNode {
	nodes := make([]*TreeNode, len(beans))
	for i, b := range beans {
		nodes[i] = &TreeNode{
			Bean:     b,
			Matched:  matchedSet[b.ID],
			Children: buildNodes(children[b.ID], children, matchedSet),
		}
	}
	return nodes
}

// Tree rendering constants
const (
	treeBranch     = "├─"
	treeLastBranch = "└─"
	treePipe       = "" // no continuation lines
	treeSpace      = "" // no spacing for completed branches
	treeIndent     = 2  // width of connector (├─ or └─)
)

// calculateMaxDepth returns the maximum depth of the tree.
func calculateMaxDepth(nodes []*TreeNode) int {
	maxDepth := 0
	for _, node := range nodes {
		depth := 1 + calculateMaxDepth(node.Children)
		if depth > maxDepth {
			maxDepth = depth
		}
	}
	return maxDepth
}

// RenderTree renders the tree as an ASCII tree with styled columns.
func RenderTree(nodes []*TreeNode, cfg *config.Config, maxIDWidth int, hasTags bool) string {
	var sb strings.Builder

	// Calculate max depth to determine ID column width
	maxDepth := calculateMaxDepth(nodes)
	// ID column needs: indent (2 chars per level beyond depth 1) + connector (2 chars) + ID width
	// depth 0: 0 extra chars
	// depth 1: 2 chars (connector only)
	// depth 2: 4 chars (2 indent + 2 connector)
	// depth N: (N-1)*2 + 2 = N*2 chars
	treeColWidth := maxIDWidth
	if maxDepth > 0 {
		treeColWidth = maxIDWidth + maxDepth*treeIndent
	}

	// Column styles with widths for alignment
	idStyle := lipgloss.NewStyle().Width(treeColWidth)
	typeStyle := lipgloss.NewStyle().Width(12)
	statusStyle := lipgloss.NewStyle().Width(14)
	tagsStyle := lipgloss.NewStyle().Width(24)
	headerCol := lipgloss.NewStyle().Foreground(ColorMuted)

	// Header
	var header string
	var dividerWidth int
	if hasTags {
		header = lipgloss.JoinHorizontal(lipgloss.Top,
			idStyle.Render(headerCol.Render("ID")),
			typeStyle.Render(headerCol.Render("TYPE")),
			statusStyle.Render(headerCol.Render("STATUS")),
			tagsStyle.Render(headerCol.Render("TAGS")),
			headerCol.Render("TITLE"),
		)
		dividerWidth = treeColWidth + 12 + 14 + 24 + 30
	} else {
		header = lipgloss.JoinHorizontal(lipgloss.Top,
			idStyle.Render(headerCol.Render("ID")),
			typeStyle.Render(headerCol.Render("TYPE")),
			statusStyle.Render(headerCol.Render("STATUS")),
			headerCol.Render("TITLE"),
		)
		dividerWidth = treeColWidth + 12 + 14 + 30
	}
	sb.WriteString(header)
	sb.WriteString("\n")
	sb.WriteString(Muted.Render(strings.Repeat("─", dividerWidth)))
	sb.WriteString("\n")

	// Render nodes (depth 0 = root level)
	renderNodes(&sb, nodes, 0, cfg, treeColWidth, hasTags)

	return sb.String()
}

// renderNodes recursively renders tree nodes with proper indentation.
// depth 0 = root level (no connector), depth 1+ = nested (has connector)
func renderNodes(sb *strings.Builder, nodes []*TreeNode, depth int, cfg *config.Config, treeColWidth int, hasTags bool) {
	for i, node := range nodes {
		isLast := i == len(nodes)-1
		renderNode(sb, node, depth, isLast, cfg, treeColWidth, hasTags)
		renderNodes(sb, node.Children, depth+1, cfg, treeColWidth, hasTags)
	}
}

// renderNode renders a single tree node with tree connectors.
// treeColWidth is the fixed width of the ID column (includes space for tree connectors).
// depth 0 = root (no connector), depth 1+ = nested (has connector)
func renderNode(sb *strings.Builder, node *TreeNode, depth int, isLast bool, cfg *config.Config, treeColWidth int, hasTags bool) {
	b := node.Bean

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

	// Get priority color and render symbol
	priorityColor := ""
	if priorityCfg := cfg.GetPriority(b.Priority); priorityCfg != nil {
		priorityColor = priorityCfg.Color
	}
	prioritySymbol := RenderPrioritySymbol(b.Priority, priorityColor)
	if prioritySymbol != "" {
		prioritySymbol += " "
	}

	// Column styles - fixed widths for alignment
	typeStyle := lipgloss.NewStyle().Width(12)
	statusStyle := lipgloss.NewStyle().Width(14)
	tagsStyle := lipgloss.NewStyle().Width(24)

	// Build indentation and connector
	// depth 0: no indent, no connector
	// depth 1: connector only (├─ or └─)
	// depth 2+: indent + connector
	var indent string
	var connector string
	if depth > 0 {
		// Add indentation for depth > 1 (2 spaces per level beyond first)
		if depth > 1 {
			indent = strings.Repeat("  ", depth-1)
		}
		if isLast {
			connector = treeLastBranch
		} else {
			connector = treeBranch
		}
	}

	// Build the ID cell content: indent + connector + ID
	var idText string
	if node.Matched {
		idText = ID.Render(b.ID)
	} else {
		// Dim unmatched (ancestor) beans
		idText = Muted.Render(b.ID)
	}

	// Calculate visual width of indent + connector + ID (without ANSI codes)
	visualWidth := len(indent) + runeWidth(connector) + len(b.ID)
	// Pad to fixed width
	padding := ""
	if treeColWidth > visualWidth {
		padding = strings.Repeat(" ", treeColWidth-visualWidth)
	}
	idCell := indent + connector + idText + padding

	typeText := ""
	if b.Type != "" {
		if node.Matched {
			typeText = RenderTypeText(b.Type, typeColor)
		} else {
			typeText = Muted.Render(b.Type)
		}
	}

	var statusText string
	if node.Matched {
		statusText = RenderStatusTextWithColor(b.Status, statusColor, isArchive)
	} else {
		statusText = Muted.Render(b.Status)
	}

	var titleText string
	title := truncateString(b.Title, 50)
	if node.Matched {
		titleText = prioritySymbol + title
	} else {
		titleText = Muted.Render(title)
	}

	var row string
	if hasTags {
		var tagsStr string
		if node.Matched {
			tagsStr = RenderTagsCompact(b.Tags, 1)
		} else {
			// For unmatched, just show muted tags
			if len(b.Tags) > 0 {
				tagsStr = Muted.Render(b.Tags[0])
				if len(b.Tags) > 1 {
					tagsStr += Muted.Render(" +" + string(rune('0'+len(b.Tags)-1)))
				}
			}
		}

		row = lipgloss.JoinHorizontal(lipgloss.Top,
			idCell,
			typeStyle.Render(typeText),
			statusStyle.Render(statusText),
			tagsStyle.Render(tagsStr),
			titleText,
		)
	} else {
		row = lipgloss.JoinHorizontal(lipgloss.Top,
			idCell,
			typeStyle.Render(typeText),
			statusStyle.Render(statusText),
			titleText,
		)
	}

	sb.WriteString(row)
	sb.WriteString("\n")
}

// truncateString truncates a string to maxLen, adding "..." if truncated.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// runeWidth returns the visual width of a string (counting runes, not bytes).
// This assumes all runes are single-width (which works for our tree connectors).
func runeWidth(s string) int {
	return len([]rune(s))
}

// FlatItem represents a flattened tree node with rendering context.
// Used by TUI to render tree structure in a flat list.
type FlatItem struct {
	Bean       *bean.Bean
	Depth      int    // 0 = root, 1+ = nested
	IsLast     bool   // last child at this level
	Matched    bool   // true if bean matched filter (vs. shown for context)
	TreePrefix string // pre-computed tree prefix (e.g., "  └─")
}

// FlattenTree converts a tree into a flat slice with tree context preserved.
// Each item includes the pre-computed tree prefix for rendering.
func FlattenTree(nodes []*TreeNode) []FlatItem {
	var items []FlatItem
	flattenNodes(nodes, 0, &items)
	return items
}

func flattenNodes(nodes []*TreeNode, depth int, items *[]FlatItem) {
	for i, node := range nodes {
		isLast := i == len(nodes)-1

		// Compute tree prefix
		var prefix string
		if depth > 0 {
			// Add indentation for depth > 1
			if depth > 1 {
				prefix = strings.Repeat("  ", depth-1)
			}
			// Add connector
			if isLast {
				prefix += treeLastBranch
			} else {
				prefix += treeBranch
			}
		}

		*items = append(*items, FlatItem{
			Bean:       node.Bean,
			Depth:      depth,
			IsLast:     isLast,
			Matched:    node.Matched,
			TreePrefix: prefix,
		})

		// Recurse into children
		flattenNodes(node.Children, depth+1, items)
	}
}

// MaxTreeDepth returns the maximum depth of the flattened tree.
func MaxTreeDepth(items []FlatItem) int {
	maxDepth := 0
	for _, item := range items {
		if item.Depth > maxDepth {
			maxDepth = item.Depth
		}
	}
	return maxDepth
}
