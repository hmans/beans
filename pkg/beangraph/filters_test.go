package beangraph

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hmans/beans/pkg/bean"
	"github.com/hmans/beans/pkg/beancore"
	"github.com/hmans/beans/pkg/beangraph/model"
	"github.com/hmans/beans/pkg/config"
)

// newFilterTestCore builds a fresh on-disk Core for filter tests.
func newFilterTestCore(t *testing.T) *beancore.Core {
	t.Helper()
	tmpDir := t.TempDir()
	beansDir := filepath.Join(tmpDir, ".beans")
	if err := os.MkdirAll(beansDir, 0755); err != nil {
		t.Fatalf("failed to create test .beans dir: %v", err)
	}
	c := beancore.New(beansDir, config.Default())
	if err := c.Load(); err != nil {
		t.Fatalf("failed to load core: %v", err)
	}
	return c
}

func createFilterBean(t *testing.T, c *beancore.Core, b *bean.Bean) *bean.Bean {
	t.Helper()
	if b.Slug == "" {
		b.Slug = bean.Slugify(b.Title)
	}
	if err := c.Create(b); err != nil {
		t.Fatalf("failed to create test bean %q: %v", b.ID, err)
	}
	return b
}

func filterIDs(beans []*bean.Bean) map[string]bool {
	out := make(map[string]bool, len(beans))
	for _, b := range beans {
		out[b.ID] = true
	}
	return out
}

// TestHasBlockingReverseEdges verifies that --has-blocking surfaces a blocker
// wired via the dependent's blocked_by field (reverse edge), not just via the
// blocker's own forward Blocking list. Also verifies active-only semantics:
// a blocker whose only dependent is completed must NOT surface.
func TestHasBlockingReverseEdges(t *testing.T) {
	c := newFilterTestCore(t)

	// Forward: fwd-blocker.Blocking = [fwd-dependent]
	createFilterBean(t, c, &bean.Bean{ID: "fwd-blocker", Title: "Forward Blocker", Status: "todo", Type: "task", Blocking: []string{"fwd-dependent"}})
	createFilterBean(t, c, &bean.Bean{ID: "fwd-dependent", Title: "Forward Dependent", Status: "todo", Type: "task"})

	// Reverse: rev-dependent.BlockedBy = [rev-blocker] (rev-blocker has no forward Blocking)
	createFilterBean(t, c, &bean.Bean{ID: "rev-blocker", Title: "Reverse Blocker", Status: "todo", Type: "task"})
	createFilterBean(t, c, &bean.Bean{ID: "rev-dependent", Title: "Reverse Dependent", Status: "todo", Type: "task", BlockedBy: []string{"rev-blocker"}})

	// Resolved dependent: dead-blocker only blocks a completed dependent — must NOT surface.
	createFilterBean(t, c, &bean.Bean{ID: "dead-blocker", Title: "Dead Blocker", Status: "todo", Type: "task", Blocking: []string{"done-dependent"}})
	createFilterBean(t, c, &bean.Bean{ID: "done-dependent", Title: "Done Dependent", Status: "completed", Type: "task"})

	// Plain bean: blocks nothing.
	createFilterBean(t, c, &bean.Bean{ID: "plain", Title: "Plain", Status: "todo", Type: "task"})

	all := c.All()

	t.Run("has-blocking surfaces forward and reverse, excludes resolved", func(t *testing.T) {
		yes := true
		got := ApplyFilter(all, &model.BeanFilter{HasBlocking: &yes}, c)
		ids := filterIDs(got)
		if !ids["fwd-blocker"] {
			t.Errorf("expected fwd-blocker in --has-blocking results, got %v", ids)
		}
		if !ids["rev-blocker"] {
			t.Errorf("expected rev-blocker (reverse blocked_by edge) in --has-blocking results, got %v", ids)
		}
		if ids["dead-blocker"] {
			t.Errorf("dead-blocker (only blocks a completed dependent) must NOT surface, got %v", ids)
		}
		if ids["plain"] || ids["fwd-dependent"] || ids["rev-dependent"] {
			t.Errorf("non-blockers must not surface, got %v", ids)
		}
	})

	t.Run("no-blocking is the exact negation", func(t *testing.T) {
		yes := true
		got := ApplyFilter(all, &model.BeanFilter{NoBlocking: &yes}, c)
		ids := filterIDs(got)
		if ids["fwd-blocker"] || ids["rev-blocker"] {
			t.Errorf("active blockers must NOT surface under --no-blocking, got %v", ids)
		}
		if !ids["dead-blocker"] {
			t.Errorf("dead-blocker (no active dependents) must surface under --no-blocking, got %v", ids)
		}
		if !ids["plain"] || !ids["fwd-dependent"] || !ids["rev-dependent"] {
			t.Errorf("non-blockers must surface under --no-blocking, got %v", ids)
		}
	})

	t.Run("blocking-id honors reverse edges", func(t *testing.T) {
		// rev-blocker blocks rev-dependent only via the reverse blocked_by edge.
		target := "rev-dependent"
		got := ApplyFilter(all, &model.BeanFilter{BlockingID: &target}, c)
		ids := filterIDs(got)
		if !ids["rev-blocker"] {
			t.Errorf("expected rev-blocker under --blocking rev-dependent, got %v", ids)
		}
		if len(ids) != 1 {
			t.Errorf("expected exactly rev-blocker, got %v", ids)
		}
	})
}
