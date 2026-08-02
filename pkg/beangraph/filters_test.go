package beangraph

import (
	"testing"

	"github.com/hmans/beans/pkg/bean"
	"github.com/hmans/beans/pkg/beancore"
	"github.com/hmans/beans/pkg/beangraph/model"
	"github.com/hmans/beans/pkg/config"
)

func TestApplyFilterNormalizesShortParentID(t *testing.T) {
	core := beancore.New(t.TempDir(), config.DefaultWithPrefix("beans-"))
	parent := &bean.Bean{ID: "beans-parent1", Title: "Parent", Type: "epic"}
	child := &bean.Bean{ID: "beans-child1", Title: "Child", Type: "task", Parent: parent.ID}
	if err := core.Create(parent, beancore.WithPersist(false)); err != nil {
		t.Fatalf("create parent: %v", err)
	}
	if err := core.Create(child, beancore.WithPersist(false)); err != nil {
		t.Fatalf("create child: %v", err)
	}

	shortParentID := "parent1"
	filtered := ApplyFilter([]*bean.Bean{child}, &model.BeanFilter{ParentID: &shortParentID}, core)
	if len(filtered) != 1 || filtered[0].ID != child.ID {
		t.Fatalf("short parent ID filtered out child: got %#v", filtered)
	}
}
