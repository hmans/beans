package bean

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name                string
		input               string
		expectedTitle       string
		expectedStatus      string
		expectedDescription string
		wantErr             bool
	}{
		{
			name: "basic bean",
			input: `---
title: Test Bean
status: open
---

This is the description.`,
			expectedTitle:       "Test Bean",
			expectedStatus:      "open",
			expectedDescription: "\nThis is the description.",
		},
		{
			name: "with timestamps",
			input: `---
title: With Times
status: in-progress
created_at: 2024-01-15T10:30:00Z
updated_at: 2024-01-16T14:45:00Z
---

Description content here.`,
			expectedTitle:       "With Times",
			expectedStatus:      "in-progress",
			expectedDescription: "\nDescription content here.",
		},
		{
			name: "empty description",
			input: `---
title: No Description
status: done
---`,
			expectedTitle:       "No Description",
			expectedStatus:      "done",
			expectedDescription: "",
		},
		{
			name: "multiline description",
			input: `---
title: Multi Line
status: open
---

# Header

- Item 1
- Item 2

Paragraph text.`,
			expectedTitle:       "Multi Line",
			expectedStatus:      "open",
			expectedDescription: "\n# Header\n\n- Item 1\n- Item 2\n\nParagraph text.",
		},
		{
			name: "plain text without frontmatter",
			input:               `Just plain text without any YAML frontmatter.`,
			expectedTitle:       "",
			expectedStatus:      "",
			expectedDescription: "Just plain text without any YAML frontmatter.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bean, err := Parse(strings.NewReader(tt.input))
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if bean.Title != tt.expectedTitle {
				t.Errorf("Title = %q, want %q", bean.Title, tt.expectedTitle)
			}
			if bean.Status != tt.expectedStatus {
				t.Errorf("Status = %q, want %q", bean.Status, tt.expectedStatus)
			}
			if bean.Description != tt.expectedDescription {
				t.Errorf("Description = %q, want %q", bean.Description, tt.expectedDescription)
			}
		})
	}
}

func TestParseWithType(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedType string
	}{
		{
			name: "with type field",
			input: `---
title: Bug Report
status: open
type: bug
---

Description of the bug.`,
			expectedType: "bug",
		},
		{
			name: "without type field",
			input: `---
title: No Type
status: open
---

No type specified.`,
			expectedType: "",
		},
		{
			// Backwards compatibility: beans with types not in current config
			// should still be readable without error
			name: "with unknown type (backwards compatibility)",
			input: `---
title: Legacy Bean
status: open
type: deprecated-type-no-longer-in-config
---`,
			expectedType: "deprecated-type-no-longer-in-config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bean, err := Parse(strings.NewReader(tt.input))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if bean.Type != tt.expectedType {
				t.Errorf("Type = %q, want %q", bean.Type, tt.expectedType)
			}
		})
	}
}

func TestRender(t *testing.T) {
	now := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name     string
		bean     *Bean
		contains []string
	}{
		{
			name: "basic bean",
			bean: &Bean{
				Title:  "Test Bean",
				Status: "open",
			},
			contains: []string{
				"---",
				"title: Test Bean",
				"status: open",
			},
		},
		{
			name: "with description",
			bean: &Bean{
				Title:       "With Description",
				Status:      "done",
				Description: "This is content.",
			},
			contains: []string{
				"title: With Description",
				"status: done",
				"This is content.",
			},
		},
		{
			name: "with timestamps",
			bean: &Bean{
				Title:     "Timed",
				Status:    "open",
				CreatedAt: &now,
				UpdatedAt: &now,
			},
			contains: []string{
				"title: Timed",
				"created_at:",
				"updated_at:",
			},
		},
		{
			name: "with type",
			bean: &Bean{
				Title:  "Typed Bean",
				Status: "open",
				Type:   "bug",
			},
			contains: []string{
				"title: Typed Bean",
				"status: open",
				"type: bug",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := tt.bean.Render()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			result := string(output)
			for _, want := range tt.contains {
				if !strings.Contains(result, want) {
					t.Errorf("output missing %q\ngot:\n%s", want, result)
				}
			}
		})
	}
}

func TestParseRenderRoundtrip(t *testing.T) {
	now := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	later := time.Date(2024, 1, 16, 14, 45, 0, 0, time.UTC)

	tests := []struct {
		name string
		bean *Bean
	}{
		{
			name: "basic",
			bean: &Bean{
				Title:  "Basic Bean",
				Status: "open",
			},
		},
		{
			name: "with description",
			bean: &Bean{
				Title:       "Bean With Description",
				Status:      "in-progress",
				Description: "This is the description content.\n\nWith multiple paragraphs.",
			},
		},
		{
			name: "with timestamps",
			bean: &Bean{
				Title:       "Timestamped Bean",
				Status:      "done",
				CreatedAt:   &now,
				UpdatedAt:   &later,
				Description: "Some content.",
			},
		},
		{
			name: "with type",
			bean: &Bean{
				Title:       "Typed Bean",
				Status:      "open",
				Type:        "bug",
				Description: "Bug description.",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Render to bytes
			rendered, err := tt.bean.Render()
			if err != nil {
				t.Fatalf("Render error: %v", err)
			}

			// Parse back
			parsed, err := Parse(strings.NewReader(string(rendered)))
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			// Compare fields
			if parsed.Title != tt.bean.Title {
				t.Errorf("Title roundtrip: got %q, want %q", parsed.Title, tt.bean.Title)
			}
			if parsed.Status != tt.bean.Status {
				t.Errorf("Status roundtrip: got %q, want %q", parsed.Status, tt.bean.Status)
			}
			if parsed.Type != tt.bean.Type {
				t.Errorf("Type roundtrip: got %q, want %q", parsed.Type, tt.bean.Type)
			}

			// Description comparison (parse adds newline prefix for non-empty description)
			wantDescription := tt.bean.Description
			if wantDescription != "" {
				wantDescription = "\n" + wantDescription
			}
			if parsed.Description != wantDescription {
				t.Errorf("Description roundtrip: got %q, want %q", parsed.Description, wantDescription)
			}

			// Timestamp comparison
			if tt.bean.CreatedAt != nil {
				if parsed.CreatedAt == nil {
					t.Error("CreatedAt: got nil, want non-nil")
				} else if !parsed.CreatedAt.Equal(*tt.bean.CreatedAt) {
					t.Errorf("CreatedAt: got %v, want %v", parsed.CreatedAt, tt.bean.CreatedAt)
				}
			}
			if tt.bean.UpdatedAt != nil {
				if parsed.UpdatedAt == nil {
					t.Error("UpdatedAt: got nil, want non-nil")
				} else if !parsed.UpdatedAt.Equal(*tt.bean.UpdatedAt) {
					t.Errorf("UpdatedAt: got %v, want %v", parsed.UpdatedAt, tt.bean.UpdatedAt)
				}
			}
		})
	}
}

func TestBeanJSONSerialization(t *testing.T) {
	t.Run("description omitted when empty", func(t *testing.T) {
		bean := &Bean{
			ID:          "test-123",
			Title:       "Test Bean",
			Status:      "open",
			Description: "",
		}

		data, err := json.Marshal(bean)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		jsonStr := string(data)
		if strings.Contains(jsonStr, `"description"`) {
			t.Errorf("JSON should not contain 'description' field when empty, got: %s", jsonStr)
		}
	})

	t.Run("description included when non-empty", func(t *testing.T) {
		bean := &Bean{
			ID:          "test-123",
			Title:       "Test Bean",
			Status:      "open",
			Description: "This is the description content.",
		}

		data, err := json.Marshal(bean)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		jsonStr := string(data)
		if !strings.Contains(jsonStr, `"description":"This is the description content."`) {
			t.Errorf("JSON should contain 'description' field with content, got: %s", jsonStr)
		}
	})
}
