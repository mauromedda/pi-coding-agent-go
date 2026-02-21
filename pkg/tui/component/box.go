// ABOUTME: Box component that wraps a child with padding and optional background
// ABOUTME: Renders child content with configurable horizontal and vertical padding

package component

import (
	"strings"

	"github.com/mauromedda/pi-coding-agent-go/pkg/tui"
)

// Box wraps a child component with padding.
type Box struct {
	Child   tui.Component
	PadTop  int
	PadBot  int
	PadLeft int
	PadRight int
	BgColor string // ANSI color code, e.g., "\x1b[44m" for blue background
}

// NewBox creates a Box around the given child component.
func NewBox(child tui.Component) *Box {
	return &Box{Child: child}
}

// WithPadding sets uniform padding on all sides.
func (b *Box) WithPadding(pad int) *Box {
	b.PadTop = pad
	b.PadBot = pad
	b.PadLeft = pad
	b.PadRight = pad
	return b
}

// Render draws the child with padding applied.
func (b *Box) Render(out *tui.RenderBuffer, width int) {
	innerWidth := width - b.PadLeft - b.PadRight
	if innerWidth <= 0 {
		return
	}

	leftPad := strings.Repeat(" ", b.PadLeft)

	// Top padding
	for i := 0; i < b.PadTop; i++ {
		out.WriteLine(b.bgLine(width))
	}

	// Render child
	childBuf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(childBuf)

	b.Child.Render(childBuf, innerWidth)

	for _, line := range childBuf.Lines {
		padded := leftPad + line
		if b.BgColor != "" {
			padded = b.BgColor + padded + "\x1b[0m"
		}
		out.WriteLine(padded)
	}

	// Bottom padding
	for i := 0; i < b.PadBot; i++ {
		out.WriteLine(b.bgLine(width))
	}
}

// Invalidate invalidates the child.
func (b *Box) Invalidate() {
	if b.Child != nil {
		b.Child.Invalidate()
	}
}

func (b *Box) bgLine(width int) string {
	line := strings.Repeat(" ", width)
	if b.BgColor != "" {
		return b.BgColor + line + "\x1b[0m"
	}
	return line
}
