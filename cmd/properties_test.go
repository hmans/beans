package cmd

import (
	"testing"
)

func TestParsePropertyFlags(t *testing.T) {
	tests := []struct {
		name    string
		flags   []string
		wantErr bool
		check   func(map[string]interface{}) bool
	}{
		{
			name:  "string value",
			flags: []string{"author=alice"},
			check: func(m map[string]interface{}) bool {
				return m["author"] == "alice"
			},
		},
		{
			name:  "integer value",
			flags: []string{"estimate=3"},
			check: func(m map[string]interface{}) bool {
				return m["estimate"] == 3
			},
		},
		{
			name:  "boolean value",
			flags: []string{"reviewed=true"},
			check: func(m map[string]interface{}) bool {
				return m["reviewed"] == true
			},
		},
		{
			name:  "float value",
			flags: []string{"score=4.5"},
			check: func(m map[string]interface{}) bool {
				return m["score"] == 4.5
			},
		},
		{
			name:  "empty value",
			flags: []string{"note="},
			check: func(m map[string]interface{}) bool {
				return m["note"] == ""
			},
		},
		{
			name:  "value with equals sign",
			flags: []string{"formula=a=b"},
			check: func(m map[string]interface{}) bool {
				return m["formula"] == "a=b"
			},
		},
		{
			name:  "multiple flags",
			flags: []string{"author=alice", "estimate=3", "reviewed=true"},
			check: func(m map[string]interface{}) bool {
				return m["author"] == "alice" && m["estimate"] == 3 && m["reviewed"] == true
			},
		},
		{
			name:    "missing equals sign",
			flags:   []string{"badformat"},
			wantErr: true,
		},
		{
			name:    "empty key",
			flags:   []string{"=value"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsePropertyFlags(tt.flags)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.check != nil && !tt.check(got) {
				t.Errorf("check failed, got: %v", got)
			}
		})
	}
}
