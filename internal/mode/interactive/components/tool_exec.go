// ABOUTME: Tool execution progress display component
// ABOUTME: Shows tool name, arguments, and streaming output

package components

import (
	"fmt"

	"github.com/mauromedda/pi-coding-agent-go/pkg/tui"
)

// ToolExec renders tool execution progress.
type ToolExec struct {
	name   string
	args   string
	output string
	done   bool
	err    string
}

// NewToolExec creates a ToolExec for the given tool.
func NewToolExec(name, args string) *ToolExec {
	return &ToolExec{name: name, args: args}
}

// AppendOutput adds streaming output.
func (t *ToolExec) AppendOutput(output string) {
	t.output += output
}

// SetDone marks execution as complete.
func (t *ToolExec) SetDone(errMsg string) {
	t.done = true
	t.err = errMsg
}

// Render draws the tool execution status.
func (t *ToolExec) Render(out *tui.RenderBuffer, _ int) {
	// Tool name with status indicator
	status := "\x1b[33m⟳\x1b[0m" // Yellow spinner
	if t.done {
		if t.err != "" {
			status = "\x1b[31m✗\x1b[0m" // Red cross
		} else {
			status = "\x1b[32m✓\x1b[0m" // Green check
		}
	}

	out.WriteLine(fmt.Sprintf("%s \x1b[1m%s\x1b[0m \x1b[2m%s\x1b[0m", status, t.name, t.args))

	if t.err != "" {
		out.WriteLine("\x1b[31m  " + t.err + "\x1b[0m")
	}
}

// Invalidate is a no-op for ToolExec.
func (t *ToolExec) Invalidate() {}
