// ABOUTME: UserMsgModel is a Bubble Tea leaf that renders a user prompt message
// ABOUTME: Port of components/user_msg.go; uses UserBg style with bold "> " prefix

package btea

import (
	tea "github.com/charmbracelet/bubbletea"
)

// UserMsgModel displays a user's input text with a highlighted background
// and a bold "> " prompt prefix.
type UserMsgModel struct {
	text  string
	width int
}

// NewUserMsgModel creates a UserMsgModel with the given text.
func NewUserMsgModel(text string) UserMsgModel {
	return UserMsgModel{text: text}
}

// Init returns nil; no commands needed for a static message.
func (m UserMsgModel) Init() tea.Cmd {
	return nil
}

// Update handles tea.WindowSizeMsg to track terminal width.
func (m UserMsgModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = ws.Width
	}
	return m, nil
}

// View renders a blank line followed by the user text with UserBg style
// and bold "> " prefix.
func (m UserMsgModel) View() string {
	s := Styles()
	line := s.UserBg.Render(s.Bold.Render(" > ") + m.text + " ")
	return "\n" + line
}
