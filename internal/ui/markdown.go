package ui

import (
	"os"

	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

// DarkMarkdownStyle is glamour's dark style without its document margin, so
// rendered markdown starts at column 0.
func DarkMarkdownStyle() ansi.StyleConfig {
	return withoutDocumentMargin(styles.DarkStyleConfig)
}

// AutoMarkdownStyle picks a style the way glamour.WithAutoStyle does - plain
// when stdout is not a terminal, otherwise dark or light by background - and
// removes its document margin.
func AutoMarkdownStyle() ansi.StyleConfig {
	switch {
	case !term.IsTerminal(int(os.Stdout.Fd())):
		return withoutDocumentMargin(styles.NoTTYStyleConfig)
	case lipgloss.HasDarkBackground():
		return withoutDocumentMargin(styles.DarkStyleConfig)
	default:
		return withoutDocumentMargin(styles.LightStyleConfig)
	}
}

// withoutDocumentMargin returns a copy of cfg; the package-level glamour
// styles stay untouched because only the copy's Margin pointer is replaced.
func withoutDocumentMargin(cfg ansi.StyleConfig) ansi.StyleConfig {
	zero := uint(0)
	cfg.Document.Margin = &zero
	return cfg
}
