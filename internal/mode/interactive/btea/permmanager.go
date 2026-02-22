// ABOUTME: PermManagerModel is a Bubble Tea overlay for managing permission rules
// ABOUTME: Supports navigation, remove, allow/deny color coding; value semantics

package btea

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// RuleEntry represents a single permission rule for display.
type RuleEntry struct {
	Tool  string
	Allow bool
}

// PermManagerModel displays and manages a list of permission rules.
// Implements tea.Model with value semantics.
type PermManagerModel struct {
	rules     []RuleEntry
	selected  int
	scrollOff int
	maxHeight int
	width     int
}

// NewPermManagerModel creates an empty PermManagerModel.
func NewPermManagerModel() PermManagerModel {
	return PermManagerModel{
		maxHeight: 100,
	}
}

// Init returns nil; no commands needed at startup.
func (m PermManagerModel) Init() tea.Cmd {
	return nil
}

// Update handles key messages for navigation and remove.
func (m PermManagerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyUp:
			m.moveUp()
		case tea.KeyDown:
			m.moveDown()
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

// View renders the rule list with allow/deny color coding and a bordered layout.
func (m PermManagerModel) View() string {
	if len(m.rules) == 0 {
		return ""
	}

	s := Styles()
	end := min(m.scrollOff+m.maxHeight, len(m.rules))
	var b strings.Builder

	for i := m.scrollOff; i < end; i++ {
		rule := m.rules[i]
		if i > m.scrollOff {
			b.WriteByte('\n')
		}

		line := formatRuleLine(s, rule)
		if i == m.selected {
			line = s.Bold.Render(s.Selection.Render(line))
		}
		b.WriteString(line)
	}

	return b.String()
}

// SetRules replaces the rule list and resets selection. Returns a new model.
func (m PermManagerModel) SetRules(rules []RuleEntry) PermManagerModel {
	m.rules = make([]RuleEntry, len(rules))
	copy(m.rules, rules)
	m.selected = 0
	m.scrollOff = 0
	return m
}

// RemoveRule removes the selected rule from the list. Returns a new model.
func (m PermManagerModel) RemoveRule() PermManagerModel {
	if len(m.rules) == 0 {
		return m
	}
	m.rules = append(m.rules[:m.selected], m.rules[m.selected+1:]...)
	if m.selected >= len(m.rules) && m.selected > 0 {
		m.selected--
	}
	m.adjustScroll()
	return m
}

// SelectedRule returns the currently selected rule.
// Returns a zero-value RuleEntry if the list is empty.
func (m PermManagerModel) SelectedRule() RuleEntry {
	if len(m.rules) == 0 {
		return RuleEntry{}
	}
	return m.rules[m.selected]
}

// Count returns the number of rules.
func (m PermManagerModel) Count() int {
	return len(m.rules)
}

// Reset clears all rules and selection. Returns a new model.
func (m PermManagerModel) Reset() PermManagerModel {
	m.rules = nil
	m.selected = 0
	m.scrollOff = 0
	return m
}

// --- Internal helpers ---

func (m *PermManagerModel) moveUp() {
	if m.selected > 0 {
		m.selected--
		m.adjustScroll()
	}
}

func (m *PermManagerModel) moveDown() {
	if m.selected < len(m.rules)-1 {
		m.selected++
		m.adjustScroll()
	}
}

func (m *PermManagerModel) remove() {
	if len(m.rules) == 0 {
		return
	}
	m.rules = append(m.rules[:m.selected], m.rules[m.selected+1:]...)
	if m.selected >= len(m.rules) && m.selected > 0 {
		m.selected--
	}
	m.adjustScroll()
}

func (m *PermManagerModel) adjustScroll() {
	if m.selected < m.scrollOff {
		m.scrollOff = m.selected
	}
	if m.selected >= m.scrollOff+m.maxHeight {
		m.scrollOff = m.selected - m.maxHeight + 1
	}
}

func formatRuleLine(s ThemeStyles, rule RuleEntry) string {
	var status string
	if rule.Allow {
		status = s.Success.Render("[ALLOW]")
	} else {
		status = s.Error.Render("[DENY] ")
	}
	return fmt.Sprintf("  %s %s", status, rule.Tool)
}
