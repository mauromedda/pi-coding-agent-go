// ABOUTME: SessionSelectorModel is a Bubble Tea overlay for selecting a session to resume
// ABOUTME: Returns SessionSelectedMsg on enter, SessionSelectorDismissMsg on esc

package btea

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// SessionEntry represents a session available for resumption.
type SessionEntry struct {
	ID    string
	Model string
	CWD   string
}

// SessionSelectedMsg is returned when the user selects a session.
type SessionSelectedMsg struct{ Session SessionEntry }

// SessionSelectorDismissMsg is returned when the user dismisses the selector.
type SessionSelectorDismissMsg struct{}

// SessionSelectorModel displays a list of sessions for selection.
// Implements tea.Model with value semantics.
type SessionSelectorModel struct {
	sessions []SessionEntry
	selected int
	width    int
}

// NewSessionSelectorModel creates a SessionSelectorModel with the given sessions.
func NewSessionSelectorModel(sessions []SessionEntry) SessionSelectorModel {
	return SessionSelectorModel{
		sessions: sessions,
	}
}

// Init returns nil; no commands needed at startup.
func (m SessionSelectorModel) Init() tea.Cmd {
	return nil
}

// Update handles key messages for navigation, selection, and dismiss.
func (m SessionSelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyUp:
			m.moveUp()
		case tea.KeyDown:
			m.moveDown()
		case tea.KeyEnter:
			if len(m.sessions) == 0 {
				return m, nil
			}
			selected := m.sessions[m.selected]
			return m, func() tea.Msg { return SessionSelectedMsg{Session: selected} }
		case tea.KeyEsc:
			return m, func() tea.Msg { return SessionSelectorDismissMsg{} }
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
	}
	return m, nil
}

// View renders the session selector with header, session details, and selection indicator.
func (m SessionSelectorModel) View() string {
	s := Styles()
	var b strings.Builder

	// Header
	b.WriteString(s.Bold.Render("Resume Session"))
	b.WriteByte('\n')

	if len(m.sessions) == 0 {
		b.WriteString(s.Muted.Render("  No sessions available"))
		return b.String()
	}

	for i, sess := range m.sessions {
		b.WriteByte('\n')

		var prefix string
		if i == m.selected {
			prefix = "> "
		} else {
			prefix = "  "
		}

		line := fmt.Sprintf("%s%s  %s  %s", prefix, sess.ID, s.Muted.Render(sess.Model), s.Dim.Render(sess.CWD))
		if i == m.selected {
			line = s.Bold.Render(s.Selection.Render(line))
		}
		b.WriteString(line)
	}

	return b.String()
}

// --- Internal helpers ---

func (m *SessionSelectorModel) moveUp() {
	if m.selected > 0 {
		m.selected--
	}
}

func (m *SessionSelectorModel) moveDown() {
	if m.selected < len(m.sessions)-1 {
		m.selected++
	}
}
