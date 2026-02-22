// ABOUTME: User message display component with background highlight
// ABOUTME: Renders user input with a prompt prefix and subtle background distinction

package components

import (
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/theme"
)

// UserMessage renders a user's message.
type UserMessage struct {
	text string
}

// NewUserMessage creates a UserMessage.
func NewUserMessage(text string) *UserMessage {
	return &UserMessage{text: text}
}

// Render draws the user message with a blank spacer, background highlight,
// and bold prompt prefix for visual distinction from assistant text.
func (u *UserMessage) Render(out *tui.RenderBuffer, _ int) {
	out.WriteLine("")
	// \x1b[100m = bright black background (basic 8-color), compatible with all terminals
	p := theme.Current().Palette
	out.WriteLine(p.UserBg.Code() + p.Prompt.Code() + " > " + u.text + " \x1b[0m")
}

// Invalidate is a no-op for UserMessage.
func (u *UserMessage) Invalidate() {}
