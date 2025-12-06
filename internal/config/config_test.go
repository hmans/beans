package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	if cfg.Beans.IDLength != 4 {
		t.Errorf("IDLength = %d, want 4", cfg.Beans.IDLength)
	}
	if cfg.Beans.Prefix != "" {
		t.Errorf("Prefix = %q, want empty", cfg.Beans.Prefix)
	}
	if cfg.Beans.DefaultStatus != "open" {
		t.Errorf("DefaultStatus = %q, want \"open\"", cfg.Beans.DefaultStatus)
	}
	if len(cfg.Statuses) != 3 {
		t.Errorf("len(Statuses) = %d, want 3", len(cfg.Statuses))
	}
	if len(cfg.Types) != 4 {
		t.Errorf("len(Types) = %d, want 4", len(cfg.Types))
	}
}

func TestDefaultWithPrefix(t *testing.T) {
	cfg := DefaultWithPrefix("myapp-")

	if cfg.Beans.Prefix != "myapp-" {
		t.Errorf("Prefix = %q, want \"myapp-\"", cfg.Beans.Prefix)
	}
	// Other defaults should still apply
	if cfg.Beans.IDLength != 4 {
		t.Errorf("IDLength = %d, want 4", cfg.Beans.IDLength)
	}
}

func TestIsValidStatus(t *testing.T) {
	cfg := Default()

	tests := []struct {
		status string
		want   bool
	}{
		{"open", true},
		{"in-progress", true},
		{"done", true},
		{"invalid", false},
		{"", false},
		{"OPEN", false}, // case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			got := cfg.IsValidStatus(tt.status)
			if got != tt.want {
				t.Errorf("IsValidStatus(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestStatusList(t *testing.T) {
	cfg := Default()
	got := cfg.StatusList()
	want := "open, in-progress, done"

	if got != want {
		t.Errorf("StatusList() = %q, want %q", got, want)
	}
}

func TestStatusNames(t *testing.T) {
	cfg := Default()
	got := cfg.StatusNames()

	if len(got) != 3 {
		t.Fatalf("len(StatusNames()) = %d, want 3", len(got))
	}
	if got[0] != "open" || got[1] != "in-progress" || got[2] != "done" {
		t.Errorf("StatusNames() = %v, want [open, in-progress, done]", got)
	}
}

func TestGetStatus(t *testing.T) {
	cfg := Default()

	t.Run("existing status", func(t *testing.T) {
		s := cfg.GetStatus("open")
		if s == nil {
			t.Fatal("GetStatus(\"open\") = nil, want non-nil")
		}
		if s.Name != "open" {
			t.Errorf("Name = %q, want \"open\"", s.Name)
		}
		if s.Color != "green" {
			t.Errorf("Color = %q, want \"green\"", s.Color)
		}
	})

	t.Run("non-existing status", func(t *testing.T) {
		s := cfg.GetStatus("invalid")
		if s != nil {
			t.Errorf("GetStatus(\"invalid\") = %v, want nil", s)
		}
	})
}

func TestGetDefaultStatus(t *testing.T) {
	cfg := Default()
	got := cfg.GetDefaultStatus()

	if got != "open" {
		t.Errorf("GetDefaultStatus() = %q, want \"open\"", got)
	}
}

func TestIsArchiveStatus(t *testing.T) {
	cfg := Default()

	tests := []struct {
		status string
		want   bool
	}{
		{"done", true},
		{"open", false},
		{"in-progress", false},
		{"invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			got := cfg.IsArchiveStatus(tt.status)
			if got != tt.want {
				t.Errorf("IsArchiveStatus(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestLoadNonExistent(t *testing.T) {
	// Load from non-existent directory should return defaults
	cfg, err := Load("/nonexistent/path/that/does/not/exist")
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	// Should have default values
	if cfg.Beans.IDLength != 4 {
		t.Errorf("IDLength = %d, want 4", cfg.Beans.IDLength)
	}
}

func TestLoadAndSave(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Create a config
	cfg := &Config{
		Beans: BeansConfig{
			Prefix:        "test-",
			IDLength:      6,
			DefaultStatus: "todo",
		},
		Statuses: []StatusConfig{
			{Name: "todo", Color: "blue"},
			{Name: "doing", Color: "yellow"},
			{Name: "finished", Color: "green", Archive: true},
		},
	}

	// Save it
	if err := cfg.Save(tmpDir); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify file exists
	configPath := filepath.Join(tmpDir, ConfigFile)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("config file was not created")
	}

	// Load it back
	loaded, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify values
	if loaded.Beans.Prefix != "test-" {
		t.Errorf("Prefix = %q, want \"test-\"", loaded.Beans.Prefix)
	}
	if loaded.Beans.IDLength != 6 {
		t.Errorf("IDLength = %d, want 6", loaded.Beans.IDLength)
	}
	if loaded.Beans.DefaultStatus != "todo" {
		t.Errorf("DefaultStatus = %q, want \"todo\"", loaded.Beans.DefaultStatus)
	}
	if len(loaded.Statuses) != 3 {
		t.Errorf("len(Statuses) = %d, want 3", len(loaded.Statuses))
	}
}

func TestLoadAppliesDefaults(t *testing.T) {
	// Create temp directory with minimal config
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ConfigFile)

	// Write minimal config (missing id_length, default_status, statuses)
	minimalConfig := `[beans]
prefix = "my-"
`
	if err := os.WriteFile(configPath, []byte(minimalConfig), 0644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	// Load it
	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify defaults were applied
	if cfg.Beans.IDLength != 4 {
		t.Errorf("IDLength default not applied: got %d, want 4", cfg.Beans.IDLength)
	}
	if len(cfg.Statuses) != 3 {
		t.Errorf("Default statuses not applied: got %d, want 3", len(cfg.Statuses))
	}
	// DefaultStatus should be first status name when not specified
	if cfg.Beans.DefaultStatus != "open" {
		t.Errorf("DefaultStatus default not applied: got %q, want \"open\"", cfg.Beans.DefaultStatus)
	}
}

func TestCustomStatuses(t *testing.T) {
	cfg := &Config{
		Beans: BeansConfig{
			Prefix:        "",
			IDLength:      4,
			DefaultStatus: "backlog",
		},
		Statuses: []StatusConfig{
			{Name: "backlog", Color: "gray"},
			{Name: "active", Color: "#FF6B6B"},
			{Name: "review", Color: "purple"},
			{Name: "shipped", Color: "green", Archive: true},
		},
	}

	// Test custom status validation
	if !cfg.IsValidStatus("backlog") {
		t.Error("IsValidStatus(\"backlog\") = false, want true")
	}
	if !cfg.IsValidStatus("active") {
		t.Error("IsValidStatus(\"active\") = false, want true")
	}
	if cfg.IsValidStatus("open") {
		t.Error("IsValidStatus(\"open\") = true, want false (not in custom statuses)")
	}

	// Test custom archive status
	if !cfg.IsArchiveStatus("shipped") {
		t.Error("IsArchiveStatus(\"shipped\") = false, want true")
	}
	if cfg.IsArchiveStatus("active") {
		t.Error("IsArchiveStatus(\"active\") = true, want false")
	}

	// Test hex color in custom status
	s := cfg.GetStatus("active")
	if s == nil || s.Color != "#FF6B6B" {
		t.Error("Custom hex color not preserved")
	}
}

func TestIsValidType(t *testing.T) {
	cfg := Default()

	tests := []struct {
		typeName string
		want     bool
	}{
		{"task", true},
		{"feature", true},
		{"bug", true},
		{"epic", true},
		{"invalid", false},
		{"", false},
		{"TASK", false}, // case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.typeName, func(t *testing.T) {
			got := cfg.IsValidType(tt.typeName)
			if got != tt.want {
				t.Errorf("IsValidType(%q) = %v, want %v", tt.typeName, got, tt.want)
			}
		})
	}
}

func TestTypeList(t *testing.T) {
	cfg := Default()
	got := cfg.TypeList()
	want := "task, feature, bug, epic"

	if got != want {
		t.Errorf("TypeList() = %q, want %q", got, want)
	}
}

func TestGetType(t *testing.T) {
	cfg := &Config{
		Types: []TypeConfig{
			{Name: "bug", Color: "red"},
			{Name: "feature", Color: "green"},
			{Name: "epic", Color: "purple"},
		},
	}

	t.Run("existing type", func(t *testing.T) {
		typ := cfg.GetType("bug")
		if typ == nil {
			t.Fatal("GetType(\"bug\") = nil, want non-nil")
		}
		if typ.Name != "bug" {
			t.Errorf("Name = %q, want \"bug\"", typ.Name)
		}
		if typ.Color != "red" {
			t.Errorf("Color = %q, want \"red\"", typ.Color)
		}
	})

	t.Run("non-existing type", func(t *testing.T) {
		// Backwards compatibility: GetType returns nil for unknown types
		// but this should not cause errors - callers handle nil gracefully
		typ := cfg.GetType("deprecated-type-no-longer-in-config")
		if typ != nil {
			t.Errorf("GetType(\"deprecated-type-no-longer-in-config\") = %v, want nil", typ)
		}
	})

	t.Run("default types config", func(t *testing.T) {
		defaultCfg := Default()
		typ := defaultCfg.GetType("bug")
		if typ == nil {
			t.Fatal("GetType(\"bug\") on default config = nil, want non-nil")
		}
		if typ.Color != "red" {
			t.Errorf("bug color = %q, want \"red\"", typ.Color)
		}
	})
}

func TestTypesConfig(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Create a config with types
	cfg := &Config{
		Beans: BeansConfig{
			Prefix:        "test-",
			IDLength:      4,
			DefaultStatus: "open",
		},
		Statuses: DefaultStatuses,
		Types: []TypeConfig{
			{Name: "bug", Color: "red"},
			{Name: "feature", Color: "blue"},
		},
	}

	// Save it
	if err := cfg.Save(tmpDir); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Load it back
	loaded, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify types were preserved
	if len(loaded.Types) != 2 {
		t.Errorf("len(Types) = %d, want 2", len(loaded.Types))
	}
	if loaded.Types[0].Name != "bug" || loaded.Types[0].Color != "red" {
		t.Errorf("Types[0] = %+v, want {Name:bug Color:red}", loaded.Types[0])
	}
}
