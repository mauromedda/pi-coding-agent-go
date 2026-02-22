// ABOUTME: Semantic color theme types: Color, Palette, Theme
// ABOUTME: Color.Apply wraps text in ANSI codes; Palette maps semantic roles to colors

package theme

// Color represents a terminal color that can style text.
type Color struct {
	code string
}

// NewColor creates a Color from a raw ANSI escape code.
func NewColor(code string) Color {
	return Color{code: code}
}

// Apply wraps text with the ANSI color code and a reset suffix.
// If the color code is empty, the text is returned unchanged.
func (c Color) Apply(text string) string {
	if c.code == "" {
		return text
	}
	return c.code + text + "\x1b[0m"
}

// Code returns the raw ANSI escape code.
func (c Color) Code() string {
	return c.code
}

// Bold returns a new Color that prepends bold (\x1b[1m) to the code.
func (c Color) Bold() Color {
	return Color{code: "\x1b[1m" + c.code}
}

// Dim returns a new Color that prepends dim (\x1b[2m) to the code.
func (c Color) Dim() Color {
	return Color{code: "\x1b[2m" + c.code}
}

// Palette holds all semantic colors for a theme.
type Palette struct {
	// Text
	Primary   Color
	Secondary Color
	Muted     Color
	Accent    Color

	// Semantic
	Success Color
	Warning Color
	Error   Color
	Info    Color

	// UI
	Border    Color
	Selection Color
	Prompt    Color

	// Tool categories
	ToolRead  Color
	ToolBash  Color
	ToolWrite Color
	ToolOther Color

	// Footer
	FooterPath   Color
	FooterBranch Color
	FooterModel  Color
	FooterCost   Color
	FooterPerm   Color

	// Regions
	UserBg Color // Background for user messages

	// Formatting
	Bold      Color
	Dim       Color
	Italic    Color
	Underline Color
}

// Theme holds a named palette.
type Theme struct {
	Name    string  `json:"name"`
	Palette Palette `json:"palette"`
}

// DefaultPalette returns the palette matching the current hardcoded colors.
func DefaultPalette() Palette {
	return Palette{
		// Text
		Primary:   NewColor("\x1b[0m"),
		Secondary: NewColor("\x1b[90m"),
		Muted:     NewColor("\x1b[2m"),
		Accent:    NewColor("\x1b[38;5;208m"),

		// Semantic
		Success: NewColor("\x1b[32m"),
		Warning: NewColor("\x1b[33m"),
		Error:   NewColor("\x1b[31m"),
		Info:    NewColor("\x1b[36m"),

		// UI
		Border:    NewColor("\x1b[90m"),
		Selection: NewColor("\x1b[7m"),
		Prompt:    NewColor("\x1b[1m"),

		// Tool categories
		ToolRead:  NewColor("\x1b[36m"),
		ToolBash:  NewColor("\x1b[33m"),
		ToolWrite: NewColor("\x1b[32m"),
		ToolOther: NewColor("\x1b[35m"),

		// Footer
		FooterPath:   NewColor("\x1b[1m"),
		FooterBranch: NewColor("\x1b[36m"),
		FooterModel:  NewColor("\x1b[36m"),
		FooterCost:   NewColor("\x1b[33m"),
		FooterPerm:   NewColor("\x1b[32m"),

		// Regions
		UserBg: NewColor("\x1b[100m"),

		// Formatting
		Bold:      NewColor("\x1b[1m"),
		Dim:       NewColor("\x1b[2m"),
		Italic:    NewColor("\x1b[3m"),
		Underline: NewColor("\x1b[4m"),
	}
}
