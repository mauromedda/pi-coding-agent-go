// ABOUTME: Welcome banner component with ASCII π logo, version, model, shortcuts
// ABOUTME: Implements tui.Component; renders branded startup screen at session start

package components

import (
	"fmt"

	"github.com/mauromedda/pi-coding-agent-go/pkg/tui"
)

// WelcomeMessage renders the startup banner with version info, keyboard
// shortcuts, and registered tool count.
type WelcomeMessage struct {
	version   string
	modelName string
	cwd       string
	toolCount int
}

// NewWelcomeMessage creates a WelcomeMessage component.
func NewWelcomeMessage(version, modelName, cwd string, toolCount int) *WelcomeMessage {
	return &WelcomeMessage{
		version:   version,
		modelName: modelName,
		cwd:       cwd,
		toolCount: toolCount,
	}
}

// Render writes the welcome banner into the buffer.
func (w *WelcomeMessage) Render(out *tui.RenderBuffer, _ int) {
	ver := w.version
	if ver == "" {
		ver = "dev"
	}

	// ASCII π logo (orange/warm accent)
	out.WriteLine("\x1b[38;5;208m  ╭───────╮\x1b[0m")
	out.WriteLine("\x1b[38;5;208m  │  \x1b[1mπ\x1b[0m\x1b[38;5;208m    │\x1b[0m")
	out.WriteLine("\x1b[38;5;208m  ╰───────╯\x1b[0m")

	// Version, model, cwd
	out.WriteLine(fmt.Sprintf("  \x1b[1mpi-go\x1b[0m \x1b[2mv%s\x1b[0m", ver))
	out.WriteLine(fmt.Sprintf("  \x1b[2m%s\x1b[0m", w.modelName))
	out.WriteLine(fmt.Sprintf("  \x1b[36m%s\x1b[0m", w.cwd))

	// Blank separator
	out.WriteLine("")

	// Keyboard shortcuts: two-column layout with padded keys
	shortcuts := []struct {
		key  string
		desc string
	}{
		{"escape", "interrupt"},
		{"ctrl+c", "clear"},
		{"ctrl+c twice", "exit"},
		{"ctrl+d", "exit (empty)"},
		{"shift+tab", "cycle mode"},
		{"/", "commands"},
		{"!", "run bash"},
	}

	const keyPad = 16 // pad key column to 16 chars
	for _, s := range shortcuts {
		padded := s.key
		for len(padded) < keyPad {
			padded += " "
		}
		out.WriteLine(fmt.Sprintf("  \x1b[1m%s\x1b[0m\x1b[2m%s\x1b[0m", padded, s.desc))
	}

	// Blank separator
	out.WriteLine("")

	// Tool count (dim)
	out.WriteLine(fmt.Sprintf("\x1b[2m  [Tools: %d registered]\x1b[0m", w.toolCount))
}

// Invalidate is a no-op for WelcomeMessage.
func (w *WelcomeMessage) Invalidate() {}
