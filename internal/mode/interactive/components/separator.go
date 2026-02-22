// ABOUTME: Separator renders a dim horizontal line spanning the given width
// ABOUTME: Implements tui.Component; used to visually divide sections in the TUI

package components

import (
	"strings"

	"github.com/mauromedda/pi-coding-agent-go/pkg/tui"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/theme"
)

// Separator is a thin horizontal rule rendered as dim box-drawing characters.
type Separator struct{}

// NewSeparator creates a new Separator component.
func NewSeparator() *Separator { return &Separator{} }

// Render writes a single dim horizontal line of width w into out.
func (s *Separator) Render(out *tui.RenderBuffer, w int) {
	if w <= 0 {
		out.WriteLine("")
		return
	}
	p := theme.Current().Palette
	out.WriteLine(p.Border.Apply(strings.Repeat("─", w)))
}

// Invalidate is a no-op; Separator has no cached state.
func (s *Separator) Invalidate() {}
