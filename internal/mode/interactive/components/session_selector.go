// ABOUTME: Session picker overlay for resuming previous sessions
// ABOUTME: Displays a list of stored sessions with timestamps

package components

import (
	"github.com/mauromedda/pi-coding-agent-go/internal/session"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui"
)

// SessionSelector displays a list of sessions to resume.
type SessionSelector struct {
	sessions []session.SessionStartData
	selected int
	onSelect func(session.SessionStartData)
}

// NewSessionSelector creates a session picker.
func NewSessionSelector(sessions []session.SessionStartData, onSelect func(session.SessionStartData)) *SessionSelector {
	return &SessionSelector{
		sessions: sessions,
		onSelect: onSelect,
	}
}

// Render draws the session list.
func (s *SessionSelector) Render(out *tui.RenderBuffer, _ int) {
	out.WriteLine("\x1b[1m  Resume Session  \x1b[0m")
	out.WriteLine("")

	if len(s.sessions) == 0 {
		out.WriteLine("  No previous sessions found.")
		return
	}

	for i, sess := range s.sessions {
		prefix := "  "
		if i == s.selected {
			prefix = "\x1b[7m> "
		}
		line := prefix + sess.ID + " - " + sess.Model + " (" + sess.CWD + ")"
		if i == s.selected {
			line += "\x1b[0m"
		}
		out.WriteLine(line)
	}
}

// Invalidate is a no-op.
func (s *SessionSelector) Invalidate() {}

// MoveUp moves selection up.
func (s *SessionSelector) MoveUp() {
	if s.selected > 0 {
		s.selected--
	}
}

// MoveDown moves selection down.
func (s *SessionSelector) MoveDown() {
	if s.selected < len(s.sessions)-1 {
		s.selected++
	}
}

// Confirm selects the current session.
func (s *SessionSelector) Confirm() {
	if s.selected < len(s.sessions) && s.onSelect != nil {
		s.onSelect(s.sessions[s.selected])
	}
}
