package bean

import (
	"maps"
	"strings"
	"testing"
)

func TestValidatePropertyKey(t *testing.T) {
	tests := []struct {
		key     string
		wantErr bool
	}{
		{"github_issue", false},
		{"estimate", false},
		{"review-by", false},
		{"v2", false},
		{"", true},
		{"GitHub", true},
		{"2fa", true},
		{"_private", true},
		{"my key", true},
		{"my.key", true},
		{"my:key", true},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			err := ValidatePropertyKey(tt.key)
			if tt.wantErr && err == nil {
				t.Errorf("ValidatePropertyKey(%q) = nil, want error", tt.key)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("ValidatePropertyKey(%q) = %v, want nil", tt.key, err)
			}
		})
	}
}

func TestParseProperties(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  map[string]string
	}{
		{
			name:  "no properties",
			input: "---\ntitle: Test\nstatus: todo\n---\n",
			want:  nil,
		},
		{
			name:  "string values",
			input: "---\ntitle: Test\nstatus: todo\nproperties:\n  github_issue: \"#135\"\n  author: alice@bob.com\n---\n",
			want:  map[string]string{"github_issue": "#135", "author": "alice@bob.com"},
		},
		{
			name:  "non-string scalars are read as strings",
			input: "---\ntitle: Test\nstatus: todo\nproperties:\n  estimate: 3\n  reviewed: true\n---\n",
			want:  map[string]string{"estimate": "3", "reviewed": "true"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := Parse(strings.NewReader(tt.input))
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}
			if !maps.Equal(b.Properties, tt.want) {
				t.Errorf("Properties = %v, want %v", b.Properties, tt.want)
			}
		})
	}
}

func TestRenderProperties(t *testing.T) {
	b := &Bean{
		Title:      "Test",
		Status:     "todo",
		Properties: map[string]string{"zeta": "last", "alpha": "first", "reviewed": "true"},
	}

	rendered, err := b.Render()
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	want := "properties:\n    alpha: first\n    reviewed: \"true\"\n    zeta: last\n"
	if !strings.Contains(string(rendered), want) {
		t.Errorf("rendered front matter missing sorted, quoted properties block %q:\n%s", want, rendered)
	}
}

func TestRenderWithoutPropertiesOmitsKey(t *testing.T) {
	for _, props := range []map[string]string{nil, {}} {
		b := &Bean{Title: "Test", Status: "todo", Properties: props}
		rendered, err := b.Render()
		if err != nil {
			t.Fatalf("Render error: %v", err)
		}
		if strings.Contains(string(rendered), "properties") {
			t.Errorf("Properties %#v: rendered output contains properties key:\n%s", props, rendered)
		}
	}
}

func TestPropertiesRoundtrip(t *testing.T) {
	original := &Bean{
		Title:      "Test",
		Status:     "todo",
		Properties: map[string]string{"github_issue": "#135", "estimate": "3", "note": "multi\nline"},
	}

	rendered, err := original.Render()
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}
	parsed, err := Parse(strings.NewReader(string(rendered)))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if !maps.Equal(parsed.Properties, original.Properties) {
		t.Errorf("Properties = %v, want %v", parsed.Properties, original.Properties)
	}
	if parsed.ETag() != original.ETag() {
		t.Errorf("ETag changed across roundtrip: %s != %s", parsed.ETag(), original.ETag())
	}
}
