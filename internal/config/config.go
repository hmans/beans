package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

const ConfigFile = "beans.toml"

// DefaultStatuses defines the default status configuration.
var DefaultStatuses = []StatusConfig{
	{Name: "open", Color: "green"},
	{Name: "in-progress", Color: "yellow"},
	{Name: "done", Color: "gray", Archive: true},
}

// StatusConfig defines a single status with its display color.
type StatusConfig struct {
	Name    string `toml:"name"`
	Color   string `toml:"color"`
	Archive bool   `toml:"archive,omitempty"`
}

// Config holds the beans configuration.
type Config struct {
	Beans    BeansConfig    `toml:"beans"`
	Statuses []StatusConfig `toml:"statuses"`
}

// BeansConfig defines settings for bean creation.
type BeansConfig struct {
	Prefix        string `toml:"prefix"`
	IDLength      int    `toml:"id_length"`
	DefaultStatus string `toml:"default_status,omitempty"`
}

// Default returns a Config with default values.
func Default() *Config {
	return &Config{
		Beans: BeansConfig{
			Prefix:        "",
			IDLength:      4,
			DefaultStatus: "open",
		},
		Statuses: DefaultStatuses,
	}
}

// DefaultWithPrefix returns a Config with the given prefix.
func DefaultWithPrefix(prefix string) *Config {
	cfg := Default()
	cfg.Beans.Prefix = prefix
	return cfg
}

// Load reads configuration from the given .beans directory.
// Returns default config if the file doesn't exist.
func Load(root string) (*Config, error) {
	path := filepath.Join(root, ConfigFile)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Default(), nil
		}
		return nil, err
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Apply defaults for missing values
	if cfg.Beans.IDLength == 0 {
		cfg.Beans.IDLength = 4
	}

	// Apply default statuses if none defined
	if len(cfg.Statuses) == 0 {
		cfg.Statuses = DefaultStatuses
	}

	// Apply default status values if not specified
	if cfg.Beans.DefaultStatus == "" {
		cfg.Beans.DefaultStatus = cfg.Statuses[0].Name
	}

	return &cfg, nil
}

// Save writes the configuration to the given .beans directory.
func (c *Config) Save(root string) error {
	path := filepath.Join(root, ConfigFile)

	data, err := toml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// IsValidStatus returns true if the status is in the config's allowed list.
func (c *Config) IsValidStatus(status string) bool {
	for _, s := range c.Statuses {
		if s.Name == status {
			return true
		}
	}
	return false
}

// StatusList returns a comma-separated list of valid statuses.
func (c *Config) StatusList() string {
	names := make([]string, len(c.Statuses))
	for i, s := range c.Statuses {
		names[i] = s.Name
	}
	return strings.Join(names, ", ")
}

// StatusNames returns a slice of valid status names.
func (c *Config) StatusNames() []string {
	names := make([]string, len(c.Statuses))
	for i, s := range c.Statuses {
		names[i] = s.Name
	}
	return names
}

// GetStatus returns the StatusConfig for a given status name, or nil if not found.
func (c *Config) GetStatus(name string) *StatusConfig {
	for i := range c.Statuses {
		if c.Statuses[i].Name == name {
			return &c.Statuses[i]
		}
	}
	return nil
}

// GetDefaultStatus returns the default status name for new beans.
func (c *Config) GetDefaultStatus() string {
	return c.Beans.DefaultStatus
}

// IsArchiveStatus returns true if the given status is marked for archiving.
func (c *Config) IsArchiveStatus(name string) bool {
	if s := c.GetStatus(name); s != nil {
		return s.Archive
	}
	return false
}
