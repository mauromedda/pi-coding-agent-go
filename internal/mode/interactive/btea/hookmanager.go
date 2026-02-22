// ABOUTME: HookManagerModel is a Bubble Tea overlay for managing tool hooks
// ABOUTME: Supports navigation, toggle enabled/disabled, and remove; value semantics

package btea

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Hook represents a single tool hook configuration.
type Hook struct {
	Pattern string
	Enabled bool
	Tools   []string
	Event   string
}

// HookManagerModel displays and manages a list of hooks.
// Implements tea.Model with value semantics.
type HookManagerModel struct {
	hooks     []Hook
	selected  int
	scrollOff int
	maxHeight int
	width     int
}

// NewHookManagerModel creates an empty HookManagerModel.
func NewHookManagerModel() HookManagerModel {
	return HookManagerModel{
		maxHeight: 100,
	}
}

// Init returns nil; no commands needed at startup.
func (m HookManagerModel) Init() tea.Cmd {
	return nil
}

// Update handles key messages for navigation, toggle, and remove.
func (m HookManagerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyUp:
			m.moveUp()
		case tea.KeyDown:
			m.moveDown()
		case tea.KeyEnter:
			m.toggle()
		case tea.KeyDelete:
			m.remove()
		case tea.KeyRunes:
			if len(msg.Runes) > 0 && msg.Runes[0] == 'd' {
				m.remove()
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
	}
	return m, nil
}

// View renders the hook list with enabled/disabled status indicators.
func (m HookManagerModel) View() string {
	if len(m.hooks) == 0 {
		return ""
	}

	s := Styles()
	end := min(m.scrollOff+m.maxHeight, len(m.hooks))
	var b strings.Builder

	for i := m.scrollOff; i < end; i++ {
		hook := m.hooks[i]
		if i > m.scrollOff {
			b.WriteByte('\n')
		}

		line := formatHookLine(s, hook)
		if i == m.selected {
			line = s.Bold.Render(s.Selection.Render(line))
		}
		b.WriteString(line)
	}

	return b.String()
}

// SetHooks replaces the hook list and resets selection. Returns a new model.
func (m HookManagerModel) SetHooks(hooks []Hook) HookManagerModel {
	m.hooks = make([]Hook, len(hooks))
	copy(m.hooks, hooks)
	m.selected = 0
	m.scrollOff = 0
	return m
}

// ToggleHook toggles the enabled state of the selected hook. Returns a new model.
func (m HookManagerModel) ToggleHook() HookManagerModel {
	if len(m.hooks) == 0 {
		return m
	}
	m.hooks[m.selected].Enabled = !m.hooks[m.selected].Enabled
	return m
}

// RemoveHook removes the selected hook from the list. Returns a new model.
func (m HookManagerModel) RemoveHook() HookManagerModel {
	if len(m.hooks) == 0 {
		return m
	}
	m.hooks = append(m.hooks[:m.selected], m.hooks[m.selected+1:]...)
	if m.selected >= len(m.hooks) && m.selected > 0 {
		m.selected--
	}
	m.adjustScroll()
	return m
}

// SelectedHook returns the currently selected hook.
// Returns a zero-value Hook if the list is empty.
func (m HookManagerModel) SelectedHook() Hook {
	if len(m.hooks) == 0 {
		return Hook{}
	}
	return m.hooks[m.selected]
}

// Count returns the number of hooks.
func (m HookManagerModel) Count() int {
	return len(m.hooks)
}

// Reset clears all hooks and selection. Returns a new model.
func (m HookManagerModel) Reset() HookManagerModel {
	m.hooks = nil
	m.selected = 0
	m.scrollOff = 0
	return m
}

// --- Internal helpers ---

func (m *HookManagerModel) moveUp() {
	if m.selected > 0 {
		m.selected--
		m.adjustScroll()
	}
}

func (m *HookManagerModel) moveDown() {
	if m.selected < len(m.hooks)-1 {
		m.selected++
		m.adjustScroll()
	}
}

func (m *HookManagerModel) toggle() {
	if len(m.hooks) > 0 {
		m.hooks[m.selected].Enabled = !m.hooks[m.selected].Enabled
	}
}

func (m *HookManagerModel) remove() {
	if len(m.hooks) == 0 {
		return
	}
	m.hooks = append(m.hooks[:m.selected], m.hooks[m.selected+1:]...)
	if m.selected >= len(m.hooks) && m.selected > 0 {
		m.selected--
	}
	m.adjustScroll()
}

func (m *HookManagerModel) adjustScroll() {
	if m.selected < m.scrollOff {
		m.scrollOff = m.selected
	}
	if m.selected >= m.scrollOff+m.maxHeight {
		m.scrollOff = m.selected - m.maxHeight + 1
	}
}

func formatHookLine(s ThemeStyles, hook Hook) string {
	var status string
	if hook.Enabled {
		status = s.Success.Render("[ON] ")
	} else {
		status = s.Error.Render("[OFF]")
	}

	tools := ""
	if len(hook.Tools) > 0 {
		tools = fmt.Sprintf("  tools: %s", strings.Join(hook.Tools, ", "))
	}

	return fmt.Sprintf("  %s %s  %s%s", status, hook.Pattern, s.Muted.Render(hook.Event), tools)
}
