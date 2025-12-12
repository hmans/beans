package graph

import (
	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/beancore"
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

// filterByHasParent filters beans to include only those with a parent.
func filterByHasParent(beans []*bean.Bean) []*bean.Bean {
	var result []*bean.Bean
	for _, b := range beans {
		if b.Parent != "" {
			result = append(result, b)
		}
	}
	return result
}

// filterByNoParent filters beans to include only those without a parent.
func filterByNoParent(beans []*bean.Bean) []*bean.Bean {
	var result []*bean.Bean
	for _, b := range beans {
		if b.Parent == "" {
			result = append(result, b)
		}
	}
	return result
}

// filterByParentID filters beans with specific parent ID.
func filterByParentID(beans []*bean.Bean, parentID string) []*bean.Bean {
	var result []*bean.Bean
	for _, b := range beans {
		if b.Parent == parentID {
			result = append(result, b)
		}
	}
	return result
}

// filterByHasBlocks filters beans that block other beans.
func filterByHasBlocks(beans []*bean.Bean) []*bean.Bean {
	var result []*bean.Bean
	for _, b := range beans {
		if len(b.Blocks) > 0 {
			result = append(result, b)
		}
	}
	return result
}

// filterByNoBlocks filters beans that don't block other beans.
func filterByNoBlocks(beans []*bean.Bean) []*bean.Bean {
	var result []*bean.Bean
	for _, b := range beans {
		if len(b.Blocks) == 0 {
			result = append(result, b)
		}
	}
	return result
}

// filterByIsBlocked filters beans that are blocked by others.
func filterByIsBlocked(beans []*bean.Bean, core *beancore.Core) []*bean.Bean {
	var result []*bean.Bean
	for _, b := range beans {
		incoming := core.FindIncomingLinks(b.ID)
		for _, link := range incoming {
			if link.LinkType == "blocks" {
				result = append(result, b)
				break
			}
		}
	}
	return result
}
