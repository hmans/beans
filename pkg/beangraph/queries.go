package beangraph

import (
	"context"

	"github.com/hmans/beans/internal/gitutil"
	"github.com/hmans/beans/pkg/bean"
	"github.com/hmans/beans/pkg/beangraph/model"
	"github.com/hmans/beans/pkg/beancore"
)

// Bean returns a single bean by ID, or nil if not found.
func (r *CoreResolver) Bean(ctx context.Context, id string) (*bean.Bean, error) {
	b, err := r.Core.Get(id)
	if err == beancore.ErrNotFound {
		return nil, nil
	}
	return b, err
}

// Beans returns a filtered, sorted list of beans.
func (r *CoreResolver) Beans(ctx context.Context, filter *model.BeanFilter) ([]*bean.Bean, error) {
	var beans []*bean.Bean

	// If search filter is provided, start with search results
	if filter != nil && filter.Search != nil && *filter.Search != "" {
		searchResults, err := r.Core.Search(*filter.Search)
		if err != nil {
			return nil, err
		}
		beans = searchResults
	} else {
		beans = r.Core.All()
	}

	result := ApplyFilter(beans, filter, r.Core)

	// Sort using the same logic as CLI and TUI
	cfg := r.Core.Config()
	bean.SortByStatusPriorityAndType(result, cfg.StatusNames(), cfg.PriorityNames(), cfg.TypeNames())

	return result, nil
}

// ProjectName returns the configured project name.
func (r *CoreResolver) ProjectName(ctx context.Context) (string, error) {
	cfg := r.Core.Config()
	if cfg == nil {
		return "", nil
	}
	return cfg.GetProjectName(), nil
}

// MainBranch returns the current branch of the main repository.
func (r *CoreResolver) MainBranch(ctx context.Context, projectRoot string) (string, error) {
	if branch, ok := gitutil.CurrentBranch(projectRoot); ok {
		return branch, nil
	}
	return "main", nil
}
