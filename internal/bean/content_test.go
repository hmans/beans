package bean

import (
	"testing"
)

func TestReplaceOnce(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		old     string
		new     string
		want    string
		wantErr string
	}{
		{
			name: "simple replacement",
			text: "hello world",
			old:  "world",
			new:  "there",
			want: "hello there",
		},
		{
			name: "replace checkbox unchecked to checked",
			text: "## Tasks\n- [ ] Task 1\n- [ ] Task 2",
			old:  "- [ ] Task 1",
			new:  "- [x] Task 1",
			want: "## Tasks\n- [x] Task 1\n- [ ] Task 2",
		},
		{
			name: "delete text with empty new",
			text: "hello world",
			old:  " world",
			new:  "",
			want: "hello",
		},
		{
			name: "replace at start",
			text: "hello world",
			old:  "hello",
			new:  "hi",
			want: "hi world",
		},
		{
			name: "replace at end",
			text: "hello world",
			old:  "world",
			new:  "universe",
			want: "hello universe",
		},
		{
			name: "replace entire string",
			text: "hello",
			old:  "hello",
			new:  "goodbye",
			want: "goodbye",
		},
		{
			name: "replace with longer text",
			text: "a",
			old:  "a",
			new:  "abc",
			want: "abc",
		},
		{
			name: "replace multiline",
			text: "line1\nline2\nline3",
			old:  "line2",
			new:  "replaced",
			want: "line1\nreplaced\nline3",
		},
		{
			name:    "empty old string",
			text:    "hello",
			old:     "",
			new:     "world",
			wantErr: "old text cannot be empty",
		},
		{
			name:    "text not found",
			text:    "hello world",
			old:     "foo",
			new:     "bar",
			wantErr: "text not found in body",
		},
		{
			name:    "text found multiple times",
			text:    "hello hello",
			old:     "hello",
			new:     "hi",
			wantErr: "text found 2 times in body (must be unique)",
		},
		{
			name:    "text found three times",
			text:    "aaa",
			old:     "a",
			new:     "b",
			wantErr: "text found 3 times in body (must be unique)",
		},
		{
			name:    "empty text with non-empty old",
			text:    "",
			old:     "hello",
			new:     "world",
			wantErr: "text not found in body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReplaceOnce(tt.text, tt.old, tt.new)
			if tt.wantErr != "" {
				if err == nil {
					t.Errorf("ReplaceOnce() error = nil, wantErr %q", tt.wantErr)
					return
				}
				if err.Error() != tt.wantErr {
					t.Errorf("ReplaceOnce() error = %q, wantErr %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Errorf("ReplaceOnce() unexpected error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("ReplaceOnce() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAppendWithSeparator(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		addition string
		want     string
	}{
		{
			name:     "append to non-empty text",
			text:     "hello",
			addition: "world",
			want:     "hello\n\nworld",
		},
		{
			name:     "append to empty text",
			text:     "",
			addition: "world",
			want:     "world",
		},
		{
			name:     "append empty to non-empty text (no-op)",
			text:     "hello",
			addition: "",
			want:     "hello",
		},
		{
			name:     "append empty to empty text (no-op)",
			text:     "",
			addition: "",
			want:     "",
		},
		{
			name:     "text with trailing newline",
			text:     "hello\n",
			addition: "world",
			want:     "hello\n\nworld",
		},
		{
			name:     "text with multiple trailing newlines",
			text:     "hello\n\n\n",
			addition: "world",
			want:     "hello\n\nworld",
		},
		{
			name:     "multiline text",
			text:     "line1\nline2",
			addition: "line3",
			want:     "line1\nline2\n\nline3",
		},
		{
			name:     "multiline addition",
			text:     "header",
			addition: "line1\nline2",
			want:     "header\n\nline1\nline2",
		},
		{
			name:     "typical usage - adding notes section",
			text:     "## Tasks\n- [ ] Task 1",
			addition: "## Notes\n\nSome notes here",
			want:     "## Tasks\n- [ ] Task 1\n\n## Notes\n\nSome notes here",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AppendWithSeparator(tt.text, tt.addition)
			if got != tt.want {
				t.Errorf("AppendWithSeparator() = %q, want %q", got, tt.want)
			}
		})
	}
}
