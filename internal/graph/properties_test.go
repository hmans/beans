package graph

import (
	"context"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hmans/beans/pkg/beangraph/model"
)

func props(kv ...string) []*model.PropertyInput {
	result := make([]*model.PropertyInput, 0, len(kv)/2)
	for i := 0; i < len(kv); i += 2 {
		result = append(result, &model.PropertyInput{Key: kv[i], Value: kv[i+1]})
	}
	return result
}

func TestMutationCreateBeanWithProperties(t *testing.T) {
	resolver, core := setupTestResolver(t)
	ctx := context.Background()
	mr := resolver.Mutation()

	got, err := mr.CreateBean(ctx, model.CreateBeanInput{
		Title:      "With props",
		Properties: props("github_issue", "#135", "estimate", "3"),
	})
	if err != nil {
		t.Fatalf("CreateBean() error = %v", err)
	}
	want := map[string]string{"github_issue": "#135", "estimate": "3"}
	if !maps.Equal(got.Properties, want) {
		t.Errorf("Properties = %v, want %v", got.Properties, want)
	}

	content, err := os.ReadFile(filepath.Join(core.Root(), got.Path))
	if err != nil {
		t.Fatalf("reading bean file: %v", err)
	}
	if !strings.Contains(string(content), "github_issue: '#135'") {
		t.Errorf("bean file does not contain property:\n%s", content)
	}

	if _, err := mr.CreateBean(ctx, model.CreateBeanInput{
		Title:      "Bad key",
		Properties: props("Bad Key", "x"),
	}); err == nil {
		t.Error("CreateBean() with invalid property key: want error, got nil")
	}
}

func TestMutationUpdateBeanProperties(t *testing.T) {
	tests := []struct {
		name    string
		initial map[string]string
		set     []*model.PropertyInput
		remove  []string
		want    map[string]string
		wantErr bool
	}{
		{
			name: "set on bean without properties",
			set:  props("a", "1"),
			want: map[string]string{"a": "1"},
		},
		{
			name:    "overwrite and add",
			initial: map[string]string{"a": "1", "b": "2"},
			set:     props("a", "10", "c", "3"),
			want:    map[string]string{"a": "10", "b": "2", "c": "3"},
		},
		{
			name:    "remove existing and missing key",
			initial: map[string]string{"a": "1", "b": "2"},
			remove:  []string{"a", "missing"},
			want:    map[string]string{"b": "2"},
		},
		{
			name:    "remove last key clears map",
			initial: map[string]string{"a": "1"},
			remove:  []string{"a"},
			want:    nil,
		},
		{
			name:    "untouched without property input",
			initial: map[string]string{"a": "1"},
			want:    map[string]string{"a": "1"},
		},
		{
			name:    "invalid key leaves properties unchanged",
			initial: map[string]string{"a": "1"},
			set:     props("b", "2", "Bad", "x"),
			want:    map[string]string{"a": "1"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver, core := setupTestResolver(t)
			ctx := context.Background()
			b := createTestBean(t, core, "prop-test", "Prop Test", "todo")
			b.Properties = maps.Clone(tt.initial)

			title := "Updated"
			got, err := resolver.Mutation().UpdateBean(ctx, "prop-test", model.UpdateBeanInput{
				Title:            &title,
				SetProperties:    tt.set,
				RemoveProperties: tt.remove,
			})
			if tt.wantErr {
				if err == nil {
					t.Fatal("UpdateBean() want error, got nil")
				}
				got = b
			} else if err != nil {
				t.Fatalf("UpdateBean() error = %v", err)
			}
			if !maps.Equal(got.Properties, tt.want) {
				t.Errorf("Properties = %v, want %v", got.Properties, tt.want)
			}
			if tt.want == nil && got.Properties != nil {
				t.Errorf("Properties = %#v, want nil", got.Properties)
			}
		})
	}
}

func TestBeanPropertiesResolverSortsByKey(t *testing.T) {
	resolver, core := setupTestResolver(t)
	b := createTestBean(t, core, "sort-test", "Sort Test", "todo")
	b.Properties = map[string]string{"zeta": "z", "alpha": "a", "mid": "m"}

	got, err := resolver.Bean().Properties(context.Background(), b)
	if err != nil {
		t.Fatalf("Properties() error = %v", err)
	}
	var keys []string
	for _, p := range got {
		keys = append(keys, p.Key)
	}
	if strings.Join(keys, ",") != "alpha,mid,zeta" {
		t.Errorf("keys = %v, want [alpha mid zeta]", keys)
	}

	empty := createTestBean(t, core, "empty-test", "Empty", "todo")
	got, err = resolver.Bean().Properties(context.Background(), empty)
	if err != nil || got == nil || len(got) != 0 {
		t.Errorf("Properties() on bean without properties = %v, %v; want empty non-nil slice", got, err)
	}
}
