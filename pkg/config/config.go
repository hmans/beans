package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	// ConfigFileName is the name of the config file at project root
	ConfigFileName = ".beans.yml"
	// DefaultBeansPath is the default directory for storing beans
	DefaultBeansPath = ".beans"
	// LegacyConfigFile is the old config file location (deprecated)
	LegacyConfigFile = "config.yaml"
	// DefaultServerPort is the default port for the web server
	DefaultServerPort = 8080
)

// DefaultStatuses defines the hardcoded status configuration.
// Statuses are not configurable - they are hardcoded like types.
// Order determines sort priority: in-progress first (active work), then todo, draft, and done states last.
var DefaultStatuses = []StatusConfig{
	{Name: "in-progress", Color: "yellow", Description: "Currently being worked on"},
	{Name: "todo", Color: "green", Description: "Ready to be worked on"},
	{Name: "draft", Color: "blue", Description: "Needs refinement before it can be worked on"},
	{Name: "completed", Color: "gray", Archive: true, Description: "Finished successfully"},
	{Name: "scrapped", Color: "gray", Archive: true, Description: "Will not be done"},
}

// DefaultTypes defines the default type configuration.
var DefaultTypes = []TypeConfig{
	{Name: "milestone", Color: "cyan", Description: "A target release or checkpoint; group work that should ship together"},
	{Name: "epic", Color: "purple", Description: "A thematic container for related work; should have child beans, not be worked on directly"},
	{Name: "bug", Color: "red", Description: "Something that is broken and needs fixing"},
	{Name: "feature", Color: "green", Description: "A user-facing capability or enhancement"},
	{Name: "task", Color: "blue", Description: "A concrete piece of work to complete (eg. a chore, or a sub-task for a feature)"},
}

// DefaultPriorities defines the hardcoded priority configuration.
// Priorities are ordered from highest to lowest urgency.
var DefaultPriorities = []PriorityConfig{
	{Name: "critical", Color: "red", Description: "Urgent, blocking work. When possible, address immediately"},
	{Name: "high", Color: "yellow", Description: "Important, should be done before normal work"},
	{Name: "normal", Color: "white", Description: "Standard priority"},
	{Name: "low", Color: "gray", Description: "Less important, can be delayed"},
	{Name: "deferred", Color: "gray", Description: "Explicitly pushed back, avoid doing unless necessary"},
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

// PriorityConfig defines a single priority level with its display color.
type PriorityConfig struct {
	Name        string `yaml:"name"`
	Color       string `yaml:"color"`
	Description string `yaml:"description,omitempty"`
}

// PermissionMode represents the default agent permission mode.
type PermissionMode string

const (
	PermissionModeAct  PermissionMode = "act"
	PermissionModePlan PermissionMode = "plan"
)

// IntegrateMode represents the worktree integration strategy.
type IntegrateMode string

const (
	IntegrateModeLocal IntegrateMode = "local"
	IntegrateModePR    IntegrateMode = "pr"
)

// WorktreeConfig defines settings for git worktree management.
type WorktreeConfig struct {
	// BaseRef is the git ref to use as the starting point for new worktree branches.
	// Default: "main"
	BaseRef string `yaml:"base_ref,omitempty"`

	// Path is the directory where worktrees are created.
	// Default: ~/.beans/worktrees/<project-name>/
	// Supports ~ for home directory.
	Path string `yaml:"path,omitempty"`

	// Setup is a shell command to run inside a worktree after creation (e.g. "pnpm install").
	Setup string `yaml:"setup,omitempty"`

	// Run is a shell command to run the project (e.g. "mise dev").
	// When set, a "Run" button appears in the workspace toolbar.
	Run string `yaml:"run,omitempty"`

	// Integrate controls the worktree integration strategy.
	// "local" (default): squash-merge locally, hides PR buttons.
	// "pr": push and create PRs, hides the local Integrate button.
	Integrate IntegrateMode `yaml:"integrate,omitempty"`

	// FetchTimeout is the timeout in seconds for the git fetch that runs before
	// creating a new worktree. This fetch updates the base ref from the remote.
	// Set to 0 to disable the fetch entirely (useful for airgapped environments).
	// Default: 10 (seconds).
	FetchTimeout *int `yaml:"fetch_timeout,omitempty"`
}

// AgentConfig defines settings for agent sessions.
type AgentConfig struct {
	// Enabled controls whether agent functionality is available.
	// When false, the web UI hides agent chats, status panes, and worktree features.
	// Default: true
	Enabled *bool `yaml:"enabled,omitempty"`

	// DefaultMode is the default mode for new agent sessions.
	// Valid values: "act" (fully autonomous), "plan" (read-only).
	// Default: "act"
	DefaultMode PermissionMode `yaml:"default_mode,omitempty"`

	// DefaultEffort is the default thinking effort level for new agent sessions.
	// Valid values: "low", "medium", "high", "max".
	// When omitted, new sessions start with no effort override (uses CLI default).
	DefaultEffort string `yaml:"default_effort,omitempty"`
}

// ProjectConfig defines project-level settings.
type ProjectConfig struct {
	// Name is the human-readable project name, displayed in the UI.
	// Default: derived from directory name during `beans init`.
	Name string `yaml:"name,omitempty"`
}

// ServerConfig defines settings for the web server.
type ServerConfig struct {
	// Port is the port to listen on (default: 8080)
	Port int `yaml:"port,omitempty"`
	// CORSOrigins is the list of allowed origins for CORS and WebSocket.
	// Supports exact origins and port wildcards (e.g. "http://localhost:*").
	// Use "*" to allow all origins (not recommended for production).
	// Default: ["http://localhost:*", "http://127.0.0.1:*"]
	CORSOrigins []string `yaml:"cors_origins,omitempty"`
}

// Config holds the beans configuration.
// Note: Statuses are no longer stored in config - they are hardcoded like types.
type Config struct {
	Project  ProjectConfig  `yaml:"project,omitempty"`
	Beans    BeansConfig    `yaml:"beans"`
	Worktree WorktreeConfig `yaml:"worktree,omitempty"`
	Agent    AgentConfig    `yaml:"agent,omitempty"`
	Server   ServerConfig   `yaml:"server,omitempty"`

	// configDir is the directory containing the config file (not serialized)
	// Used to resolve relative paths
	configDir string `yaml:"-"`
}

// BeansConfig defines settings for bean creation.
type BeansConfig struct {
	// Path is the path to the beans directory (relative to config file location)
	Path           string `yaml:"path,omitempty"`
	Prefix         string `yaml:"prefix"`
	IDLength       int    `yaml:"id_length"`
	DefaultStatus  string `yaml:"default_status,omitempty"`
	DefaultType    string `yaml:"default_type,omitempty"`
	RequireIfMatch bool   `yaml:"require_if_match,omitempty"`
}

// Default returns a Config with default values.
func Default() *Config {
	return &Config{
		Beans: BeansConfig{
			Path:          DefaultBeansPath,
			Prefix:        "",
			IDLength:      4,
			DefaultStatus: "todo",
			DefaultType:   "task",
		},
		Worktree: WorktreeConfig{
			BaseRef:   DefaultWorktreeBaseRef,
			Integrate: IntegrateModeLocal,
		},
		Agent: AgentConfig{
			Enabled:     boolPtr(true),
			DefaultMode: PermissionModeAct,
		},
		Server: ServerConfig{
			Port: DefaultServerPort,
		},
	}
}

// DefaultWithPrefix returns a Config with the given prefix.
func DefaultWithPrefix(prefix string) *Config {
	cfg := Default()
	cfg.Beans.Prefix = prefix
	return cfg
}

// FindConfig searches upward from the given directory for a .beans.yml config file.
// Returns the absolute path to the config file, or empty string if not found.
func FindConfig(startDir string) (string, error) {
	return FindConfigWithin(startDir, "")
}

// FindConfigWithin searches upward from startDir for a .beans.yml config file,
// stopping at rootDir (inclusive). If rootDir is empty, searches up to the
// filesystem root.
func FindConfigWithin(startDir, rootDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}

	var absRoot string
	if rootDir != "" {
		absRoot, err = filepath.Abs(rootDir)
		if err != nil {
			return "", err
		}
	}

	for {
		configPath := filepath.Join(dir, ConfigFileName)
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}

		// Stop if we've reached the root boundary
		if absRoot != "" && dir == absRoot {
			return "", nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			return "", nil
		}
		dir = parent
	}
}

// Load reads configuration from the given config file path.
// Returns default config if the file doesn't exist.
func Load(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
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

	// Store the config directory for resolving relative paths
	cfg.configDir = filepath.Dir(configPath)

	// Apply defaults for missing values
	if cfg.Beans.Path == "" {
		cfg.Beans.Path = DefaultBeansPath
	}
	if cfg.Beans.IDLength == 0 {
		cfg.Beans.IDLength = 4
	}
	if cfg.Beans.DefaultStatus == "" {
		cfg.Beans.DefaultStatus = "todo"
	}
	if cfg.Beans.DefaultType == "" {
		cfg.Beans.DefaultType = DefaultTypes[0].Name
	}

	return &cfg, nil
}

// LoadFromDirectory finds and loads the config file by searching upward from the given directory.
// If no config file is found, returns a default config anchored at the given directory.
func LoadFromDirectory(startDir string) (*Config, error) {
	return LoadFromDirectoryWithin(startDir, "")
}

// LoadFromDirectoryWithin is like LoadFromDirectory but limits the search to
// within rootDir (see FindConfigWithin).
func LoadFromDirectoryWithin(startDir, rootDir string) (*Config, error) {
	configPath, err := FindConfigWithin(startDir, rootDir)
	if err != nil {
		return nil, err
	}

	if configPath == "" {
		// No config found, return default anchored at startDir
		cfg := Default()
		cfg.configDir = startDir
		return cfg, nil
	}

	return Load(configPath)
}

// ResolveBeansPath returns the absolute path to the beans directory.
func (c *Config) ResolveBeansPath() string {
	if filepath.IsAbs(c.Beans.Path) {
		return c.Beans.Path
	}
	if c.configDir == "" {
		// Fallback: use current directory
		cwd, _ := os.Getwd()
		return filepath.Join(cwd, c.Beans.Path)
	}
	return filepath.Join(c.configDir, c.Beans.Path)
}

// ConfigDir returns the directory containing the config file.
func (c *Config) ConfigDir() string {
	return c.configDir
}

// SetConfigDir sets the config directory (for testing or when creating new configs).
func (c *Config) SetConfigDir(dir string) {
	c.configDir = dir
}

// Save writes the configuration to the config file with helpful comments.
// If configDir is set, saves to that directory; otherwise saves to the given directory.
func (c *Config) Save(dir string) error {
	targetDir := c.configDir
	if targetDir == "" {
		targetDir = dir
	}
	path := filepath.Join(targetDir, ConfigFileName)

	doc := c.toYAMLNode()

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(4)
	if err := enc.Encode(doc); err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}
	if err := enc.Close(); err != nil {
		return fmt.Errorf("closing encoder: %w", err)
	}

	return os.WriteFile(path, buf.Bytes(), 0644)
}

// toYAMLNode builds a yaml.Node document tree with inline comments.
func (c *Config) toYAMLNode() *yaml.Node {
	// Helper to create a scalar node
	scalar := func(value string, tag string) *yaml.Node {
		return &yaml.Node{Kind: yaml.ScalarNode, Value: value, Tag: tag}
	}
	strNode := func(value string) *yaml.Node {
		return scalar(value, "!!str")
	}
	intNode := func(value int) *yaml.Node {
		return scalar(fmt.Sprintf("%d", value), "!!int")
	}

	// Build the project mapping
	projectMapping := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	if c.Project.Name != "" {
		key := strNode("name")
		key.HeadComment = "Human-readable project name (displayed in the UI)"
		projectMapping.Content = append(projectMapping.Content, key, strNode(c.Project.Name))
	}

	// Build the beans mapping
	beansMapping := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}

	if c.Beans.Path != "" {
		key := strNode("path")
		key.HeadComment = "Directory where bean files are stored"
		beansMapping.Content = append(beansMapping.Content, key, strNode(c.Beans.Path))
	}

	prefixKey := strNode("prefix")
	prefixKey.HeadComment = "Prefix for bean IDs (e.g., \"myproject-abc1\")"
	beansMapping.Content = append(beansMapping.Content, prefixKey, strNode(c.Beans.Prefix))

	idLenKey := strNode("id_length")
	idLenKey.HeadComment = "Length of the random ID suffix"
	beansMapping.Content = append(beansMapping.Content, idLenKey, intNode(c.Beans.IDLength))

	if c.Beans.DefaultStatus != "" {
		key := strNode("default_status")
		key.HeadComment = "Default status for new beans"
		beansMapping.Content = append(beansMapping.Content, key, strNode(c.Beans.DefaultStatus))
	}

	if c.Beans.DefaultType != "" {
		key := strNode("default_type")
		key.HeadComment = "Default type for new beans"
		beansMapping.Content = append(beansMapping.Content, key, strNode(c.Beans.DefaultType))
	}

	if c.Beans.RequireIfMatch {
		key := strNode("require_if_match")
		key.HeadComment = "Require ETag for updates (optimistic concurrency)"
		beansMapping.Content = append(beansMapping.Content, key, scalar("true", "!!bool"))
	}

	// Build the worktree mapping
	worktreeMapping := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	if c.Worktree.BaseRef != "" {
		key := strNode("base_ref")
		key.HeadComment = "Git ref to use as the base for new worktree branches (default: main)"
		worktreeMapping.Content = append(worktreeMapping.Content, key, strNode(c.Worktree.BaseRef))
	}
	if c.Worktree.Path != "" {
		key := strNode("path")
		key.HeadComment = "Directory for worktrees (default: ~/.beans/worktrees/<project>/)"
		worktreeMapping.Content = append(worktreeMapping.Content, key, strNode(c.Worktree.Path))
	}
	setupKey := strNode("setup")
	setupKey.HeadComment = "Shell command to run inside a worktree after creation (e.g. \"pnpm install\")"
	worktreeMapping.Content = append(worktreeMapping.Content, setupKey, strNode(c.Worktree.Setup))

	runKey := strNode("run")
	runKey.HeadComment = "Shell command to run the project (adds a \"Run\" button to workspace toolbar)"
	worktreeMapping.Content = append(worktreeMapping.Content, runKey, strNode(c.Worktree.Run))

	integrateKey := strNode("integrate")
	integrateKey.HeadComment = "Integration strategy: \"local\" (squash-merge locally) or \"pr\" (push and create PRs)"
	worktreeMapping.Content = append(worktreeMapping.Content, integrateKey, strNode(string(c.GetWorktreeIntegrate())))

	// Build the agent mapping
	agentMapping := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	if c.Agent.Enabled != nil {
		key := strNode("enabled")
		key.HeadComment = "Enable agent functionality in the web UI (true, false)"
		agentMapping.Content = append(agentMapping.Content, key, scalar(fmt.Sprintf("%t", *c.Agent.Enabled), "!!bool"))
	}
	if c.Agent.DefaultMode != "" {
		key := strNode("default_mode")
		key.HeadComment = "Default mode for agent sessions (act, plan)"
		agentMapping.Content = append(agentMapping.Content, key, strNode(string(c.Agent.DefaultMode)))
	}
	// Build the server mapping
	serverMapping := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	if c.Server.Port != 0 {
		portKey := strNode("port")
		portKey.HeadComment = "Port for the web UI (used by `beans-serve`)"
		serverMapping.Content = append(serverMapping.Content, portKey, intNode(c.Server.Port))
	}

	// Build the top-level mapping
	topMapping := &yaml.Node{
		Kind:        yaml.MappingNode,
		Tag:         "!!map",
		HeadComment: "Beans configuration\nSee: https://github.com/hmans/beans",
	}
	if len(projectMapping.Content) > 0 {
		topMapping.Content = append(topMapping.Content, strNode("project"), projectMapping)
	}

	topMapping.Content = append(topMapping.Content, strNode("beans"), beansMapping)

	if len(worktreeMapping.Content) > 0 {
		topMapping.Content = append(topMapping.Content, strNode("worktree"), worktreeMapping)
	}

	if len(agentMapping.Content) > 0 {
		topMapping.Content = append(topMapping.Content, strNode("agent"), agentMapping)
	}

	if len(serverMapping.Content) > 0 {
		topMapping.Content = append(topMapping.Content, strNode("server"), serverMapping)
	}

	// Wrap in a document node
	return &yaml.Node{
		Kind:    yaml.DocumentNode,
		Content: []*yaml.Node{topMapping},
	}
}

// IsValidStatus returns true if the status is a valid hardcoded status.
func (c *Config) IsValidStatus(status string) bool {
	for _, s := range DefaultStatuses {
		if s.Name == status {
			return true
		}
	}
	return false
}

// StatusList returns a comma-separated list of valid statuses.
// Statuses are hardcoded and not configurable.
func (c *Config) StatusList() string {
	names := make([]string, len(DefaultStatuses))
	for i, s := range DefaultStatuses {
		names[i] = s.Name
	}
	return strings.Join(names, ", ")
}

// StatusNames returns a slice of valid status names.
// Statuses are hardcoded and not configurable.
func (c *Config) StatusNames() []string {
	names := make([]string, len(DefaultStatuses))
	for i, s := range DefaultStatuses {
		names[i] = s.Name
	}
	return names
}

// GetStatus returns the StatusConfig for a given status name, or nil if not found.
// Statuses are hardcoded and not configurable.
func (c *Config) GetStatus(name string) *StatusConfig {
	for i := range DefaultStatuses {
		if DefaultStatuses[i].Name == name {
			return &DefaultStatuses[i]
		}
	}
	return nil
}

// GetDefaultStatus returns the default status name for new beans.
func (c *Config) GetDefaultStatus() string {
	if c.Beans.DefaultStatus == "" {
		return "todo"
	}
	return c.Beans.DefaultStatus
}

// GetDefaultType returns the default type name for new beans.
func (c *Config) GetDefaultType() string {
	return c.Beans.DefaultType
}

// IsArchiveStatus returns true if the given status is marked for archiving.
// Statuses are hardcoded and not configurable.
func (c *Config) IsArchiveStatus(name string) bool {
	if s := c.GetStatus(name); s != nil {
		return s.Archive
	}
	return false
}

// GetType returns the TypeConfig for a given type name, or nil if not found.
// Types are hardcoded and not configurable.
func (c *Config) GetType(name string) *TypeConfig {
	for i := range DefaultTypes {
		if DefaultTypes[i].Name == name {
			return &DefaultTypes[i]
		}
	}
	return nil
}

// TypeNames returns a slice of valid type names.
// Types are hardcoded and not configurable.
func (c *Config) TypeNames() []string {
	names := make([]string, len(DefaultTypes))
	for i, t := range DefaultTypes {
		names[i] = t.Name
	}
	return names
}

// IsValidType returns true if the type is a valid hardcoded type.
func (c *Config) IsValidType(typeName string) bool {
	for _, t := range DefaultTypes {
		if t.Name == typeName {
			return true
		}
	}
	return false
}

// TypeList returns a comma-separated list of valid types.
func (c *Config) TypeList() string {
	names := make([]string, len(DefaultTypes))
	for i, t := range DefaultTypes {
		names[i] = t.Name
	}
	return strings.Join(names, ", ")
}

// BeanColors holds resolved color information for rendering a bean
type BeanColors struct {
	StatusColor   string
	TypeColor     string
	PriorityColor string
	IsArchive     bool
}

// GetBeanColors returns the resolved colors for a bean based on its status, type, and priority.
func (c *Config) GetBeanColors(status, typeName, priority string) BeanColors {
	colors := BeanColors{
		StatusColor:   "gray",
		TypeColor:     "",
		PriorityColor: "",
		IsArchive:     false,
	}

	if statusCfg := c.GetStatus(status); statusCfg != nil {
		colors.StatusColor = statusCfg.Color
	}
	colors.IsArchive = c.IsArchiveStatus(status)

	if typeCfg := c.GetType(typeName); typeCfg != nil {
		colors.TypeColor = typeCfg.Color
	}

	if priorityCfg := c.GetPriority(priority); priorityCfg != nil {
		colors.PriorityColor = priorityCfg.Color
	}

	return colors
}

// GetPriority returns the PriorityConfig for a given priority name, or nil if not found.
func (c *Config) GetPriority(name string) *PriorityConfig {
	for i := range DefaultPriorities {
		if DefaultPriorities[i].Name == name {
			return &DefaultPriorities[i]
		}
	}
	return nil
}

// PriorityNames returns a slice of valid priority names in order from highest to lowest.
func (c *Config) PriorityNames() []string {
	names := make([]string, len(DefaultPriorities))
	for i, p := range DefaultPriorities {
		names[i] = p.Name
	}
	return names
}

// IsValidPriority returns true if the priority is a valid hardcoded priority.
// Empty string is valid (means no priority set).
func (c *Config) IsValidPriority(priority string) bool {
	if priority == "" {
		return true
	}
	for _, p := range DefaultPriorities {
		if p.Name == priority {
			return true
		}
	}
	return false
}

// PriorityList returns a comma-separated list of valid priorities.
func (c *Config) PriorityList() string {
	names := make([]string, len(DefaultPriorities))
	for i, p := range DefaultPriorities {
		names[i] = p.Name
	}
	return strings.Join(names, ", ")
}

// boolPtr returns a pointer to the given bool value.
func boolPtr(b bool) *bool { return &b }

// DefaultWorktreeBaseRef is the default base ref for new worktree branches.
const DefaultWorktreeBaseRef = "main"

// ResolveWorktreePath returns the absolute path to the directory where worktrees
// should be created. If worktree.path is configured, it is used (with ~ expansion).
// Otherwise, defaults to ~/.beans/worktrees/<projectName>/.
// projectName is used only when computing the default path.
func (c *Config) ResolveWorktreePath(projectName string) (string, error) {
	if c.Worktree.Path != "" {
		return expandHome(c.Worktree.Path)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	if projectName == "" {
		projectName = "default"
	}
	return filepath.Join(home, ".beans", "worktrees", projectName), nil
}

// expandHome expands a leading ~ in a path to the user's home directory.
func expandHome(path string) (string, error) {
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, path[1:]), nil
	}
	if filepath.IsAbs(path) {
		return path, nil
	}
	// Relative paths resolve against cwd
	return filepath.Abs(path)
}

// GetWorktreeBaseRef returns the configured base ref for new worktree branches.
// Returns "main" if not set.
func (c *Config) GetWorktreeBaseRef() string {
	if c.Worktree.BaseRef == "" {
		return DefaultWorktreeBaseRef
	}
	return c.Worktree.BaseRef
}

// GetWorktreeSetup returns the configured setup command for new worktrees.
func (c *Config) GetWorktreeSetup() string {
	return c.Worktree.Setup
}

// GetWorktreeRun returns the configured run command for worktrees.
func (c *Config) GetWorktreeRun() string {
	return c.Worktree.Run
}

// GetWorktreeFetchTimeout returns the configured fetch timeout as a time.Duration.
// Returns 10s by default. Returns 0 if explicitly set to 0 (disables fetch).
func (c *Config) GetWorktreeFetchTimeout() time.Duration {
	if c.Worktree.FetchTimeout == nil {
		return 10 * time.Second
	}
	return time.Duration(*c.Worktree.FetchTimeout) * time.Second
}

// GetWorktreeIntegrate returns the configured integration mode.
// Returns "local" if not set or invalid.
func (c *Config) GetWorktreeIntegrate() IntegrateMode {
	switch c.Worktree.Integrate {
	case IntegrateModeLocal, IntegrateModePR:
		return c.Worktree.Integrate
	default:
		return IntegrateModeLocal
	}
}

// IsAgentEnabled returns whether agent functionality is enabled.
// Returns true if not explicitly set.
func (c *Config) IsAgentEnabled() bool {
	if c.Agent.Enabled == nil {
		return true
	}
	return *c.Agent.Enabled
}

// GetDefaultMode returns the configured default permission mode for agent sessions.
// Returns "act" if not set or invalid. Also accepts "yolo" as a backwards-compatible alias.
func (c *Config) GetDefaultMode() PermissionMode {
	switch c.Agent.DefaultMode {
	case PermissionModeAct, PermissionModePlan:
		return c.Agent.DefaultMode
	case "yolo":
		return PermissionModeAct // backwards-compatible alias
	default:
		return PermissionModeAct
	}
}

// GetDefaultEffort returns the raw configured default effort level for agent sessions.
// Returns empty string if not set. Use IsValidEffortLevel to validate before use.
func (c *Config) GetDefaultEffort() string {
	return c.Agent.DefaultEffort
}

// IsValidEffortLevel returns true if the effort level is a valid value.
func IsValidEffortLevel(effort string) bool {
	switch effort {
	case "low", "medium", "high", "max":
		return true
	default:
		return false
	}
}

// IsValidPermissionMode returns true if the mode is a valid permission mode.
func IsValidPermissionMode(mode string) bool {
	switch PermissionMode(mode) {
	case PermissionModeAct, PermissionModePlan, "yolo":
		return true
	default:
		return false
	}
}

// GetProjectName returns the configured project name, or empty string if not set.
func (c *Config) GetProjectName() string {
	return c.Project.Name
}

// GetServerPort returns the configured server port, or the default if not set.
func (c *Config) GetServerPort() int {
	if c.Server.Port == 0 {
		return DefaultServerPort
	}
	return c.Server.Port
}

// GetCORSOrigins returns the configured CORS origins, or the defaults if not set.
func (c *Config) GetCORSOrigins() []string {
	if len(c.Server.CORSOrigins) > 0 {
		return c.Server.CORSOrigins
	}
	return []string{"http://localhost:*", "http://127.0.0.1:*"}
}
