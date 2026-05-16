package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/hmans/beans/pkg/bean"
	"github.com/hmans/beans/pkg/config"
)

// TreeNode represents a node in the bean tree hierarchy.
type TreeNode struct {
	Bean            *bean.Bean
	Children        []*TreeNode
	Matched         bool   // true if this bean matched the filter (vs. shown for context)
	ImplicitStatus string // implicit terminal status from an ancestor, if any
	Blocked         bool   // true if this bean has at least one active (non-archive) blocker
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
// implicitStatuses: optional map of beanID -> implicit terminal status (may be nil)
// activeBlockers: optional map of beanID -> list of active blocker IDs (may be nil).
//   When non-nil, siblings are topologically reordered within their existing sort
//   bucket so that an active sibling blocker appears before the bean it blocks.
//   A bean with at least one entry in this map is marked Blocked=true.
func BuildTree(matchedBeans []*bean.Bean, allBeans []*bean.Bean, sortFn func([]*bean.Bean), implicitStatuses map[string]string, activeBlockers map[string][]string) []*TreeNode {
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

	// Sort children at each level, then apply sibling-blocker topo reorder.
	for parentID := range children {
		sortFn(children[parentID])
		children[parentID] = reorderByActiveBlockers(children[parentID], activeBlockers)
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
	roots = reorderByActiveBlockers(roots, activeBlockers)

	// Build tree nodes recursively
	return buildNodes(roots, children, matchedSet, implicitStatuses, activeBlockers)
}

// reorderByActiveBlockers returns a permutation of `siblings` where, for any
// pair (A, B) of siblings such that A is an active blocker of B, A appears
// before B. Pairs unrelated by an active-blocker edge keep their incoming order
// (stable). Edges to non-siblings are ignored. Cycles can't occur — the core
// already rejects them at write time.
func reorderByActiveBlockers(siblings []*bean.Bean, activeBlockers map[string][]string) []*bean.Bean {
	if len(activeBlockers) == 0 || len(siblings) < 2 {
		return siblings
	}
	// Build a set of sibling IDs for fast lookup.
	siblingSet := make(map[string]bool, len(siblings))
	for _, s := range siblings {
		siblingSet[s.ID] = true
	}
	// In-degree counts only sibling edges. Adjacency: blocker -> [blockee...].
	indeg := make(map[string]int, len(siblings))
	adj := make(map[string][]string, len(siblings))
	for _, s := range siblings {
		for _, blockerID := range activeBlockers[s.ID] {
			if siblingSet[blockerID] {
				indeg[s.ID]++
				adj[blockerID] = append(adj[blockerID], s.ID)
			}
		}
	}
	if len(adj) == 0 {
		return siblings // no sibling-level blocker edges
	}
	// Kahn's algorithm, processing siblings in their incoming order to keep
	// the sort stable for unrelated pairs.
	out := make([]*bean.Bean, 0, len(siblings))
	enqueued := make(map[string]bool, len(siblings))
	byID := make(map[string]*bean.Bean, len(siblings))
	for _, s := range siblings {
		byID[s.ID] = s
	}
	var queue []string
	for _, s := range siblings {
		if indeg[s.ID] == 0 {
			queue = append(queue, s.ID)
			enqueued[s.ID] = true
		}
	}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		out = append(out, byID[id])
		for _, next := range adj[id] {
			indeg[next]--
			if indeg[next] == 0 && !enqueued[next] {
				queue = append(queue, next)
				enqueued[next] = true
			}
		}
	}
	// Defensive: if a cycle ever slipped through, append remaining in original
	// order so we never drop beans from the tree.
	if len(out) != len(siblings) {
		for _, s := range siblings {
			if !enqueued[s.ID] {
				out = append(out, s)
			}
		}
	}
	return out
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
func buildNodes(beans []*bean.Bean, children map[string][]*bean.Bean, matchedSet map[string]bool, implicitStatuses map[string]string, activeBlockers map[string][]string) []*TreeNode {
	nodes := make([]*TreeNode, len(beans))
	for i, b := range beans {
		nodes[i] = &TreeNode{
			Bean:            b,
			Matched:         matchedSet[b.ID],
			Children:        buildNodes(children[b.ID], children, matchedSet, implicitStatuses, activeBlockers),
			ImplicitStatus: implicitStatuses[b.ID],
			Blocked:         len(activeBlockers[b.ID]) > 0,
		}
	}
	return nodes
}

// Tree rendering constants
const (
	treeBranch     = "├─ "
	treeLastBranch = "└─ "
	treePipe       = "│  " // vertical line for ongoing branches
	treeSpace      = "   " // empty space for completed branches
	treeIndent     = 3     // width of connector
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
// termWidth is used to calculate responsive column widths.
func RenderTree(nodes []*TreeNode, cfg *config.Config, maxIDWidth int, hasTags bool, termWidth int) string {
	var sb strings.Builder

	// Calculate max depth to determine ID column width
	maxDepth := calculateMaxDepth(nodes)
	// ID column needs: indent (3 chars per level beyond depth 1) + connector (3 chars) + ID width
	// depth 0: 0 extra chars
	// depth 1: 3 chars (connector only)
	// depth 2: 6 chars (3 indent + 3 connector)
	// depth N: (N-1)*3 + 3 = N*3 chars
	treeColWidth := maxIDWidth
	if maxDepth > 0 {
		treeColWidth = maxIDWidth + maxDepth*treeIndent
	}

	// Calculate responsive columns based on terminal width
	// Adjust for tree column width vs default ID column width
	adjustedWidth := termWidth - treeColWidth + ColWidthID
	cols := CalculateResponsiveColumns(adjustedWidth, hasTags)

	// Calculate title width from remaining space
	// Account for: tree/ID col, type col, status col, priority symbol (2), space before tags (1)
	titleWidth := termWidth - treeColWidth - ColWidthType - ColWidthStatus - 3
	if cols.ShowTags {
		titleWidth -= cols.Tags
	}
	if titleWidth < 20 {
		titleWidth = 20
	}

	// Header with manual padding (lipgloss Width doesn't handle styled strings well)
	headerCol := lipgloss.NewStyle().Foreground(ColorMuted)
	idHeader := headerCol.Render("ID") + strings.Repeat(" ", treeColWidth-2)
	typeHeader := headerCol.Render("T") + strings.Repeat(" ", ColWidthType-1)
	statusHeader := headerCol.Render("S") + strings.Repeat(" ", ColWidthStatus-1)

	header := idHeader + typeHeader + statusHeader + headerCol.Render("TITLE")
	if cols.ShowTags && titleWidth > 5 {
		header += strings.Repeat(" ", titleWidth-5+3) + headerCol.Render("TAGS") // +3 for priority/spacing
	}
	dividerWidth := termWidth - 1 // -1 to avoid wrapping on exact terminal width
	sb.WriteString(header)
	sb.WriteString("\n")
	sb.WriteString(Muted.Render(strings.Repeat("─", dividerWidth)))
	sb.WriteString("\n")

	// Build render config from responsive columns
	renderCfg := treeRenderConfig{
		treeColWidth: treeColWidth,
		titleWidth:   titleWidth,
		cols:         cols,
	}

	// Render nodes (depth 0 = root level, no ancestry yet)
	renderNodes(&sb, nodes, 0, nil, cfg, renderCfg)

	return sb.String()
}

// treeRenderConfig holds computed rendering configuration for tree output
type treeRenderConfig struct {
	treeColWidth int
	titleWidth   int
	cols         ResponsiveColumns
}

// renderNodes recursively renders tree nodes with proper indentation.
// depth 0 = root level (no connector), depth 1+ = nested (has connector)
// ancestry tracks whether each parent level was a last child (true = last, no continuation line needed)
func renderNodes(sb *strings.Builder, nodes []*TreeNode, depth int, ancestry []bool, cfg *config.Config, renderCfg treeRenderConfig) {
	for i, node := range nodes {
		isLast := i == len(nodes)-1
		renderNode(sb, node, depth, isLast, ancestry, cfg, renderCfg)
		// Only add to ancestry when depth > 0 (roots have no connectors to continue)
		if len(node.Children) > 0 {
			var newAncestry []bool
			if depth > 0 {
				newAncestry = append(ancestry, isLast)
			}
			renderNodes(sb, node.Children, depth+1, newAncestry, cfg, renderCfg)
		}
	}
}

// renderNode renders a single tree node with tree connectors.
// depth 0 = root (no connector), depth 1+ = nested (has connector)
// ancestry tracks whether each parent level was a last child (true = last, no continuation line needed)
func renderNode(sb *strings.Builder, node *TreeNode, depth int, isLast bool, ancestry []bool, cfg *config.Config, renderCfg treeRenderConfig) {
	b := node.Bean

	// Build tree prefix from ancestry
	var prefix string
	if depth > 0 {
		for _, wasLast := range ancestry {
			if wasLast {
				prefix += treeSpace
			} else {
				prefix += treePipe
			}
		}
		if isLast {
			prefix += treeLastBranch
		} else {
			prefix += treeBranch
		}
	}

	// Get colors from config
	colors := cfg.GetBeanColors(b.Status, b.Type, b.Priority)

	// Use shared RenderBeanRow function with responsive columns
	row := RenderBeanRow(b.ID, b.Status, b.Type, b.Title, BeanRowConfig{
		StatusColor:     colors.StatusColor,
		TypeColor:       colors.TypeColor,
		PriorityColor:   colors.PriorityColor,
		Priority:        b.Priority,
		IsArchive:       colors.IsArchive,
		MaxTitleWidth:   renderCfg.titleWidth,
		ShowCursor:      false,
		Tags:            b.Tags,
		ShowTags:        renderCfg.cols.ShowTags,
		TagsColWidth:    renderCfg.cols.Tags,
		MaxTags:         renderCfg.cols.MaxTags,
		TreePrefix:      prefix,
		Dimmed:          !node.Matched,
		IDColWidth:      renderCfg.treeColWidth,
		ImplicitStatus: node.ImplicitStatus,
		Blocked:         node.Blocked,
	})

	sb.WriteString(row)
	sb.WriteString("\n")
}

// FlatItem represents a flattened tree node with rendering context.
// Used by TUI to render tree structure in a flat list.
type FlatItem struct {
	Bean            *bean.Bean
	Depth           int    // 0 = root, 1+ = nested
	IsLast          bool   // last child at this level
	Matched         bool   // true if bean matched filter (vs. shown for context)
	TreePrefix      string // pre-computed tree prefix (e.g., "  └─")
	ImplicitStatus string // implicit terminal status from an ancestor, if any
	Blocked         bool   // true if this bean has at least one active blocker
}

// FlattenTree converts a tree into a flat slice with tree context preserved.
// Each item includes the pre-computed tree prefix for rendering.
func FlattenTree(nodes []*TreeNode) []FlatItem {
	var items []FlatItem
	flattenNodes(nodes, 0, nil, &items)
	return items
}

// flattenNodes recursively flattens tree nodes.
// ancestry tracks whether each parent level was a last child (true = last, no continuation line needed)
func flattenNodes(nodes []*TreeNode, depth int, ancestry []bool, items *[]FlatItem) {
	for i, node := range nodes {
		isLast := i == len(nodes)-1

		// Compute tree prefix
		var prefix string
		if depth > 0 {
			// Build prefix from ancestry - each level adds either │ or space
			for _, wasLast := range ancestry {
				if wasLast {
					prefix += treeSpace // parent was last child, no continuation line
				} else {
					prefix += treePipe // parent has more siblings, show continuation line
				}
			}
			// Add connector for this node
			if isLast {
				prefix += treeLastBranch
			} else {
				prefix += treeBranch
			}
		}

		*items = append(*items, FlatItem{
			Bean:            node.Bean,
			Depth:           depth,
			IsLast:          isLast,
			Matched:         node.Matched,
			TreePrefix:      prefix,
			ImplicitStatus: node.ImplicitStatus,
			Blocked:         node.Blocked,
		})

		// Recurse into children, passing updated ancestry
		// Only add to ancestry when depth > 0 (roots have no connectors to continue)
		if len(node.Children) > 0 {
			var newAncestry []bool
			if depth > 0 {
				newAncestry = append(ancestry, isLast)
			}
			flattenNodes(node.Children, depth+1, newAncestry, items)
		}
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
