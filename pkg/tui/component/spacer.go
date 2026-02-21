// ABOUTME: Vertical spacer component that renders empty lines
// ABOUTME: Used for layout spacing between other components

package component

import "github.com/mauromedda/pi-coding-agent-go/pkg/tui"

// Spacer renders a fixed number of empty lines.
type Spacer struct {
	Height int
}

// NewSpacer creates a spacer with the given height in lines.
func NewSpacer(height int) *Spacer {
	return &Spacer{Height: height}
}

// Render writes empty lines into the buffer.
func (s *Spacer) Render(out *tui.RenderBuffer, _ int) {
	for i := 0; i < s.Height; i++ {
		out.WriteLine("")
	}
}

// Invalidate is a no-op for Spacer.
func (s *Spacer) Invalidate() {}
