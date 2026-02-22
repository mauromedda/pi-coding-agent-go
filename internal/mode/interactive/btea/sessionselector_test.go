// ABOUTME: Tests for SessionSelectorModel overlay: session selection list
// ABOUTME: Verifies navigation, enter selects, esc dismisses, empty list message

package btea

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// Compile-time checks.
var (
	_ tea.Model = SessionSelectorModel{}
	_ tea.Msg   = SessionSelectedMsg{}
	_ tea.Msg   = SessionSelectorDismissMsg{}
)

func testSessions() []SessionEntry {
	return []SessionEntry{
		{ID: "sess-001", Model: "opus", CWD: "/home/user/project-a"},
		{ID: "sess-002", Model: "sonnet", CWD: "/home/user/project-b"},
		{ID: "sess-003", Model: "haiku", CWD: "/tmp"},
	}
}

func TestSessionSelectorModel_Init(t *testing.T) {
	m := NewSessionSelectorModel(testSessions())
	if cmd := m.Init(); cmd != nil {
		t.Errorf("Init() returned non-nil cmd")
	}
}

func TestSessionSelectorModel_Navigation(t *testing.T) {
	m := NewSessionSelectorModel(testSessions())

	tests := []struct {
		name   string
		key    tea.KeyType
		wantID string
	}{
		{"down to sess-002", tea.KeyDown, "sess-002"},
		{"down to sess-003", tea.KeyDown, "sess-003"},
		{"down at bottom stays", tea.KeyDown, "sess-003"},
		{"up to sess-002", tea.KeyUp, "sess-002"},
		{"up to sess-001", tea.KeyUp, "sess-001"},
		{"up at top stays", tea.KeyUp, "sess-001"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updated, _ := m.Update(tea.KeyMsg{Type: tt.key})
			m = updated.(SessionSelectorModel)
			if m.sessions[m.selected].ID != tt.wantID {
				t.Errorf("selected session ID = %q; want %q", m.sessions[m.selected].ID, tt.wantID)
			}
		})
	}
}

func TestSessionSelectorModel_EnterSelectsSession(t *testing.T) {
	m := NewSessionSelectorModel(testSessions())

	// Move to sess-002
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(SessionSelectorModel)

	// Press enter
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("Update(enter) returned nil cmd; want SessionSelectedMsg")
	}
	msg := cmd()
	sel, ok := msg.(SessionSelectedMsg)
	if !ok {
		t.Fatalf("cmd() returned %T; want SessionSelectedMsg", msg)
	}
	if sel.Session.ID != "sess-002" {
		t.Errorf("SessionSelectedMsg.Session.ID = %q; want %q", sel.Session.ID, "sess-002")
	}
	if sel.Session.Model != "sonnet" {
		t.Errorf("SessionSelectedMsg.Session.Model = %q; want %q", sel.Session.Model, "sonnet")
	}
	if sel.Session.CWD != "/home/user/project-b" {
		t.Errorf("SessionSelectedMsg.Session.CWD = %q; want %q", sel.Session.CWD, "/home/user/project-b")
	}
}

func TestSessionSelectorModel_EscDismisses(t *testing.T) {
	m := NewSessionSelectorModel(testSessions())

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("Update(esc) returned nil cmd; want SessionSelectorDismissMsg")
	}
	msg := cmd()
	if _, ok := msg.(SessionSelectorDismissMsg); !ok {
		t.Errorf("cmd() returned %T; want SessionSelectorDismissMsg", msg)
	}
}

func TestSessionSelectorModel_ViewContainsHeader(t *testing.T) {
	m := NewSessionSelectorModel(testSessions())
	m.width = 80
	view := m.View()

	if !strings.Contains(view, "Resume Session") {
		t.Error("View() missing 'Resume Session' header")
	}
}

func TestSessionSelectorModel_ViewShowsSessions(t *testing.T) {
	m := NewSessionSelectorModel(testSessions())
	m.width = 80
	view := m.View()

	for _, sess := range testSessions() {
		if !strings.Contains(view, sess.ID) {
			t.Errorf("View() missing session ID %q", sess.ID)
		}
		if !strings.Contains(view, sess.Model) {
			t.Errorf("View() missing session model %q", sess.Model)
		}
	}
}

func TestSessionSelectorModel_ViewShowsCWD(t *testing.T) {
	m := NewSessionSelectorModel(testSessions())
	m.width = 80
	view := m.View()

	if !strings.Contains(view, "/home/user/project-a") {
		t.Error("View() missing CWD '/home/user/project-a'")
	}
}

func TestSessionSelectorModel_ViewSelectionIndicator(t *testing.T) {
	m := NewSessionSelectorModel(testSessions())
	m.width = 80
	view := m.View()

	if !strings.Contains(view, "> ") {
		t.Error("View() missing selection indicator '> '")
	}
}

func TestSessionSelectorModel_ViewEmptyList(t *testing.T) {
	m := NewSessionSelectorModel(nil)
	m.width = 80
	view := m.View()

	if !strings.Contains(view, "Resume Session") {
		t.Error("View() on empty list missing 'Resume Session' header")
	}
	if !strings.Contains(view, "No sessions") {
		t.Error("View() on empty list missing 'No sessions' message")
	}
}

func TestSessionSelectorModel_EnterEmptyList(t *testing.T) {
	m := NewSessionSelectorModel(nil)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Errorf("Update(enter) on empty list returned non-nil cmd; want nil")
	}
}
