// ABOUTME: Tests for UserMsgModel Bubble Tea leaf component
// ABOUTME: Verifies tea.Model interface, View output with prompt prefix and user text

package btea

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// Compile-time check: UserMsgModel must satisfy tea.Model.
var _ tea.Model = UserMsgModel{}

func TestUserMsgModel_Init(t *testing.T) {
	m := NewUserMsgModel("hello")
	if cmd := m.Init(); cmd != nil {
		t.Errorf("Init() returned non-nil cmd")
	}
}

func TestUserMsgModel_View(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		wantText string
		wantGt   bool
	}{
		{"simple text", "hello world", "hello world", true},
		{"empty text", "", "", true},
		{"with special chars", "foo & bar", "foo & bar", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewUserMsgModel(tt.text)
			view := m.View()

			if tt.wantText != "" && !strings.Contains(view, tt.wantText) {
				t.Errorf("View() missing text %q; got %q", tt.wantText, view)
			}

			if tt.wantGt && !strings.Contains(view, ">") {
				t.Errorf("View() missing '>' prompt prefix; got %q", view)
			}
		})
	}
}

func TestUserMsgModel_ViewContainsBlankLine(t *testing.T) {
	m := NewUserMsgModel("test")
	view := m.View()
	if !strings.HasPrefix(view, "\n") {
		t.Errorf("View() should start with blank line; got %q", view[:min(20, len(view))])
	}
}

func TestUserMsgModel_WindowSizeMsg(t *testing.T) {
	m := NewUserMsgModel("test")
	updated, cmd := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	if cmd != nil {
		t.Errorf("Update(WindowSizeMsg) returned non-nil cmd")
	}
	u := updated.(UserMsgModel)
	if u.width != 80 {
		t.Errorf("width = %d; want 80", u.width)
	}
}
