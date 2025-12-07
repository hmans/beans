package cmd

import "testing"

func TestParseLink(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantType   string
		wantTarget string
		wantErr    bool
	}{
		{
			name:       "valid blocks link",
			input:      "blocks:abc123",
			wantType:   "blocks",
			wantTarget: "abc123",
			wantErr:    false,
		},
		{
			name:       "valid parent link",
			input:      "parent:epic-1",
			wantType:   "parent",
			wantTarget: "epic-1",
			wantErr:    false,
		},
		{
			name:       "valid related link",
			input:      "related:other-bean",
			wantType:   "related",
			wantTarget: "other-bean",
			wantErr:    false,
		},
		{
			name:       "valid duplicates link",
			input:      "duplicates:dup-id",
			wantType:   "duplicates",
			wantTarget: "dup-id",
			wantErr:    false,
		},
		{
			name:    "missing colon",
			input:   "blocksabc123",
			wantErr: true,
		},
		{
			name:    "empty type",
			input:   ":abc123",
			wantErr: true,
		},
		{
			name:    "empty target",
			input:   "blocks:",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:       "target with colons",
			input:      "blocks:id:with:colons",
			wantType:   "blocks",
			wantTarget: "id:with:colons",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			linkType, targetID, err := parseLink(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("parseLink(%q) expected error, got nil", tt.input)
				}
				return
			}

			if err != nil {
				t.Errorf("parseLink(%q) unexpected error: %v", tt.input, err)
				return
			}

			if linkType != tt.wantType {
				t.Errorf("parseLink(%q) type = %q, want %q", tt.input, linkType, tt.wantType)
			}

			if targetID != tt.wantTarget {
				t.Errorf("parseLink(%q) target = %q, want %q", tt.input, targetID, tt.wantTarget)
			}
		})
	}
}

func TestIsKnownLinkType(t *testing.T) {
	tests := []struct {
		linkType string
		want     bool
	}{
		{"blocks", true},
		{"duplicates", true},
		{"parent", true},
		{"related", true},
		{"unknown", false},
		{"BLOCKS", false}, // case sensitive
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.linkType, func(t *testing.T) {
			got := isKnownLinkType(tt.linkType)
			if got != tt.want {
				t.Errorf("isKnownLinkType(%q) = %v, want %v", tt.linkType, got, tt.want)
			}
		})
	}
}
