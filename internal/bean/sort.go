package bean

import (
	"sort"
	"strings"
)

// SortByStatusPriorityAndType sorts beans by status order, then priority, then type, then title.
// This is the default sorting used by both CLI and TUI.
// Unrecognized statuses, priorities, and types are sorted last within their category.
// Beans without priority are treated as "normal" priority for sorting purposes.
func SortByStatusPriorityAndType(beans []*Bean, statusNames, priorityNames, typeNames []string) {
	statusOrder := make(map[string]int)
	for i, s := range statusNames {
		statusOrder[s] = i
	}
	priorityOrder := make(map[string]int)
	for i, p := range priorityNames {
		priorityOrder[p] = i
	}
	typeOrder := make(map[string]int)
	for i, t := range typeNames {
		typeOrder[t] = i
	}

	// Find the index of "normal" priority for beans without priority set
	normalPriorityOrder := len(priorityNames) // default to last if "normal" not found
	for i, p := range priorityNames {
		if p == "normal" {
			normalPriorityOrder = i
			break
		}
	}

	// Helper to get order with unrecognized values sorted last
	getStatusOrder := func(status string) int {
		if order, ok := statusOrder[status]; ok {
			return order
		}
		return len(statusNames) // Unrecognized statuses come last
	}
	getPriorityOrder := func(priority string) int {
		if priority == "" {
			return normalPriorityOrder // No priority = normal
		}
		if order, ok := priorityOrder[priority]; ok {
			return order
		}
		return len(priorityNames) // Unrecognized priorities come last
	}
	getTypeOrder := func(typ string) int {
		if order, ok := typeOrder[typ]; ok {
			return order
		}
		return len(typeNames) // Unrecognized types come last
	}

	sort.Slice(beans, func(i, j int) bool {
		// Primary: status order
		oi, oj := getStatusOrder(beans[i].Status), getStatusOrder(beans[j].Status)
		if oi != oj {
			return oi < oj
		}
		// Secondary: priority order
		pi, pj := getPriorityOrder(beans[i].Priority), getPriorityOrder(beans[j].Priority)
		if pi != pj {
			return pi < pj
		}
		// Tertiary: type order
		ti, tj := getTypeOrder(beans[i].Type), getTypeOrder(beans[j].Type)
		if ti != tj {
			return ti < tj
		}
		// Quaternary: title (case-insensitive) for stable, user-friendly ordering
		return strings.ToLower(beans[i].Title) < strings.ToLower(beans[j].Title)
	})
}
