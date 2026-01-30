package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hmans/beans/internal/config"
)

func testPromptData() promptData {
	return promptData{
		GraphQLSchema: "type Query { bean(id: ID!): Bean }",
		Types:         config.DefaultTypes,
		Statuses:      config.DefaultStatuses,
		Priorities:    config.DefaultPriorities,
	}
}

func TestRenderPrime_BuiltinTemplate(t *testing.T) {
	cfg := config.Default()
	cfg.SetConfigDir(t.TempDir())

	result, err := renderPrime(cfg, testPromptData())
	if err != nil {
		t.Fatalf("renderPrime() error = %v", err)
	}

	if result.Warning != "" {
		t.Errorf("unexpected warning: %s", result.Warning)
	}
	if !strings.Contains(result.Output, "Beans Usage Guide") {
		t.Error("built-in output should contain 'Beans Usage Guide'")
	}
	// All types should be rendered
	for _, typ := range config.DefaultTypes {
		if !strings.Contains(result.Output, typ.Name) {
			t.Errorf("built-in output should contain type %q", typ.Name)
		}
	}
	// All statuses should be rendered
	for _, status := range config.DefaultStatuses {
		if !strings.Contains(result.Output, status.Name) {
			t.Errorf("built-in output should contain status %q", status.Name)
		}
	}
	// All priorities should be rendered
	for _, priority := range config.DefaultPriorities {
		if !strings.Contains(result.Output, priority.Name) {
			t.Errorf("built-in output should contain priority %q", priority.Name)
		}
	}
}

func TestRenderPrime_CustomTemplate(t *testing.T) {
	tmpDir := t.TempDir()

	customContent := `# Custom Prime
{{range .Types}}- {{.Name}}
{{end}}
Schema: {{.GraphQLSchema}}
Original: {{.OriginalPrime}}`

	if err := os.WriteFile(filepath.Join(tmpDir, "prime.tmpl"), []byte(customContent), 0644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	cfg := &config.Config{
		Templates: config.TemplatesConfig{Prime: "prime.tmpl"},
	}
	cfg.SetConfigDir(tmpDir)

	result, err := renderPrime(cfg, testPromptData())
	if err != nil {
		t.Fatalf("renderPrime() error = %v", err)
	}

	if result.Warning != "" {
		t.Errorf("unexpected warning: %s", result.Warning)
	}
	if !strings.Contains(result.Output, "Custom Prime") {
		t.Error("output should contain custom header")
	}
	// Types should be rendered via the custom template
	for _, typ := range config.DefaultTypes {
		if !strings.Contains(result.Output, typ.Name) {
			t.Errorf("output should contain type %q", typ.Name)
		}
	}
	// GraphQL schema should be accessible
	if !strings.Contains(result.Output, "type Query") {
		t.Error("output should contain GraphQL schema")
	}
	// OriginalPrime should contain the built-in output
	if !strings.Contains(result.Output, "Beans Usage Guide") {
		t.Error("OriginalPrime in output should contain built-in prime content")
	}
}

func TestRenderPrime_CustomTemplateWithOriginalPrime(t *testing.T) {
	tmpDir := t.TempDir()

	customContent := `# Before
{{.OriginalPrime}}
# After`

	if err := os.WriteFile(filepath.Join(tmpDir, "prime.tmpl"), []byte(customContent), 0644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	cfg := &config.Config{
		Templates: config.TemplatesConfig{Prime: "prime.tmpl"},
	}
	cfg.SetConfigDir(tmpDir)

	result, err := renderPrime(cfg, testPromptData())
	if err != nil {
		t.Fatalf("renderPrime() error = %v", err)
	}

	if result.Warning != "" {
		t.Errorf("unexpected warning: %s", result.Warning)
	}

	// Verify ordering: custom header, then original, then custom footer
	beforeIdx := strings.Index(result.Output, "# Before")
	usageIdx := strings.Index(result.Output, "Beans Usage Guide")
	afterIdx := strings.Index(result.Output, "# After")

	if beforeIdx == -1 || usageIdx == -1 || afterIdx == -1 {
		t.Fatalf("missing expected sections in output:\n%s", result.Output)
	}
	if beforeIdx >= usageIdx {
		t.Error("'# Before' should appear before 'Beans Usage Guide'")
	}
	if usageIdx >= afterIdx {
		t.Error("'Beans Usage Guide' should appear before '# After'")
	}
}

func TestRenderPrime_FallbackOnMissingTemplate(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Templates: config.TemplatesConfig{Prime: "nonexistent.tmpl"},
	}
	cfg.SetConfigDir(tmpDir)

	result, err := renderPrime(cfg, testPromptData())
	if err != nil {
		t.Fatalf("renderPrime() error = %v, expected fallback", err)
	}

	// Should have a warning
	if result.Warning == "" {
		t.Error("expected warning about missing template")
	}
	if !strings.Contains(result.Warning, "nonexistent.tmpl") {
		t.Errorf("warning should mention the missing file, got: %s", result.Warning)
	}
	if !strings.Contains(result.Warning, "falling back") {
		t.Errorf("warning should mention fallback, got: %s", result.Warning)
	}

	// Should still output the built-in prime
	if !strings.Contains(result.Output, "Beans Usage Guide") {
		t.Error("fallback output should contain built-in prime content")
	}
}

func TestRenderPrime_FallbackOnBadTemplateSyntax(t *testing.T) {
	tmpDir := t.TempDir()

	badContent := `{{.Unclosed`
	if err := os.WriteFile(filepath.Join(tmpDir, "bad.tmpl"), []byte(badContent), 0644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	cfg := &config.Config{
		Templates: config.TemplatesConfig{Prime: "bad.tmpl"},
	}
	cfg.SetConfigDir(tmpDir)

	result, err := renderPrime(cfg, testPromptData())
	if err != nil {
		t.Fatalf("renderPrime() error = %v, expected fallback", err)
	}

	// Should have a warning about parse error
	if result.Warning == "" {
		t.Error("expected warning about parse error")
	}
	if !strings.Contains(result.Warning, "parsing") {
		t.Errorf("warning should mention parsing, got: %s", result.Warning)
	}
	if !strings.Contains(result.Warning, "falling back") {
		t.Errorf("warning should mention fallback, got: %s", result.Warning)
	}

	// Should still output the built-in prime
	if !strings.Contains(result.Output, "Beans Usage Guide") {
		t.Error("fallback output should contain built-in prime content")
	}
}

func TestRenderPrime_NoCustomTemplateConfigured(t *testing.T) {
	cfg := config.Default()
	cfg.SetConfigDir(t.TempDir())

	// Verify that Templates.Prime is empty
	if cfg.Templates.Prime != "" {
		t.Fatalf("expected empty Templates.Prime in default config, got %q", cfg.Templates.Prime)
	}

	result, err := renderPrime(cfg, testPromptData())
	if err != nil {
		t.Fatalf("renderPrime() error = %v", err)
	}

	if result.Warning != "" {
		t.Errorf("unexpected warning: %s", result.Warning)
	}
	if !strings.Contains(result.Output, "Beans Usage Guide") {
		t.Error("output should contain built-in prime content")
	}
}

func TestRenderPrime_CustomTemplateWithoutOriginalPrime(t *testing.T) {
	tmpDir := t.TempDir()

	// Custom template that completely replaces the built-in (doesn't use OriginalPrime)
	customContent := `# Fully Custom
This project uses beans.
Types: {{range .Types}}{{.Name}} {{end}}`

	if err := os.WriteFile(filepath.Join(tmpDir, "prime.tmpl"), []byte(customContent), 0644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	cfg := &config.Config{
		Templates: config.TemplatesConfig{Prime: "prime.tmpl"},
	}
	cfg.SetConfigDir(tmpDir)

	result, err := renderPrime(cfg, testPromptData())
	if err != nil {
		t.Fatalf("renderPrime() error = %v", err)
	}

	if result.Warning != "" {
		t.Errorf("unexpected warning: %s", result.Warning)
	}
	if !strings.Contains(result.Output, "Fully Custom") {
		t.Error("output should contain custom content")
	}
	// Should NOT contain built-in content since OriginalPrime wasn't used
	if strings.Contains(result.Output, "EXTREMELY_IMPORTANT") {
		t.Error("output should not contain built-in prime content when OriginalPrime is not referenced")
	}
}
