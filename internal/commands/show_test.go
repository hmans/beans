package commands

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/hmans/beans/pkg/bean"
	"github.com/hmans/beans/pkg/beancore"
	"github.com/hmans/beans/pkg/beangraph"
	"github.com/hmans/beans/pkg/config"
)

func TestFormatRelationships_ClosedAncestor(t *testing.T) {
	tests := []struct {
		name         string
		bean         *bean.Bean
		closedStatus string
		closedFrom   string
		want         string
	}{
		{
			name: "open parent",
			bean: &bean.Bean{ID: "t1", Parent: "e1"},
			want: "parent: e1",
		},
		{
			name:         "closed parent",
			bean:         &bean.Bean{ID: "t1", Parent: "e1"},
			closedStatus: "completed",
			closedFrom:   "e1",
			want:         "parent: e1 !completed",
		},
		{
			name:         "closed grandparent",
			bean:         &bean.Bean{ID: "t1", Parent: "e1"},
			closedStatus: "scrapped",
			closedFrom:   "m1",
			want:         "parent: e1\nancestor: m1 !scrapped",
		},
		{
			name:         "closed parent with blocking",
			bean:         &bean.Bean{ID: "t1", Parent: "e1", Blocking: []string{"t2"}},
			closedStatus: "completed",
			closedFrom:   "e1",
			want:         "parent: e1 !completed\nblocking: t2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ansi.Strip(formatRelationships(tt.bean, nil, tt.closedStatus, tt.closedFrom))
			if got != tt.want {
				t.Errorf("got:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestJSONBeansBlockedBy(t *testing.T) {
	c := beancore.New(filepath.Join(t.TempDir(), ".beans"), config.Default())
	if err := os.MkdirAll(c.Root(), 0755); err != nil {
		t.Fatal(err)
	}
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	for _, b := range []*bean.Bean{
		{ID: "own", Title: "Own", Status: "todo"},
		{ID: "done", Title: "Done", Status: "completed"},
		{ID: "target", Title: "Target", Status: "todo", BlockedBy: []string{"own", "done"}},
		{ID: "linker", Title: "Linker", Status: "todo", Blocking: []string{"target"}},
	} {
		b.Slug = bean.Slugify(b.Title)
		if err := c.Create(b); err != nil {
			t.Fatal(err)
		}
	}
	resolver := &beangraph.CoreResolver{Core: c}
	target, err := c.Get("target")
	if err != nil {
		t.Fatal(err)
	}

	views := jsonBeans(resolver, []*bean.Bean{target}, true)
	got := slices.Sorted(slices.Values(views[0].BlockedBy))
	if want := []string{"done", "linker", "own"}; !slices.Equal(got, want) {
		t.Errorf("blocked_by = %v, want %v", got, want)
	}
	if views[0].ETag != target.ETag() {
		t.Errorf("etag = %s, want the file's %s", views[0].ETag, target.ETag())
	}
}

func TestFormatRelationships_BlockedBy(t *testing.T) {
	oldCfg := cfg
	defer func() { cfg = oldCfg }()
	cfg = config.Default()
	b := &bean.Bean{ID: "t1", Parent: "e1", Blocking: []string{"t2"}}
	blockedBy := []*bean.Bean{
		{ID: "b1", Status: "todo"},
		{ID: "b2", Status: "completed"},
		{ID: "b3", Status: "scrapped"},
	}
	got := ansi.Strip(formatRelationships(b, blockedBy, "", ""))
	want := "parent: e1\nblocking: t2\nblocked by: b1\nblocked by: b2 completed\nblocked by: b3 scrapped"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatProperties(t *testing.T) {
	got := ansi.Strip(formatProperties(map[string]string{"zeta": "z", "github_issue": "#135"}))
	want := "github_issue: #135\nzeta: z"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}
