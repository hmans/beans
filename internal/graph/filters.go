package graph

import (
	"hmans.dev/beans/internal/bean"
	"hmans.dev/beans/internal/beancore"
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

// filterByOutgoingLinks filters beans to include only those with outgoing links of the given types.
func filterByOutgoingLinks(beans []*bean.Bean, linkTypes []string) []*bean.Bean {
	typeSet := make(map[string]bool, len(linkTypes))
	for _, t := range linkTypes {
		typeSet[t] = true
	}

	var result []*bean.Bean
	for _, b := range beans {
		for _, link := range b.Links {
			if typeSet[link.Type] {
				result = append(result, b)
				break
			}
		}
	}
	return result
}

// excludeByOutgoingLinks filters beans to exclude those with outgoing links of the given types.
func excludeByOutgoingLinks(beans []*bean.Bean, linkTypes []string) []*bean.Bean {
	typeSet := make(map[string]bool, len(linkTypes))
	for _, t := range linkTypes {
		typeSet[t] = true
	}

	var result []*bean.Bean
outer:
	for _, b := range beans {
		for _, link := range b.Links {
			if typeSet[link.Type] {
				continue outer
			}
		}
		result = append(result, b)
	}
	return result
}

// filterByIncomingLinks filters beans to include only those that are targets of links of the given types.
func filterByIncomingLinks(beans []*bean.Bean, linkTypes []string, core *beancore.Core) []*bean.Bean {
	typeSet := make(map[string]bool, len(linkTypes))
	for _, t := range linkTypes {
		typeSet[t] = true
	}

	var result []*bean.Bean
	for _, b := range beans {
		incoming := core.FindIncomingLinks(b.ID)
		for _, link := range incoming {
			if typeSet[link.LinkType] {
				result = append(result, b)
				break
			}
		}
	}
	return result
}

// excludeByIncomingLinks filters beans to exclude those that are targets of links of the given types.
func excludeByIncomingLinks(beans []*bean.Bean, linkTypes []string, core *beancore.Core) []*bean.Bean {
	typeSet := make(map[string]bool, len(linkTypes))
	for _, t := range linkTypes {
		typeSet[t] = true
	}

	var result []*bean.Bean
outer:
	for _, b := range beans {
		incoming := core.FindIncomingLinks(b.ID)
		for _, link := range incoming {
			if typeSet[link.LinkType] {
				continue outer
			}
		}
		result = append(result, b)
	}
	return result
}
