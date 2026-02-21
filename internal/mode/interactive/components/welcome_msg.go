// ABOUTME: Welcome banner component showing version, model, cwd, shortcuts, and tool count
// ABOUTME: Implements tui.Component; renders styled intro screen at session start

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

	// Banner
	out.WriteLine("\x1b[1m    \u03c0\x1b[0m")

	// Version, model, cwd (dim)
	out.WriteLine("\x1b[2m  pi-go v" + ver + "\x1b[0m")
	out.WriteLine("\x1b[2m  " + w.modelName + "\x1b[0m")
	out.WriteLine("\x1b[2m  " + w.cwd + "\x1b[0m")

	// Blank separator
	out.WriteLine("")

	// Keyboard shortcuts: bold key + dim description
	shortcuts := []struct {
		key  string
		desc string
	}{
		{"escape", "to interrupt"},
		{"ctrl+c", "to clear"},
		{"ctrl+c twice", "to exit"},
		{"ctrl+d", "to exit (empty)"},
		{"shift+tab", "to cycle mode"},
		{"/", "for commands"},
		{"!", "to run bash"},
	}

	for _, s := range shortcuts {
		out.WriteLine("  \x1b[1m" + s.key + "\x1b[0m\x1b[2m " + s.desc + "\x1b[0m")
	}

	// Blank separator
	out.WriteLine("")

	// Tool count (dim)
	out.WriteLine(fmt.Sprintf("\x1b[2m  [Tools: %d registered]\x1b[0m", w.toolCount))
}

// Invalidate is a no-op for WelcomeMessage.
func (w *WelcomeMessage) Invalidate() {}
