// ABOUTME: User message display component
// ABOUTME: Renders user input with a prompt prefix

package components

import "github.com/mauromedda/pi-coding-agent-go/pkg/tui"

// UserMessage renders a user's message.
type UserMessage struct {
	text string
}

// NewUserMessage creates a UserMessage.
func NewUserMessage(text string) *UserMessage {
	return &UserMessage{text: text}
}

// Render draws the user message with a blank spacer and bold prompt prefix.
func (u *UserMessage) Render(out *tui.RenderBuffer, _ int) {
	out.WriteLine("")
	out.WriteLine("\x1b[1m> " + u.text + "\x1b[0m")
}

// Invalidate is a no-op for UserMessage.
func (u *UserMessage) Invalidate() {}
