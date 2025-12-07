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
	case "open":
		return StatusOpen.Render(status)
	case "done":
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
	case "open":
		return StatusOpenText.Render(status)
	case "done":
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

// BeanRowConfig holds configuration for rendering a bean row
type BeanRowConfig struct {
	StatusColor   string
	TypeColor     string
	IsArchive     bool
	MaxTitleWidth int  // 0 means no truncation
	ShowCursor    bool // Show selection cursor
	IsSelected    bool
	Tags          []string // Tags to display (optional)
	ShowTags      bool     // Whether to show tags column
	TagsColWidth  int      // Width of tags column (0 = default)
	MaxTags       int      // Max tags to show (0 = default of 1)
}

// Base column widths for bean lists (minimum sizes)
const (
	ColWidthID     = 12
	ColWidthStatus = 14
	ColWidthType   = 12
	ColWidthTags   = 24
)

// ResponsiveColumns holds calculated column widths based on available space
type ResponsiveColumns struct {
	ID       int
	Status   int
	Type     int
	Tags     int
	MaxTags  int // How many tags to show
	ShowTags bool
}

// CalculateResponsiveColumns determines column widths based on available width.
// It distributes extra space to tags (more tags) and title (remaining space).
func CalculateResponsiveColumns(totalWidth int, hasTags bool) ResponsiveColumns {
	// Fixed columns
	cols := ResponsiveColumns{
		ID:       ColWidthID,
		Status:   ColWidthStatus,
		Type:     ColWidthType,
		Tags:     0,
		MaxTags:  0,
		ShowTags: false,
	}

	// Cursor takes 2 chars
	cursorWidth := 2

	// Base width without tags
	baseWidth := cursorWidth + cols.ID + cols.Status + cols.Type

	// Minimum title width we want to preserve (kept low to favor title space)
	minTitleWidth := 20

	// Available space for tags + title
	available := totalWidth - baseWidth

	// Only show tags if we have them and there's enough room
	// Use higher thresholds to give more space to titles
	if hasTags && available > minTitleWidth+ColWidthTags+20 {
		cols.ShowTags = true

		// Calculate how many tags we can show based on available space
		// Be conservative - favor title width over showing more tags
		spaceForTags := available - minTitleWidth - 20 // extra 20 for title breathing room

		if spaceForTags >= 70 {
			// Lots of space: show 3 tags, wider column
			cols.Tags = 50
			cols.MaxTags = 3
		} else if spaceForTags >= 55 {
			// Good space: show 2 tags
			cols.Tags = 38
			cols.MaxTags = 2
		} else if spaceForTags >= ColWidthTags {
			// Minimal: show 1 tag
			cols.Tags = ColWidthTags
			cols.MaxTags = 1
		} else {
			// Not enough room for tags
			cols.ShowTags = false
		}
	}

	return cols
}

// RenderBeanRow renders a bean as a single row with ID, Type, Status, Tags (optional), Title
func RenderBeanRow(id, status, typeName, title string, cfg BeanRowConfig) string {
	// Column styles - use responsive widths if provided
	idStyle := lipgloss.NewStyle().Width(ColWidthID)
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

	// Build columns
	idCol := idStyle.Render(ID.Render(id))

	typeText := ""
	if typeName != "" {
		typeText = RenderTypeText(typeName, cfg.TypeColor)
	}
	typeCol := typeStyle.Render(typeText)

	statusCol := statusStyle.Render(RenderStatusTextWithColor(status, cfg.StatusColor, cfg.IsArchive))

	// Tags column (optional)
	var tagsCol string
	if cfg.ShowTags {
		tagsCol = tagsStyle.Render(RenderTagsCompact(cfg.Tags, maxTags))
	}

	// Title (truncate if needed)
	displayTitle := title
	if cfg.MaxTitleWidth > 0 && len(title) > cfg.MaxTitleWidth {
		displayTitle = title[:cfg.MaxTitleWidth-3] + "..."
	}

	// Cursor and title styling
	var cursor string
	var titleStyled string
	if cfg.ShowCursor {
		if cfg.IsSelected {
			cursor = lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render("▌") + " "
			titleStyled = lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render(displayTitle)
		} else {
			cursor = "  "
			titleStyled = displayTitle
		}
	} else {
		cursor = ""
		titleStyled = displayTitle
	}

	if cfg.ShowTags {
		return cursor + idCol + typeCol + statusCol + tagsCol + titleStyled
	}
	return cursor + idCol + typeCol + statusCol + titleStyled
}
