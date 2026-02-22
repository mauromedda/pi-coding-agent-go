// ABOUTME: SeparatorModel is a Bubble Tea leaf that renders a full-width horizontal rule
// ABOUTME: Port of components/separator.go; uses Border color from theme palette

package btea

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// SeparatorModel renders a horizontal line of "─" characters spanning the width.
type SeparatorModel struct {
	width int
}

// NewSeparatorModel creates a SeparatorModel with zero width (set via WindowSizeMsg).
func NewSeparatorModel() SeparatorModel {
	return SeparatorModel{}
}

// Init returns nil; no commands needed for a static separator.
func (m SeparatorModel) Init() tea.Cmd {
	return nil
}

// Update handles tea.WindowSizeMsg to track terminal width.
func (m SeparatorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = ws.Width
	}
	return m, nil
}

// View renders the separator line using the Border style from the theme.
func (m SeparatorModel) View() string {
	if m.width <= 0 {
		return ""
	}
	s := Styles()
	return s.Border.Render(strings.Repeat("─", m.width))
}
