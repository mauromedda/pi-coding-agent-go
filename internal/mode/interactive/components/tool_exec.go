// ABOUTME: Tool execution progress display with colored headers per tool type
// ABOUTME: Shows braille spinner while running, green/red status when done

package components

import (
	"fmt"
	"strings"
	"sync"

	"github.com/mauromedda/pi-coding-agent-go/pkg/tui"
)

// toolColor returns the ANSI color code for a tool name header.
// Read-like tools use green; bash/exec use amber; others use cyan.
func toolColor(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "read") || strings.Contains(lower, "glob") || strings.Contains(lower, "grep"):
		return "\x1b[32m" // green
	case strings.Contains(lower, "bash") || strings.Contains(lower, "exec") || strings.Contains(lower, "shell"):
		return "\x1b[33m" // yellow/amber
	case strings.Contains(lower, "write") || strings.Contains(lower, "edit"):
		return "\x1b[35m" // magenta
	default:
		return "\x1b[36m" // cyan
	}
}

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
// Snapshots state under lock; renders without holding it.
func (t *ToolExec) Render(out *tui.RenderBuffer, _ int) {
	t.mu.Lock()
	name := t.name
	args := t.args
	done := t.done
	errMsg := t.err
	t.mu.Unlock()

	out.WriteLine("")

	nameColor := toolColor(name)

	// Status indicator: braille spinner while running, check/cross when done
	var status string
	if done {
		if errMsg != "" {
			status = "\x1b[31m✗\x1b[0m" // Red cross
		} else {
			status = "\x1b[32m✓\x1b[0m" // Green check
		}
	} else {
		spinner := SpinnerFrame()
		status = fmt.Sprintf("\x1b[33m%c\x1b[0m", spinner) // Animated braille spinner
	}

	out.WriteLine(fmt.Sprintf("%s %s\x1b[1m%s\x1b[0m \x1b[2m%s\x1b[0m", status, nameColor, name, args))

	if errMsg != "" {
		out.WriteLine("\x1b[31m  " + errMsg + "\x1b[0m")
	}
}

// Invalidate is a no-op for ToolExec.
func (t *ToolExec) Invalidate() {}
