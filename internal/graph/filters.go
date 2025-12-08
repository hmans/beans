package graph

import (
	"hmans.dev/beans/internal/bean"
	"hmans.dev/beans/internal/beancore"
	"hmans.dev/beans/internal/graph/model"
)

// filterByField filters beans to include only those where getter returns a value in values (OR logic).
func filterByField(beans []*bean.Bean, values []string, getter func(*bean.Bean) string) []*bean.Bean {
	valueSet := make(map[string]bool, len(values))
	for _, v := range values {
		valueSet[v] = true
	}

	var result []*bean.Bean
	for _, b := range beans {
		if valueSet[getter(b)] {
			result = append(result, b)
		}
	}
	return result
}

// excludeByField filters beans to exclude those where getter returns a value in values.
func excludeByField(beans []*bean.Bean, values []string, getter func(*bean.Bean) string) []*bean.Bean {
	valueSet := make(map[string]bool, len(values))
	for _, v := range values {
		valueSet[v] = true
	}

	var result []*bean.Bean
	for _, b := range beans {
		if !valueSet[getter(b)] {
			result = append(result, b)
		}
	}
	return result
}

// filterByPriority filters beans to include only those with matching priorities (OR logic).
// Empty priority in the bean is treated as "normal" for matching purposes.
func filterByPriority(beans []*bean.Bean, priorities []string) []*bean.Bean {
	prioritySet := make(map[string]bool, len(priorities))
	for _, p := range priorities {
		prioritySet[p] = true
	}

	var result []*bean.Bean
	for _, b := range beans {
		priority := b.Priority
		if priority == "" {
			priority = "normal"
		}
		if prioritySet[priority] {
			result = append(result, b)
		}
	}
	return result
}

// excludeByPriority filters beans to exclude those with matching priorities.
// Empty priority in the bean is treated as "normal" for matching purposes.
func excludeByPriority(beans []*bean.Bean, priorities []string) []*bean.Bean {
	prioritySet := make(map[string]bool, len(priorities))
	for _, p := range priorities {
		prioritySet[p] = true
	}

	var result []*bean.Bean
	for _, b := range beans {
		priority := b.Priority
		if priority == "" {
			priority = "normal"
		}
		if !prioritySet[priority] {
			result = append(result, b)
		}
	}
	return result
}

// filterByTags filters beans to include only those with any of the given tags (OR logic).
func filterByTags(beans []*bean.Bean, tags []string) []*bean.Bean {
	tagSet := make(map[string]bool, len(tags))
	for _, t := range tags {
		tagSet[t] = true
	}

	var result []*bean.Bean
	for _, b := range beans {
		for _, t := range b.Tags {
			if tagSet[t] {
				result = append(result, b)
				break
			}
		}
	}
	return result
}

// excludeByTags filters beans to exclude those with any of the given tags.
func excludeByTags(beans []*bean.Bean, tags []string) []*bean.Bean {
	tagSet := make(map[string]bool, len(tags))
	for _, t := range tags {
		tagSet[t] = true
	}

	var result []*bean.Bean
outer:
	for _, b := range beans {
		for _, t := range b.Tags {
			if tagSet[t] {
				continue outer
			}
		}
		result = append(result, b)
	}
	return result
}

// matchesLinkFilter checks if a bean link matches a LinkFilter.
// If filter.Target is nil, matches any target; otherwise matches specific target.
func matchesLinkFilter(link bean.Link, filter *model.LinkFilter) bool {
	if link.Type != filter.Type {
		return false
	}
	if filter.Target == nil {
		return true // type-only match
	}
	return link.Target == *filter.Target
}

// filterByOutgoingLinks filters beans to include only those with outgoing links matching the filters (OR logic).
func filterByOutgoingLinks(beans []*bean.Bean, filters []*model.LinkFilter) []*bean.Bean {
	if len(filters) == 0 {
		return beans
	}

	var result []*bean.Bean
	for _, b := range beans {
		matched := false
		for _, link := range b.Links {
			for _, f := range filters {
				if matchesLinkFilter(link, f) {
					matched = true
					break
				}
			}
			if matched {
				break
			}
		}
		if matched {
			result = append(result, b)
		}
	}
	return result
}

// excludeByOutgoingLinks filters beans to exclude those with outgoing links matching the filters.
func excludeByOutgoingLinks(beans []*bean.Bean, filters []*model.LinkFilter) []*bean.Bean {
	if len(filters) == 0 {
		return beans
	}

	var result []*bean.Bean
outer:
	for _, b := range beans {
		for _, link := range b.Links {
			for _, f := range filters {
				if matchesLinkFilter(link, f) {
					continue outer
				}
			}
		}
		result = append(result, b)
	}
	return result
}

// filterByIncomingLinks filters beans to include only those that are targets of links matching the filters (OR logic).
func filterByIncomingLinks(beans []*bean.Bean, filters []*model.LinkFilter, core *beancore.Core) []*bean.Bean {
	if len(filters) == 0 {
		return beans
	}

	var result []*bean.Bean
	for _, b := range beans {
		matched := false
		incoming := core.FindIncomingLinks(b.ID)
		for _, link := range incoming {
			for _, f := range filters {
				// For incoming links, we check the link type and optionally the source bean ID
				if link.LinkType != f.Type {
					continue
				}
				if f.Target == nil {
					// Type-only: this bean is targeted by a link of this type
					matched = true
					break
				}
				// Target specified: check if the link is from the specified bean
				if link.FromBean.ID == *f.Target {
					matched = true
					break
				}
			}
			if matched {
				break
			}
		}
		if matched {
			result = append(result, b)
		}
	}
	return result
}

// excludeByIncomingLinks filters beans to exclude those that are targets of links matching the filters.
func excludeByIncomingLinks(beans []*bean.Bean, filters []*model.LinkFilter, core *beancore.Core) []*bean.Bean {
	if len(filters) == 0 {
		return beans
	}

	var result []*bean.Bean
outer:
	for _, b := range beans {
		incoming := core.FindIncomingLinks(b.ID)
		for _, link := range incoming {
			for _, f := range filters {
				if link.LinkType != f.Type {
					continue
				}
				if f.Target == nil {
					// Type-only: exclude if this bean is targeted by a link of this type
					continue outer
				}
				// Target specified: exclude if the link is from the specified bean
				if link.FromBean.ID == *f.Target {
					continue outer
				}
			}
		}
		result = append(result, b)
	}
	return result
}
