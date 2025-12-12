package beancore

import (
	"fmt"
	"strings"

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

// FindIncomingLinks returns all beans that link TO the given bean ID.
func (c *Core) FindIncomingLinks(targetID string) []IncomingLink {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var result []IncomingLink
	for _, b := range c.beans {
		// Check parent link
		if b.Parent == targetID {
			result = append(result, IncomingLink{
				FromBean: b,
				LinkType: "parent",
			})
		}
		// Check blocking links
		for _, blocked := range b.Blocking {
			if blocked == targetID {
				result = append(result, IncomingLink{
					FromBean: b,
					LinkType: "blocking",
				})
			}
		}
	}
	return result
}

// DetectCycle checks if adding a link from fromID to toID would create a cycle.
// Only checks for blocking and parent link types.
// Returns the cycle path if a cycle would be created, nil otherwise.
func (c *Core) DetectCycle(fromID, linkType, toID string) []string {
	// Only check hierarchical link types
	if linkType != "blocking" && linkType != "parent" {
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

	// Get targets based on link type
	var targets []string
	switch linkType {
	case "parent":
		if b.Parent != "" {
			targets = []string{b.Parent}
		}
	case "blocking":
		targets = b.Blocking
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
		// Check parent link
		if b.Parent != "" {
			if b.Parent == b.ID {
				result.SelfLinks = append(result.SelfLinks, SelfLink{
					BeanID:   b.ID,
					LinkType: "parent",
				})
			} else if _, ok := c.beans[b.Parent]; !ok {
				result.BrokenLinks = append(result.BrokenLinks, BrokenLink{
					BeanID:   b.ID,
					LinkType: "parent",
					Target:   b.Parent,
				})
			}
		}

		// Check blocking links
		for _, blocked := range b.Blocking {
			if blocked == b.ID {
				result.SelfLinks = append(result.SelfLinks, SelfLink{
					BeanID:   b.ID,
					LinkType: "blocking",
				})
			} else if _, ok := c.beans[blocked]; !ok {
				result.BrokenLinks = append(result.BrokenLinks, BrokenLink{
					BeanID:   b.ID,
					LinkType: "blocking",
					Target:   blocked,
				})
			}
		}
	}

	// Check for cycles in blocking and parent links
	for _, linkType := range []string{"blocking", "parent"} {
		cycles := c.findCycles(linkType)
		result.Cycles = append(result.Cycles, cycles...)
	}

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
			// Get targets based on link type
			var targets []string
			switch linkType {
			case "parent":
				if b.Parent != "" {
					targets = []string{b.Parent}
				}
			case "blocking":
				targets = b.Blocking
			}

			for _, target := range targets {
				// Skip self-references (they're tracked separately as SelfLinks)
				if target == id {
					continue
				}
				dfs(target, append(path, id))
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

		// Remove parent link
		if b.Parent == targetID {
			b.Parent = ""
			changed = true
			removed++
		}

		// Remove blocking links
		originalBlockingLen := len(b.Blocking)
		b.RemoveBlocking(targetID)
		if len(b.Blocking) < originalBlockingLen {
			changed = true
			removed += originalBlockingLen - len(b.Blocking)
		}

		if changed {
			if err := c.saveToDisk(b); err != nil {
				return removed, err
			}
		}
	}

	return removed, nil
}

// FixBrokenLinks removes all broken links (links to non-existent beans) and self-references.
// Returns the number of issues fixed.
func (c *Core) FixBrokenLinks() (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	fixed := 0
	for _, b := range c.beans {
		changed := false

		// Fix parent link
		if b.Parent != "" {
			// Remove self-reference or broken link
			if b.Parent == b.ID {
				b.Parent = ""
				changed = true
				fixed++
			} else if _, ok := c.beans[b.Parent]; !ok {
				b.Parent = ""
				changed = true
				fixed++
			}
		}

		// Fix blocking links
		originalBlockingLen := len(b.Blocking)
		var newBlocking []string
		for _, blocked := range b.Blocking {
			// Skip self-references
			if blocked == b.ID {
				continue
			}
			// Skip broken links (target doesn't exist)
			if _, ok := c.beans[blocked]; !ok {
				continue
			}
			newBlocking = append(newBlocking, blocked)
		}
		if len(newBlocking) < originalBlockingLen {
			b.Blocking = newBlocking
			changed = true
			fixed += originalBlockingLen - len(newBlocking)
		}

		if changed {
			if err := c.saveToDisk(b); err != nil {
				return fixed, err
			}
		}
	}

	return fixed, nil
}

// ValidParentTypes returns the valid parent types for a given bean type.
// Returns nil if the bean type cannot have a parent.
func ValidParentTypes(beanType string) []string {
	switch beanType {
	case "milestone":
		return nil // milestones cannot have parents
	case "epic":
		return []string{"milestone"}
	case "feature":
		return []string{"milestone", "epic"}
	case "task", "bug":
		return []string{"milestone", "epic", "feature"}
	default:
		return []string{"milestone", "epic", "feature"} // default for unknown types
	}
}

// ValidateParent checks if a parent is valid for the given bean.
// Returns nil if valid, error otherwise.
func (c *Core) ValidateParent(b *bean.Bean, parentID string) error {
	if parentID == "" {
		return nil
	}

	validTypes := ValidParentTypes(b.Type)
	if validTypes == nil {
		return fmt.Errorf("%s beans cannot have a parent", b.Type)
	}

	parent, err := c.Get(parentID)
	if err != nil {
		return fmt.Errorf("parent bean not found: %s", parentID)
	}

	for _, t := range validTypes {
		if parent.Type == t {
			return nil
		}
	}

	return fmt.Errorf("%s beans can only have %s as parent, not %s",
		b.Type, joinWithOr(validTypes), parent.Type)
}

// joinWithOr joins strings with commas and "or" for the last element.
func joinWithOr(items []string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	case 2:
		return items[0] + " or " + items[1]
	default:
		return strings.Join(items[:len(items)-1], ", ") + ", or " + items[len(items)-1]
	}
}
