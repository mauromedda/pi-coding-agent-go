// ABOUTME: Status bar component showing model, mode, costs, hints
// ABOUTME: Renders as a single line at the bottom of the TUI

package components

import (
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui"
)

// Footer displays status information at the bottom of the screen.
type Footer struct {
	content string
}

// NewFooter creates a Footer component.
func NewFooter() *Footer {
	return &Footer{}
}

// SetContent updates the footer text.
func (f *Footer) SetContent(content string) {
	f.content = content
}

// Content returns the current footer text.
func (f *Footer) Content() string {
	return f.content
}

// Render writes the footer line with styling.
func (f *Footer) Render(out *tui.RenderBuffer, width int) {
	// Dim style for footer
	line := "\x1b[2m" + f.content + "\x1b[0m"
	out.WriteLine(line)
}

// Invalidate is a no-op for Footer.
func (f *Footer) Invalidate() {}
