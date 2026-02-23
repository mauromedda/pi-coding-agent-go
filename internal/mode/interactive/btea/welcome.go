// ABOUTME: WelcomeModel is a Bubble Tea leaf that renders the startup banner
// ABOUTME: Port of components/welcome_msg.go; shows ASCII pi, version, model, shortcuts

package btea

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/width"
)

// WelcomeModel renders the startup banner with version info, keyboard
// shortcuts, and registered tool count.
type WelcomeModel struct {
	version   string
	model     string
	cwd       string
	toolCount int
	width     int
}

// NewWelcomeModel creates a WelcomeModel with the given session details.
func NewWelcomeModel(version, model, cwd string, toolCount int) WelcomeModel {
	return WelcomeModel{
		version:   version,
		model:     model,
		cwd:       cwd,
		toolCount: toolCount,
	}
}

// Init returns nil; no commands needed for a static welcome banner.
func (m WelcomeModel) Init() tea.Cmd {
	return nil
}

// Update handles tea.WindowSizeMsg to track terminal width.
func (m WelcomeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = ws.Width
	}
	return m, nil
}

// View renders the full welcome banner with ASCII pi box, version info,
// keyboard shortcuts table, and tool count.
func (m WelcomeModel) View() string {
	s := Styles()
	ver := m.version
	if ver == "" {
		ver = "dev"
	}

	var b strings.Builder

	// ASCII π logo box
	b.WriteString(s.Accent.Render("  ╭───────╮") + "\n")
	b.WriteString(s.Accent.Render("  │  ") + s.Bold.Render("π") + s.Accent.Render("    │") + "\n")
	b.WriteString(s.Accent.Render("  ╰───────╯") + "\n")

	// Version, model, cwd
	b.WriteString(fmt.Sprintf("  %s %s\n", s.Bold.Render("pi-go"), s.Dim.Render("v"+ver)))
	b.WriteString(fmt.Sprintf("  %s\n", s.Dim.Render(m.model)))
	b.WriteString(fmt.Sprintf("  %s\n", s.Info.Render(m.cwd)))

	// Blank separator
	b.WriteString("\n")

	// Keyboard shortcuts: two-column layout with padded keys
	shortcuts := []struct {
		key  string
		desc string
	}{
		{"escape", "interrupt"},
		{"ctrl+c", "clear"},
		{"ctrl+c twice", "exit"},
		{"ctrl+d", "exit (empty)"},
		{"shift+tab", "cycle mode"},
		{"/", "commands"},
		{"!", "run bash"},
	}

	const keyPad = 16
	for _, sc := range shortcuts {
		padded := sc.key
		for len(padded) < keyPad {
			padded += " "
		}
		b.WriteString(fmt.Sprintf("  %s%s\n", s.Bold.Render(padded), s.Dim.Render(sc.desc)))
	}

	// Blank separator
	b.WriteString("\n")

	// Tool count
	b.WriteString(s.Dim.Render(fmt.Sprintf("  [Tools: %d registered]", m.toolCount)))

	// Truncate lines to terminal width on narrow terminals
	result := b.String()
	if m.width > 0 && m.width < 40 {
		lines := strings.Split(result, "\n")
		for i, line := range lines {
			if width.VisibleWidth(line) > m.width {
				lines[i] = width.TruncateToWidth(line, m.width)
			}
		}
		return strings.Join(lines, "\n")
	}
	return result
}
