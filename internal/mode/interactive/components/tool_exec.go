// ABOUTME: Tool execution progress display component
// ABOUTME: Shows tool name, arguments, and completion status

package components

import (
	"fmt"
	"strings"
	"sync"

	"github.com/mauromedda/pi-coding-agent-go/pkg/tui"
)

// ToolExec renders tool execution progress.
type ToolExec struct {
	mu     sync.Mutex
	name   string
	args   string
	output strings.Builder
	done   bool
	err    string
}

// NewToolExec creates a ToolExec for the given tool.
func NewToolExec(name, args string) *ToolExec {
	return &ToolExec{name: name, args: args}
}

// AppendOutput adds streaming output.
func (t *ToolExec) AppendOutput(output string) {
	t.mu.Lock()
	t.output.WriteString(output)
	t.mu.Unlock()
}

// SetDone marks execution as complete and releases accumulated output memory.
func (t *ToolExec) SetDone(errMsg string) {
	t.mu.Lock()
	t.done = true
	t.err = errMsg
	// Release output buffer: it's not rendered and no longer needed
	t.output.Reset()
	t.mu.Unlock()
}

// Render draws the tool execution status with a blank spacer above.
func (t *ToolExec) Render(out *tui.RenderBuffer, _ int) {
	t.mu.Lock()
	out.WriteLine("")

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
	t.mu.Unlock()
}

// Invalidate is a no-op for ToolExec.
func (t *ToolExec) Invalidate() {}
