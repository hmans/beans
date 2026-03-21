package beangraph

import (
	"context"
	"path/filepath"

	"github.com/hmans/beans/pkg/bean"
	"github.com/hmans/beans/pkg/beangraph/model"
	"github.com/hmans/beans/pkg/beancore"
)

// BeanIsDirty returns whether a bean has unsaved runtime changes.
func (r *CoreResolver) BeanIsDirty(ctx context.Context, obj *bean.Bean) (bool, error) {
	return r.Core.IsDirty(obj.ID), nil
}

// BeanWorktreeID returns the worktree ID for a bean, or nil if not linked.
func (r *CoreResolver) BeanWorktreeID(ctx context.Context, obj *bean.Bean) (*string, error) {
	wtPath := r.Core.WorktreeForBean(obj.ID)
	if wtPath == "" {
		return nil, nil
	}
	// Extract worktree ID from the path (last path component)
	id := filepath.Base(wtPath)
	return &id, nil
}

// BeanParentID returns the parent ID as a pointer, or nil if no parent.
func (r *CoreResolver) BeanParentID(ctx context.Context, obj *bean.Bean) (*string, error) {
	if obj.Parent == "" {
		return nil, nil
	}
	return &obj.Parent, nil
}

// BeanBlockingIds returns the blocking IDs slice.
func (r *CoreResolver) BeanBlockingIds(ctx context.Context, obj *bean.Bean) ([]string, error) {
	return obj.Blocking, nil
}

// BeanBlockedByIds returns the blocked-by IDs slice.
func (r *CoreResolver) BeanBlockedByIds(ctx context.Context, obj *bean.Bean) ([]string, error) {
	return obj.BlockedBy, nil
}

// BeanBlockedBy resolves the full list of beans blocking this one.
// Combines both directions: the bean's own blocked_by field AND incoming
// blocking links (other beans that list this bean in their blocking field).
func (r *CoreResolver) BeanBlockedBy(ctx context.Context, obj *bean.Bean, filter *model.BeanFilter) ([]*bean.Bean, error) {
	seen := make(map[string]bool)
	var result []*bean.Bean

	// 1. Resolve beans from the direct blocked_by field
	for _, blockerID := range obj.BlockedBy {
		if !seen[blockerID] {
			seen[blockerID] = true
			if blocker, err := r.Core.Get(blockerID); err == nil {
				result = append(result, blocker)
			}
		}
	}

	// 2. Resolve beans from incoming blocking links (other beans blocking this one)
	incoming := r.Core.FindIncomingLinks(obj.ID)
	for _, link := range incoming {
		if link.LinkType == "blocking" && !seen[link.FromBean.ID] {
			seen[link.FromBean.ID] = true
			result = append(result, link.FromBean)
		}
	}

	filtered := ApplyFilter(result, filter, r.Core)
	cfg := r.Core.Config()
	bean.SortByStatusPriorityAndType(filtered, cfg.StatusNames(), cfg.PriorityNames(), cfg.TypeNames())
	return filtered, nil
}

// BeanBlocking resolves the beans this bean is blocking.
func (r *CoreResolver) BeanBlocking(ctx context.Context, obj *bean.Bean, filter *model.BeanFilter) ([]*bean.Bean, error) {
	var result []*bean.Bean
	for _, targetID := range obj.Blocking {
		// Filter out broken links
		if target, err := r.Core.Get(targetID); err == nil {
			result = append(result, target)
		}
	}
	filtered := ApplyFilter(result, filter, r.Core)
	cfg := r.Core.Config()
	bean.SortByStatusPriorityAndType(filtered, cfg.StatusNames(), cfg.PriorityNames(), cfg.TypeNames())
	return filtered, nil
}

// BeanParent resolves the parent bean.
func (r *CoreResolver) BeanParent(ctx context.Context, obj *bean.Bean) (*bean.Bean, error) {
	if obj.Parent == "" {
		return nil, nil
	}
	// Filter out broken links
	parent, err := r.Core.Get(obj.Parent)
	if err == beancore.ErrNotFound {
		return nil, nil
	}
	return parent, err
}

// BeanChildren resolves the child beans.
func (r *CoreResolver) BeanChildren(ctx context.Context, obj *bean.Bean, filter *model.BeanFilter) ([]*bean.Bean, error) {
	incoming := r.Core.FindIncomingLinks(obj.ID)
	var result []*bean.Bean
	for _, link := range incoming {
		if link.LinkType == "parent" {
			result = append(result, link.FromBean)
		}
	}
	filtered := ApplyFilter(result, filter, r.Core)
	cfg := r.Core.Config()
	bean.SortByStatusPriorityAndType(filtered, cfg.StatusNames(), cfg.PriorityNames(), cfg.TypeNames())
	return filtered, nil
}

// BeanImplicitStatus returns the implicit status inherited from ancestors.
func (r *CoreResolver) BeanImplicitStatus(ctx context.Context, obj *bean.Bean) (*string, error) {
	status, _ := r.Core.ImplicitStatus(obj.ID)
	if status == "" {
		return nil, nil
	}
	return &status, nil
}

// BeanImplicitStatusFrom returns the ancestor ID from which implicit status is inherited.
func (r *CoreResolver) BeanImplicitStatusFrom(ctx context.Context, obj *bean.Bean) (*string, error) {
	_, fromID := r.Core.ImplicitStatus(obj.ID)
	if fromID == "" {
		return nil, nil
	}
	return &fromID, nil
}
