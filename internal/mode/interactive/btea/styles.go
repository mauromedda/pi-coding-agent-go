// ABOUTME: Lipgloss style bridge from theme.Color ANSI escape codes
// ABOUTME: Parses SGR sequences into lipgloss styles; Styles() returns full palette

package btea

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/charmbracelet/lipgloss"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/theme"
)

// themeStylesEntry pairs a theme pointer with its pre-built styles.
type themeStylesEntry struct {
	theme  *theme.Theme
	styles ThemeStyles
}

// cachedStyles is the package-level atomic cache for ThemeStyles.
// Cache key is the theme pointer identity; invalidated when theme changes.
var cachedStyles atomic.Pointer[themeStylesEntry]

// sgrRe matches a single ANSI SGR sequence like \x1b[38;5;208m.
var sgrRe = regexp.MustCompile(`\x1b\[([\d;]+)m`)

// attrs holds parsed text attributes from an ANSI escape sequence.
type attrs struct {
	bold      bool
	dim       bool
	italic    bool
	underline bool
	reverse   bool
}

// extractColor parses an ANSI escape code string and returns a lipgloss color
// spec (e.g. "208" for 256-color, "1" for basic). Returns "" if no color is
// found (attribute-only codes like bold or dim).
func extractColor(code string) string {
	matches := sgrRe.FindAllStringSubmatch(code, -1)
	// Process all matches; last color-bearing sequence wins.
	var result string
	for _, m := range matches {
		params := strings.Split(m[1], ";")
		if c := parseColorParams(params); c != "" {
			result = c
		}
	}
	return result
}

// parseColorParams interprets SGR parameter list and returns a lipgloss color
// spec if the params contain a color. Returns "" for attribute-only params.
func parseColorParams(params []string) string {
	if len(params) == 0 {
		return ""
	}

	// 256-color: 38;5;N (fg) or 48;5;N (bg)
	if len(params) >= 3 && (params[0] == "38" || params[0] == "48") && params[1] == "5" {
		return params[2]
	}

	// Single param: basic fg/bg color or attribute
	if len(params) == 1 {
		n, err := strconv.Atoi(params[0])
		if err != nil {
			return ""
		}
		return basicColorToSpec(n)
	}

	return ""
}

// basicColorToSpec converts a basic ANSI color code to a lipgloss 256-color
// spec. Returns "" for non-color codes (attributes like bold, dim).
func basicColorToSpec(n int) string {
	switch {
	case n >= 30 && n <= 37:
		return fmt.Sprintf("%d", n-30)
	case n >= 40 && n <= 47:
		return fmt.Sprintf("%d", n-40)
	case n >= 90 && n <= 97:
		return fmt.Sprintf("%d", n-90+8)
	case n >= 100 && n <= 107:
		return fmt.Sprintf("%d", n-100+8)
	default:
		return ""
	}
}

// extractAttrs parses text attributes (bold, dim, italic, underline, reverse)
// from an ANSI escape code string.
func extractAttrs(code string) attrs {
	matches := sgrRe.FindAllStringSubmatch(code, -1)
	var a attrs
	for _, m := range matches {
		for p := range strings.SplitSeq(m[1], ";") {
			switch p {
			case "1":
				a.bold = true
			case "2":
				a.dim = true
			case "3":
				a.italic = true
			case "4":
				a.underline = true
			case "7":
				a.reverse = true
			}
		}
	}
	return a
}

// isBackground returns true if the ANSI code sets a background color.
func isBackground(code string) bool {
	matches := sgrRe.FindAllStringSubmatch(code, -1)
	for _, m := range matches {
		params := strings.Split(m[1], ";")
		if len(params) >= 3 && params[0] == "48" && params[1] == "5" {
			return true
		}
		if len(params) == 1 {
			n, err := strconv.Atoi(params[0])
			if err == nil && ((n >= 40 && n <= 47) || (n >= 100 && n <= 107)) {
				return true
			}
		}
	}
	return false
}

// colorToStyle builds a lipgloss.Style from a raw ANSI escape code string.
func colorToStyle(code string) lipgloss.Style {
	s := lipgloss.NewStyle()
	c := extractColor(code)
	if c != "" {
		if isBackground(code) {
			s = s.Background(lipgloss.Color(c))
		} else {
			s = s.Foreground(lipgloss.Color(c))
		}
	}
	a := extractAttrs(code)
	if a.bold {
		s = s.Bold(true)
	}
	if a.dim {
		s = s.Faint(true)
	}
	if a.italic {
		s = s.Italic(true)
	}
	if a.underline {
		s = s.Underline(true)
	}
	if a.reverse {
		s = s.Reverse(true)
	}
	return s
}

// ThemeStyles holds pre-built lipgloss styles for all semantic palette fields.
type ThemeStyles struct {
	Primary   lipgloss.Style
	Secondary lipgloss.Style
	Muted     lipgloss.Style
	Accent    lipgloss.Style

	Success lipgloss.Style
	Warning lipgloss.Style
	Error   lipgloss.Style
	Info    lipgloss.Style

	Border        lipgloss.Style
	Selection     lipgloss.Style
	Prompt        lipgloss.Style
	BashSeparator lipgloss.Style

	ToolRead  lipgloss.Style
	ToolBash  lipgloss.Style
	ToolWrite lipgloss.Style
	ToolEdit  lipgloss.Style
	ToolGrep  lipgloss.Style
	ToolOther lipgloss.Style

	FooterPath   lipgloss.Style
	FooterBranch lipgloss.Style
	FooterModel  lipgloss.Style
	FooterCost   lipgloss.Style
	FooterPerm   lipgloss.Style

	UserBg lipgloss.Style

	Bold      lipgloss.Style
	Dim       lipgloss.Style
	Italic    lipgloss.Style
	Underline lipgloss.Style
}

// Styles returns ThemeStyles for the current theme, using a cached value when
// the theme pointer has not changed. This avoids rebuilding 32 lipgloss styles
// (each requiring 3 regex scans) on every View() call.
func Styles() ThemeStyles {
	t := theme.Current()
	if e := cachedStyles.Load(); e != nil && e.theme == t {
		return e.styles
	}
	s := buildStyles(t)
	cachedStyles.Store(&themeStylesEntry{theme: t, styles: s})
	return s
}

// buildStyles constructs ThemeStyles from a theme's palette.
func buildStyles(t *theme.Theme) ThemeStyles {
	p := t.Palette
	return ThemeStyles{
		Primary:   colorToStyle(p.Primary.Code()),
		Secondary: colorToStyle(p.Secondary.Code()),
		Muted:     colorToStyle(p.Muted.Code()),
		Accent:    colorToStyle(p.Accent.Code()),

		Success: colorToStyle(p.Success.Code()),
		Warning: colorToStyle(p.Warning.Code()),
		Error:   colorToStyle(p.Error.Code()),
		Info:    colorToStyle(p.Info.Code()),

		Border:        colorToStyle(p.Border.Code()),
		Selection:     colorToStyle(p.Selection.Code()),
		Prompt:        colorToStyle(p.Prompt.Code()),
		BashSeparator: colorToStyle(p.BashSeparator.Code()),

		ToolRead:  colorToStyle(p.ToolRead.Code()),
		ToolBash:  colorToStyle(p.ToolBash.Code()),
		ToolWrite: colorToStyle(p.ToolWrite.Code()),
		ToolEdit:  colorToStyle(p.ToolEdit.Code()),
		ToolGrep:  colorToStyle(p.ToolGrep.Code()),
		ToolOther: colorToStyle(p.ToolOther.Code()),

		FooterPath:   colorToStyle(p.FooterPath.Code()),
		FooterBranch: colorToStyle(p.FooterBranch.Code()),
		FooterModel:  colorToStyle(p.FooterModel.Code()),
		FooterCost:   colorToStyle(p.FooterCost.Code()),
		FooterPerm:   colorToStyle(p.FooterPerm.Code()),

		UserBg: colorToStyle(p.UserBg.Code()),

		Bold:      colorToStyle(p.Bold.Code()),
		Dim:       colorToStyle(p.Dim.Code()),
		Italic:    colorToStyle(p.Italic.Code()),
		Underline: colorToStyle(p.Underline.Code()),
	}
}
