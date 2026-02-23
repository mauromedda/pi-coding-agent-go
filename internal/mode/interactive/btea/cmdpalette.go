// ABOUTME: CmdPaletteModel is a Bubble Tea leaf for slash-command autocomplete
// ABOUTME: Port of components/command_palette.go; case-insensitive filter, wrapping nav

package btea

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/width"
)

const maxCmdPaletteVisible = 10

// CommandEntry describes a single slash command for the palette.
type CommandEntry struct {
	Name        string
	Description string
}

// CmdPaletteSelectMsg is returned when the user presses enter on a command.
type CmdPaletteSelectMsg struct{ Name string }

// CmdPaletteDismissMsg is returned when the user presses escape.
type CmdPaletteDismissMsg struct{}

// CmdPaletteModel is a filterable overlay listing available slash commands.
// Implements tea.Model with value semantics.
type CmdPaletteModel struct {
	commands []CommandEntry
	visible  []CommandEntry
	selected int
	filter   string
	width    int
}

// NewCmdPaletteModel creates a palette pre-populated with the given commands.
func NewCmdPaletteModel(cmds []CommandEntry) CmdPaletteModel {
	m := CmdPaletteModel{
		commands: cmds,
	}
	m.applyFilter()
	return m
}

// Init returns nil; no commands needed at startup.
func (m CmdPaletteModel) Init() tea.Cmd {
	return nil
}

// Update handles key and window-size messages.
func (m CmdPaletteModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyRunes:
			// Typing characters filters the palette
			if len(msg.Runes) > 0 {
				m.filter += string(msg.Runes)
				m.selected = 0
				m.applyFilter()
			}
		case tea.KeyBackspace, tea.KeyLeft:
			// Delete last character from filter
			if len(m.filter) > 0 {
				m.filter = m.filter[:len(m.filter)-1]
				m.selected = 0
				m.applyFilter()
			}
		case tea.KeyUp:
			m.moveUp()
		case tea.KeyDown:
			m.moveDown()
		case tea.KeyEnter:
			name := m.Selected()
			if name != "" {
				return m, func() tea.Msg { return CmdPaletteSelectMsg{Name: name} }
			}
		case tea.KeyEsc:
			return m, func() tea.Msg { return CmdPaletteDismissMsg{} }
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
	}
	return m, nil
}

// View renders the command list capped at maxCmdPaletteVisible.
func (m CmdPaletteModel) View() string {
	total := len(m.visible)
	if total == 0 {
		return ""
	}

	// Compute viewport window around selected item
	start := 0
	end := total
	if total > maxCmdPaletteVisible {
		start = m.selected - maxCmdPaletteVisible/2
		if start < 0 {
			start = 0
		}
		end = start + maxCmdPaletteVisible
		if end > total {
			end = total
			start = end - maxCmdPaletteVisible
		}
	}

	s := Styles()
	var b strings.Builder

	for i := start; i < end; i++ {
		entry := m.visible[i]
		name := fmt.Sprintf("/%s", entry.Name)

		line := fmt.Sprintf("  %-16s %s", name, entry.Description)
		if m.width > 0 {
			line = width.TruncateToWidth(line, m.width)
		}

		if i == m.selected {
			line = s.Bold.Render(s.Selection.Render(line))
		} else {
			line = s.Dim.Render(line)
		}

		if i > start {
			b.WriteByte('\n')
		}
		b.WriteString(line)
	}

	return b.String()
}

// SetFilter updates the filter string and resets selection. Returns a new model.
func (m CmdPaletteModel) SetFilter(f string) CmdPaletteModel {
	m.filter = f
	m.selected = 0
	m.applyFilter()
	return m
}

// Selected returns the Name of the currently highlighted command.
func (m CmdPaletteModel) Selected() string {
	if len(m.visible) == 0 {
		return ""
	}
	return m.visible[m.selected].Name
}

func (m *CmdPaletteModel) moveDown() {
	if len(m.visible) == 0 {
		return
	}
	m.selected = (m.selected + 1) % len(m.visible)
}

func (m *CmdPaletteModel) moveUp() {
	if len(m.visible) == 0 {
		return
	}
	m.selected = (m.selected - 1 + len(m.visible)) % len(m.visible)
}

func (m *CmdPaletteModel) applyFilter() {
	if m.filter == "" {
		m.visible = make([]CommandEntry, len(m.commands))
		copy(m.visible, m.commands)
		return
	}

	lower := strings.ToLower(m.filter)
	m.visible = make([]CommandEntry, 0, len(m.commands))
	for _, cmd := range m.commands {
		if strings.Contains(strings.ToLower(cmd.Name), lower) {
			m.visible = append(m.visible, cmd)
		}
	}
}
