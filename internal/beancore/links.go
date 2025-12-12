package beancore

import (
	"github.com/hmans/beans/internal/bean"
)

// IncomingLink represents a link from another bean to a target bean.
type IncomingLink struct {
	FromBean *bean.Bean
	LinkType string
}

// BrokenLink represents a link to a non-existent bean.
type BrokenLink struct {
	BeanID   string `json:"bean_id"`
	LinkType string `json:"link_type"`
	Target   string `json:"target"`
}

// SelfLink represents a bean linking to itself.
type SelfLink struct {
	BeanID   string `json:"bean_id"`
	LinkType string `json:"link_type"`
}

// Cycle represents a circular dependency in links.
type Cycle struct {
	LinkType string   `json:"link_type"`
	Path     []string `json:"path"`
}

// LinkCheckResult contains all link validation issues found.
type LinkCheckResult struct {
	BrokenLinks []BrokenLink `json:"broken_links"`
	SelfLinks   []SelfLink   `json:"self_links"`
	Cycles      []Cycle      `json:"cycles"`
}

// HasIssues returns true if any link issues were found.
func (r *LinkCheckResult) HasIssues() bool {
	return len(r.BrokenLinks) > 0 || len(r.SelfLinks) > 0 || len(r.Cycles) > 0
}

// TotalIssues returns the total count of all issues.
func (r *LinkCheckResult) TotalIssues() int {
	return len(r.BrokenLinks) + len(r.SelfLinks) + len(r.Cycles)
}

// getAllLinkTargets returns all link targets from a bean with their types.
// Returns a slice of (linkType, target) tuples.
func getAllLinkTargets(b *bean.Bean) [][2]string {
	var result [][2]string

	// Hierarchy links (single targets)
	if b.Milestone != "" {
		result = append(result, [2]string{"milestone", b.Milestone})
	}
	if b.Epic != "" {
		result = append(result, [2]string{"epic", b.Epic})
	}
	if b.Feature != "" {
		result = append(result, [2]string{"feature", b.Feature})
	}

	// Relationship links (multiple targets)
	for _, target := range b.Blocks {
		result = append(result, [2]string{"blocks", target})
	}
	for _, target := range b.Related {
		result = append(result, [2]string{"related", target})
	}
	for _, target := range b.Duplicates {
		result = append(result, [2]string{"duplicates", target})
	}

	return result
}

// FindIncomingLinks returns all beans that link TO the given bean ID.
func (c *Core) FindIncomingLinks(targetID string) []IncomingLink {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var result []IncomingLink
	for _, b := range c.beans {
		for _, link := range getAllLinkTargets(b) {
			if link[1] == targetID {
				result = append(result, IncomingLink{
					FromBean: b,
					LinkType: link[0],
				})
			}
		}
	}
	return result
}

// isHierarchyLinkType returns true if the link type represents a hierarchy relationship.
func isHierarchyLinkType(linkType string) bool {
	switch linkType {
	case "blocks", "milestone", "epic", "feature":
		return true
	default:
		return false
	}
}

// DetectCycle checks if adding a link from fromID to toID would create a cycle.
// Checks hierarchical link types: blocks, milestone, epic, feature.
// Returns the cycle path if a cycle would be created, nil otherwise.
func (c *Core) DetectCycle(fromID, linkType, toID string) []string {
	// Only check hierarchical link types for cycles
	if !isHierarchyLinkType(linkType) {
		return nil
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	// Build adjacency list for the specific link type
	// Adding edge: fromID -> toID
	// Check if there's already a path from toID back to fromID
	visited := make(map[string]bool)
	path := []string{fromID, toID}

	return c.findPathToTarget(toID, fromID, linkType, visited, path)
}

// findPathToTarget uses DFS to find if there's a path from current to target.
// Returns the path if found, nil otherwise.
func (c *Core) findPathToTarget(current, target, linkType string, visited map[string]bool, path []string) []string {
	if current == target {
		return path
	}

	if visited[current] {
		return nil
	}
	visited[current] = true

	b, ok := c.beans[current]
	if !ok {
		return nil
	}

	// Get targets for the specific link type
	var targets []string
	switch linkType {
	case "blocks":
		targets = b.Blocks
	case "milestone":
		if b.Milestone != "" {
			targets = []string{b.Milestone}
		}
	case "epic":
		if b.Epic != "" {
			targets = []string{b.Epic}
		}
	case "feature":
		if b.Feature != "" {
			targets = []string{b.Feature}
		}
	}

	for _, t := range targets {
		newPath := append(path, t)
		if result := c.findPathToTarget(t, target, linkType, visited, newPath); result != nil {
			return result
		}
	}

	return nil
}

// CheckAllLinks validates all links across all beans.
func (c *Core) CheckAllLinks() *LinkCheckResult {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := &LinkCheckResult{
		BrokenLinks: []BrokenLink{},
		SelfLinks:   []SelfLink{},
		Cycles:      []Cycle{},
	}

	// Check for broken links and self-references
	for _, b := range c.beans {
		for _, link := range getAllLinkTargets(b) {
			linkType, target := link[0], link[1]

			// Check for self-reference
			if target == b.ID {
				result.SelfLinks = append(result.SelfLinks, SelfLink{
					BeanID:   b.ID,
					LinkType: linkType,
				})
				continue
			}

			// Check if target exists
			if _, ok := c.beans[target]; !ok {
				result.BrokenLinks = append(result.BrokenLinks, BrokenLink{
					BeanID:   b.ID,
					LinkType: linkType,
					Target:   target,
				})
			}
		}
	}

	// Check for cycles in blocks links only
	cycles := c.findCycles("blocks")
	result.Cycles = append(result.Cycles, cycles...)

	return result
}

// findCycles detects all cycles for a specific link type using DFS.
func (c *Core) findCycles(linkType string) []Cycle {
	var cycles []Cycle
	visited := make(map[string]bool)
	inStack := make(map[string]bool)
	seenCycles := make(map[string]bool) // To avoid duplicate cycle reports

	var dfs func(id string, path []string)
	dfs = func(id string, path []string) {
		if inStack[id] {
			// Found a cycle - find where the cycle starts
			cycleStart := -1
			for i, p := range path {
				if p == id {
					cycleStart = i
					break
				}
			}
			if cycleStart >= 0 {
				cyclePath := append(path[cycleStart:], id)
				// Create a canonical key to avoid duplicate cycles
				key := canonicalCycleKey(cyclePath)
				if !seenCycles[key] {
					seenCycles[key] = true
					cycles = append(cycles, Cycle{
						LinkType: linkType,
						Path:     cyclePath,
					})
				}
			}
			return
		}

		if visited[id] {
			return
		}

		visited[id] = true
		inStack[id] = true

		b, ok := c.beans[id]
		if ok {
			// Get targets for the specific link type
			var targets []string
			if linkType == "blocks" {
				targets = b.Blocks
			}

			for _, t := range targets {
				dfs(t, append(path, id))
			}
		}

		inStack[id] = false
	}

	for id := range c.beans {
		if !visited[id] {
			dfs(id, nil)
		}
	}

	return cycles
}

// canonicalCycleKey creates a unique key for a cycle to detect duplicates.
// It normalizes the cycle by starting from the smallest ID.
func canonicalCycleKey(path []string) string {
	if len(path) <= 1 {
		return ""
	}

	// Remove the duplicate end element (cycle closes back)
	cycle := path[:len(path)-1]

	// Find the minimum element to use as start
	minIdx := 0
	for i, id := range cycle {
		if id < cycle[minIdx] {
			minIdx = i
		}
	}

	// Rotate to start from minimum
	key := ""
	for i := 0; i < len(cycle); i++ {
		idx := (minIdx + i) % len(cycle)
		if i > 0 {
			key += "->"
		}
		key += cycle[idx]
	}

	return key
}

// RemoveLinksTo removes all links pointing to the given target ID from all beans.
// Returns the number of links removed.
func (c *Core) RemoveLinksTo(targetID string) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	removed := 0
	for _, b := range c.beans {
		changed := false

		// Check hierarchy links
		if b.Milestone == targetID {
			b.Milestone = ""
			changed = true
			removed++
		}
		if b.Epic == targetID {
			b.Epic = ""
			changed = true
			removed++
		}
		if b.Feature == targetID {
			b.Feature = ""
			changed = true
			removed++
		}

		// Check relationship links
		originalBlocks := len(b.Blocks)
		b.Blocks = removeFromSlice(b.Blocks, targetID)
		if len(b.Blocks) < originalBlocks {
			changed = true
			removed += originalBlocks - len(b.Blocks)
		}

		originalRelated := len(b.Related)
		b.Related = removeFromSlice(b.Related, targetID)
		if len(b.Related) < originalRelated {
			changed = true
			removed += originalRelated - len(b.Related)
		}

		originalDuplicates := len(b.Duplicates)
		b.Duplicates = removeFromSlice(b.Duplicates, targetID)
		if len(b.Duplicates) < originalDuplicates {
			changed = true
			removed += originalDuplicates - len(b.Duplicates)
		}

		if changed {
			if err := c.saveToDisk(b); err != nil {
				return removed, err
			}
		}
	}

	return removed, nil
}

// removeFromSlice removes all occurrences of target from slice.
func removeFromSlice(slice []string, target string) []string {
	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if s != target {
			result = append(result, s)
		}
	}
	return result
}

// FixBrokenLinks removes all broken links (links to non-existent beans) and self-references.
// Returns the number of issues fixed.
func (c *Core) FixBrokenLinks() (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	fixed := 0
	for _, b := range c.beans {
		changed := false

		// Fix hierarchy links
		if b.Milestone != "" && (b.Milestone == b.ID || c.beans[b.Milestone] == nil) {
			b.Milestone = ""
			changed = true
			fixed++
		}
		if b.Epic != "" && (b.Epic == b.ID || c.beans[b.Epic] == nil) {
			b.Epic = ""
			changed = true
			fixed++
		}
		if b.Feature != "" && (b.Feature == b.ID || c.beans[b.Feature] == nil) {
			b.Feature = ""
			changed = true
			fixed++
		}

		// Fix relationship links
		originalBlocks := len(b.Blocks)
		b.Blocks = filterValidLinks(b.Blocks, b.ID, c.beans)
		if len(b.Blocks) < originalBlocks {
			changed = true
			fixed += originalBlocks - len(b.Blocks)
		}

		originalRelated := len(b.Related)
		b.Related = filterValidLinks(b.Related, b.ID, c.beans)
		if len(b.Related) < originalRelated {
			changed = true
			fixed += originalRelated - len(b.Related)
		}

		originalDuplicates := len(b.Duplicates)
		b.Duplicates = filterValidLinks(b.Duplicates, b.ID, c.beans)
		if len(b.Duplicates) < originalDuplicates {
			changed = true
			fixed += originalDuplicates - len(b.Duplicates)
		}

		if changed {
			if err := c.saveToDisk(b); err != nil {
				return fixed, err
			}
		}
	}

	return fixed, nil
}

// filterValidLinks returns only links that exist and are not self-references.
func filterValidLinks(links []string, selfID string, beans map[string]*bean.Bean) []string {
	result := make([]string, 0, len(links))
	for _, target := range links {
		// Skip self-references
		if target == selfID {
			continue
		}
		// Skip broken links (target doesn't exist)
		if _, ok := beans[target]; !ok {
			continue
		}
		result = append(result, target)
	}
	return result
}
