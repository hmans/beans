package config

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const ConfigFile = "config.yaml"

// DefaultStatuses defines the default status configuration.
var DefaultStatuses = []StatusConfig{
	{Name: "open", Color: "green", Description: "Ready to be worked on"},
	{Name: "in-progress", Color: "yellow", Description: "Currently being worked on"},
	{Name: "done", Color: "gray", Archive: true, Description: "Completed and ready for archival"},
}

// DefaultTypes defines the default type configuration.
var DefaultTypes = []TypeConfig{
	{Name: "epic", Color: "purple", Description: "A thematic container for related work; should have child beans, not be worked on directly"},
	{Name: "milestone", Color: "cyan", Description: "A target release or checkpoint; group work that should ship together"},
	{Name: "feature", Color: "green", Description: "A user-facing capability or enhancement"},
	{Name: "bug", Color: "red", Description: "Something that is broken and needs fixing"},
	{Name: "task", Color: "blue", Description: "A concrete piece of work to complete (eg. a chore, or a sub-task for a feature)"},
}

// StatusConfig defines a single status with its display color.
type StatusConfig struct {
	Name        string `yaml:"name"`
	Color       string `yaml:"color"`
	Archive     bool   `yaml:"archive,omitempty"`
	Description string `yaml:"description,omitempty"`
}

// TypeConfig defines a single bean type with its display color.
type TypeConfig struct {
	Name        string `yaml:"name"`
	Color       string `yaml:"color"`
	Description string `yaml:"description,omitempty"`
}

// Config holds the beans configuration.
type Config struct {
	Beans    BeansConfig    `yaml:"beans"`
	Statuses []StatusConfig `yaml:"statuses"`
	Types    []TypeConfig   `yaml:"types,omitempty"`
}

// BeansConfig defines settings for bean creation.
type BeansConfig struct {
	Prefix        string `yaml:"prefix"`
	IDLength      int    `yaml:"id_length"`
	DefaultStatus string `yaml:"default_status,omitempty"`
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
		Types:    DefaultTypes,
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
	if err := yaml.Unmarshal(data, &cfg); err != nil {
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

	// Apply default types if none defined
	if len(cfg.Types) == 0 {
		cfg.Types = DefaultTypes
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

	data, err := yaml.Marshal(c)
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

// GetType returns the TypeConfig for a given type name, or nil if not found.
func (c *Config) GetType(name string) *TypeConfig {
	for i := range c.Types {
		if c.Types[i].Name == name {
			return &c.Types[i]
		}
	}
	return nil
}

// TypeNames returns a slice of configured type names.
func (c *Config) TypeNames() []string {
	names := make([]string, len(c.Types))
	for i, t := range c.Types {
		names[i] = t.Name
	}
	return names
}

// IsValidType returns true if the type is in the config's allowed list.
func (c *Config) IsValidType(typeName string) bool {
	for _, t := range c.Types {
		if t.Name == typeName {
			return true
		}
	}
	return false
}

// TypeList returns a comma-separated list of valid types.
func (c *Config) TypeList() string {
	names := make([]string, len(c.Types))
	for i, t := range c.Types {
		names[i] = t.Name
	}
	return strings.Join(names, ", ")
}

// BeanColors holds resolved color information for rendering a bean
type BeanColors struct {
	StatusColor string
	TypeColor   string
	IsArchive   bool
}

// GetBeanColors returns the resolved colors for a bean based on its status and type.
func (c *Config) GetBeanColors(status, typeName string) BeanColors {
	colors := BeanColors{
		StatusColor: "gray",
		TypeColor:   "",
		IsArchive:   false,
	}

	if statusCfg := c.GetStatus(status); statusCfg != nil {
		colors.StatusColor = statusCfg.Color
	}
	colors.IsArchive = c.IsArchiveStatus(status)

	if typeCfg := c.GetType(typeName); typeCfg != nil {
		colors.TypeColor = typeCfg.Color
	}

	return colors
}
