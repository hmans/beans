package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Color palette
var (
	ColorPrimary   = lipgloss.Color("#7C3AED") // Purple
	ColorSecondary = lipgloss.Color("#6B7280") // Gray
	ColorSuccess   = lipgloss.Color("#10B981") // Green
	ColorWarning   = lipgloss.Color("#F59E0B") // Amber
	ColorDanger    = lipgloss.Color("#EF4444") // Red
	ColorMuted     = lipgloss.Color("#9CA3AF") // Light gray
	ColorSubtle    = lipgloss.Color("#555555") // Dark gray (for tree lines)
	ColorBlue      = lipgloss.Color("#3B82F6") // Blue
	ColorCyan      = lipgloss.Color("14")      // Bright Cyan (ANSI)
)

// NamedColors maps color names to lipgloss colors.
var NamedColors = map[string]lipgloss.Color{
	"green":  ColorSuccess,
	"yellow": ColorWarning,
	"red":    ColorDanger,
	"gray":   ColorSecondary,
	"grey":   ColorSecondary,
	"blue":   ColorBlue,
	"purple": ColorPrimary,
	"cyan":   ColorCyan,
}

// ResolveColor converts a color name or hex code to a lipgloss.Color.
func ResolveColor(color string) lipgloss.Color {
	if strings.HasPrefix(color, "#") {
		return lipgloss.Color(color)
	}
	if c, ok := NamedColors[strings.ToLower(color)]; ok {
		return c
	}
	return ColorMuted
}

// IsValidColor returns true if the color is a valid named color or hex code.
func IsValidColor(color string) bool {
	if strings.HasPrefix(color, "#") {
		// Valid hex: #RGB or #RRGGBB
		return len(color) == 4 || len(color) == 7
	}
	_, ok := NamedColors[strings.ToLower(color)]
	return ok
}

// Status badge styles (for inline use, like in show command)
var (
	StatusOpen = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#fff")).
			Background(ColorSuccess).
			Padding(0, 1).
			Bold(true)

	StatusDone = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#fff")).
			Background(ColorSecondary).
			Padding(0, 1)

	StatusInProgress = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#fff")).
				Background(ColorWarning).
				Padding(0, 1).
				Bold(true)
)

// Status text styles (for table use, no background/padding)
var (
	StatusOpenText       = lipgloss.NewStyle().Foreground(ColorSuccess).Bold(true)
	StatusDoneText       = lipgloss.NewStyle().Foreground(ColorSecondary)
	StatusInProgressText = lipgloss.NewStyle().Foreground(ColorWarning).Bold(true)
)

// Tag badge style - black text on gray background
var TagBadge = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#000")).
	Background(ColorMuted).
	Padding(0, 1)

// RenderTag renders a single tag as a badge
func RenderTag(tag string) string {
	return TagBadge.Render(tag)
}

// RenderTags renders multiple tags as badges separated by spaces
func RenderTags(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	rendered := make([]string, len(tags))
	for i, tag := range tags {
		rendered[i] = RenderTag(tag)
	}
	return strings.Join(rendered, " ")
}

// RenderTagsCompact renders tags for list views with a max count.
// Shows up to maxTags badges, with "+N" indicator if there are more.
// Tags longer than 12 chars are truncated.
func RenderTagsCompact(tags []string, maxTags int) string {
	if len(tags) == 0 {
		return ""
	}
	if maxTags <= 0 {
		maxTags = 1
	}

	showTags := tags
	var extra int
	if len(tags) > maxTags {
		showTags = tags[:maxTags]
		extra = len(tags) - maxTags
	}

	rendered := make([]string, len(showTags))
	for i, tag := range showTags {
		// Truncate long tags
		displayTag := tag
		if len(displayTag) > 12 {
			displayTag = displayTag[:10] + ".."
		}
		rendered[i] = RenderTag(displayTag)
	}

	result := strings.Join(rendered, " ")
	if extra > 0 {
		result += Muted.Render(fmt.Sprintf(" +%d", extra))
	}
	return result
}

// Text styles
var (
	Bold      = lipgloss.NewStyle().Bold(true)
	Muted     = lipgloss.NewStyle().Foreground(ColorMuted)
	Primary   = lipgloss.NewStyle().Foreground(ColorPrimary)
	Success   = lipgloss.NewStyle().Foreground(ColorSuccess)
	Warning   = lipgloss.NewStyle().Foreground(ColorWarning)
	Danger    = lipgloss.NewStyle().Foreground(ColorDanger)
	Secondary = lipgloss.NewStyle().Foreground(ColorSecondary)
)

// ID style - distinctive for bean IDs
var ID = lipgloss.NewStyle().
	Foreground(ColorPrimary).
	Bold(true)

// TreeLine style - subtle for tree connectors
var TreeLine = lipgloss.NewStyle().Foreground(ColorSubtle)

// Title style
var Title = lipgloss.NewStyle().Bold(true)

// Path style - subdued
var Path = lipgloss.NewStyle().Foreground(ColorMuted)

// Header style for section headers
var Header = lipgloss.NewStyle().
	Foreground(ColorPrimary).
	Bold(true).
	MarginBottom(1)

// RenderStatus returns a styled status badge based on the status string (legacy, uses hardcoded colors)
func RenderStatus(status string) string {
	switch status {
	case "todo", "draft":
		return StatusOpen.Render(status)
	case "completed", "scrapped":
		return StatusDone.Render(status)
	case "in-progress", "in_progress":
		return StatusInProgress.Render(status)
	default:
		return Muted.Render(status)
	}
}

// RenderStatusText returns styled status text (for tables, no background) (legacy, uses hardcoded colors)
func RenderStatusText(status string) string {
	switch status {
	case "todo", "draft":
		return StatusOpenText.Render(status)
	case "completed", "scrapped":
		return StatusDoneText.Render(status)
	case "in-progress", "in_progress":
		return StatusInProgressText.Render("in-progress")
	default:
		return Muted.Render(status)
	}
}

// RenderStatusWithColor returns a styled status badge using the specified color.
func RenderStatusWithColor(status, color string, isArchiveStatus bool) string {
	c := ResolveColor(color)
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#fff")).
		Background(c).
		Padding(0, 1)

	if !isArchiveStatus {
		style = style.Bold(true)
	}

	return style.Render(status)
}

// RenderStatusTextWithColor returns styled status text (for tables) using the specified color.
func RenderStatusTextWithColor(status, color string, isArchiveStatus bool) string {
	c := ResolveColor(color)
	style := lipgloss.NewStyle().Foreground(c)

	if !isArchiveStatus {
		style = style.Bold(true)
	}

	return style.Render(status)
}

// RenderTypeText returns styled type text using the specified color.
// If color is empty, uses muted styling.
func RenderTypeText(typeName, color string) string {
	if typeName == "" {
		return ""
	}
	if color == "" {
		return Muted.Render(typeName)
	}
	c := ResolveColor(color)
	return lipgloss.NewStyle().Foreground(c).Render(typeName)
}

// RenderTypeWithColor returns a styled type badge with colored background.
func RenderTypeWithColor(typeName, color string) string {
	if typeName == "" {
		return ""
	}
	c := ResolveColor(color)
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#fff")).
		Background(c).
		Bold(true).
		Padding(0, 1)
	return style.Render(typeName)
}

// RenderPriorityWithColor returns a styled priority badge using the specified color.
func RenderPriorityWithColor(priority, color string) string {
	if priority == "" {
		return ""
	}
	c := ResolveColor(color)
	style := lipgloss.NewStyle().
		Foreground(c).
		Bold(priority == "critical" || priority == "high")
	return style.Render("[" + priority + "]")
}

// RenderPriorityText returns styled priority text for tables.
func RenderPriorityText(priority, color string) string {
	if priority == "" {
		return ""
	}
	c := ResolveColor(color)
	style := lipgloss.NewStyle().Foreground(c)
	if priority == "critical" || priority == "high" {
		style = style.Bold(true)
	}
	return style.Render(priority)
}

// ShortType returns a single-character code for the bean type.
func ShortType(t string) string {
	switch t {
	case "milestone":
		return "M"
	case "epic":
		return "E"
	case "bug":
		return "B"
	case "feature":
		return "F"
	case "task":
		return "T"
	default:
		return "?"
	}
}

// ShortStatus returns a single-character code for the bean status.
func ShortStatus(s string) string {
	switch s {
	case "draft":
		return "D"
	case "todo":
		return "T"
	case "in-progress":
		return "I"
	case "completed":
		return "C"
	case "scrapped":
		return "S"
	default:
		return "?"
	}
}

// GetPrioritySymbol returns the raw symbol for a priority without styling.
// Returns empty string for normal/empty priority.
func GetPrioritySymbol(priority string) string {
	switch priority {
	case "critical":
		return "‼"
	case "high":
		return "!"
	case "low":
		return "↓"
	case "deferred":
		return "→"
	default:
		return ""
	}
}

// RenderPrioritySymbol returns a compact symbol for priority (used in TUI).
// Returns empty string for normal/empty priority.
func RenderPrioritySymbol(priority, color string) string {
	symbol := GetPrioritySymbol(priority)
	if symbol == "" {
		return ""
	}

	c := ResolveColor(color)
	style := lipgloss.NewStyle().Foreground(c)
	if priority == "critical" || priority == "high" {
		style = style.Bold(true)
	}
	return style.Render(symbol)
}

// BeanRowConfig holds configuration for rendering a bean row
type BeanRowConfig struct {
	StatusColor   string
	TypeColor     string
	PriorityColor string
	Priority      string // Priority value (critical, high, normal, low, deferred)
	IsArchive     bool
	MaxTitleWidth int  // 0 means no truncation
	ShowCursor    bool // Show selection cursor
	IsSelected    bool
	IsMarked      bool     // Marked for multi-select batch operations
	Tags          []string // Tags to display (optional)
	ShowTags      bool     // Whether to show tags column
	TagsColWidth  int      // Width of tags column (0 = default)
	MaxTags       int      // Max tags to show (0 = default of 1)
	TreePrefix    string   // Tree prefix (e.g., "├─" or "  └─") to prepend to ID
	Dimmed        bool     // Render row dimmed (for unmatched ancestor beans in tree)
	IDColWidth    int      // Width of ID column (0 = default of ColWidthID)
	UseFullNames  bool     // Use full type/status names instead of single-char abbreviations
}

// Base column widths for bean lists (minimum sizes)
const (
	ColWidthID     = 12
	ColWidthStatus = 3
	ColWidthType   = 3
	ColWidthTags   = 24
)

// ResponsiveColumns holds calculated column widths based on available space
type ResponsiveColumns struct {
	ID                int
	Status            int
	Type              int
	Tags              int
	MaxTags           int  // How many tags to show
	ShowTags          bool
	UseFullTypeStatus bool // Use full names instead of single-char abbreviations
}

// CalculateResponsiveColumns determines column widths based on available width.
// Prioritizes title width - tags are only shown when there's plenty of room.
func CalculateResponsiveColumns(totalWidth int, hasTags bool) ResponsiveColumns {
	cols := ResponsiveColumns{
		ID:       ColWidthID,
		Status:   ColWidthStatus,
		Type:     ColWidthType,
		Tags:     0,
		MaxTags:  0,
		ShowTags: false,
	}

	// Use full type/status names when terminal is wide enough
	const minWidthForFullNames = 120
	if totalWidth >= minWidthForFullNames {
		cols.UseFullTypeStatus = true
		cols.Status = 12 // "in-progress" needs 11 chars
		cols.Type = 10   // "milestone" needs 9 chars
	}

	// Don't show tags in narrow viewports - prioritize title space
	// Only consider showing tags if terminal is wide enough (140+ columns)
	const minWidthForTags = 140

	if !hasTags || totalWidth < minWidthForTags {
		return cols
	}

	// At this point we have at least 140 columns
	// Base usage: cursor (2) + ID + status + type (use responsive widths)
	cursorWidth := 2
	baseWidth := cursorWidth + cols.ID + cols.Status + cols.Type
	available := totalWidth - baseWidth

	// Reserve generous space for title, then allocate remaining to tags
	minTitleWidth := 50
	spaceForTags := available - minTitleWidth

	if spaceForTags >= ColWidthTags {
		cols.ShowTags = true

		if spaceForTags >= 80 {
			// Lots of space: show all tags (up to 5)
			cols.Tags = 70
			cols.MaxTags = 5
		} else if spaceForTags >= 60 {
			// Good space: show 4 tags
			cols.Tags = 55
			cols.MaxTags = 4
		} else if spaceForTags >= 45 {
			// Moderate space: show 3 tags
			cols.Tags = 42
			cols.MaxTags = 3
		} else if spaceForTags >= 35 {
			// Limited space: show 2 tags
			cols.Tags = 32
			cols.MaxTags = 2
		} else {
			// Minimal: show 1 tag
			cols.Tags = ColWidthTags
			cols.MaxTags = 1
		}
	}

	return cols
}

// RenderBeanRow renders a bean as a single row with ID, Type, Status, Tags (optional), Title
func RenderBeanRow(id, status, typeName, title string, cfg BeanRowConfig) string {
	// Column styles - use responsive widths if provided
	idColWidth := ColWidthID
	if cfg.IDColWidth > 0 {
		idColWidth = cfg.IDColWidth
	}
	typeStyle := lipgloss.NewStyle().Width(ColWidthType)
	statusStyle := lipgloss.NewStyle().Width(ColWidthStatus)

	tagsColWidth := ColWidthTags
	if cfg.TagsColWidth > 0 {
		tagsColWidth = cfg.TagsColWidth
	}
	tagsStyle := lipgloss.NewStyle().Width(tagsColWidth)

	maxTags := 1
	if cfg.MaxTags > 0 {
		maxTags = cfg.MaxTags
	}

	// Highlight style for marked rows
	highlightStyle := lipgloss.NewStyle().Foreground(ColorWarning)

	// Build ID column with manual padding
	// (lipgloss Width() doesn't correctly handle Unicode box-drawing characters)
	var idCol string
	// Calculate visual width: tree prefix (in runes) + ID length
	visualWidth := len([]rune(cfg.TreePrefix)) + len(id)
	padding := ""
	if idColWidth > visualWidth {
		padding = strings.Repeat(" ", idColWidth-visualWidth)
	}
	if cfg.Dimmed {
		idCol = Muted.Render(cfg.TreePrefix) + Muted.Render(id) + padding
	} else if cfg.IsMarked {
		// Only highlight the ID when marked
		idCol = highlightStyle.Render(cfg.TreePrefix) + highlightStyle.Render(id) + padding
	} else {
		idCol = TreeLine.Render(cfg.TreePrefix) + ID.Render(id) + padding
	}

	// Type column - single character or full name
	var typeStr string
	if cfg.UseFullNames {
		typeStr = typeName
		typeStyle = typeStyle.Width(12) // wider for full names
	} else {
		typeStr = ShortType(typeName)
	}
	var typeCol string
	if cfg.Dimmed {
		typeCol = typeStyle.Render(Muted.Render(typeStr))
	} else {
		typeCol = typeStyle.Render(RenderTypeText(typeStr, cfg.TypeColor))
	}

	// Status column - single character or full name
	var statusStr string
	if cfg.UseFullNames {
		statusStr = status
		statusStyle = statusStyle.Width(12) // wider for full names
	} else {
		statusStr = ShortStatus(status)
	}
	var statusCol string
	if cfg.Dimmed {
		statusCol = statusStyle.Render(Muted.Render(statusStr))
	} else {
		statusCol = statusStyle.Render(RenderStatusTextWithColor(statusStr, cfg.StatusColor, cfg.IsArchive))
	}

	// Tags column (optional)
	var tagsCol string
	if cfg.ShowTags {
		if cfg.Dimmed {
			if len(cfg.Tags) > 0 {
				tagsCol = tagsStyle.Render(Muted.Render(cfg.Tags[0]))
			} else {
				tagsCol = tagsStyle.Render("")
			}
		} else {
			tagsCol = tagsStyle.Render(RenderTagsCompact(cfg.Tags, maxTags))
		}
	}

	// Priority symbol (prepended to title)
	var prioritySymbol string
	if !cfg.Dimmed {
		prioritySymbol = RenderPrioritySymbol(cfg.Priority, cfg.PriorityColor)
		if prioritySymbol != "" {
			prioritySymbol += " "
		}
	}

	// Title (truncate if needed, accounting for priority symbol width)
	displayTitle := title
	titleColWidth := cfg.MaxTitleWidth // Save original for padding
	maxWidth := cfg.MaxTitleWidth
	if maxWidth > 0 && prioritySymbol != "" {
		maxWidth -= 2 // Account for symbol + space
	}
	if maxWidth > 3 && len(title) > maxWidth {
		displayTitle = title[:maxWidth-3] + "..."
	} else if maxWidth > 0 && maxWidth <= 3 && len(title) > maxWidth {
		displayTitle = title[:maxWidth]
	}

	// Cursor and title styling
	var cursor string
	var titleStyled string
	if cfg.ShowCursor {
		if cfg.IsSelected {
			cursor = lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render("▌")
			titleStyled = lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render(displayTitle)
		} else {
			cursor = " "
			if cfg.Dimmed {
				titleStyled = Muted.Render(displayTitle)
			} else {
				titleStyled = displayTitle
			}
		}
	} else {
		cursor = ""
		if cfg.Dimmed {
			titleStyled = Muted.Render(displayTitle)
		} else {
			titleStyled = displayTitle
		}
	}

	if cfg.ShowTags {
		// Pad title column to fixed width so tags align in a column
		// Calculate padding needed: titleColWidth - (priority symbol width + title length)
		titleLen := len(displayTitle)
		if prioritySymbol != "" {
			titleLen += 2 // symbol + space
		}
		padding := ""
		if titleColWidth > titleLen {
			padding = strings.Repeat(" ", titleColWidth-titleLen)
		}
		return cursor + idCol + typeCol + statusCol + prioritySymbol + titleStyled + padding + " " + tagsCol
	}
	return cursor + idCol + typeCol + statusCol + prioritySymbol + titleStyled
}
