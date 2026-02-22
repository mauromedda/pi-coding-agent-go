// ABOUTME: ToolCall component for rendering tool calls inline with assistant messages
// ABOUTME: Matches Claude Code's "Code Generation" style with expand/collapse support

package components

import (
	"fmt"
	"strings"
	"sync"

	"github.com/mauromedda/pi-coding-agent-go/pkg/tui"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/theme"
)

// toolColor returns the semantic theme color for a tool name header.
func toolColor(name string) theme.Color {
	p := theme.Current().Palette
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "read") || strings.Contains(lower, "glob") || strings.Contains(lower, "grep"):
		return p.ToolRead
	case strings.Contains(lower, "bash") || strings.Contains(lower, "exec") || strings.Contains(lower, "shell"):
		return p.ToolBash
	case strings.Contains(lower, "write") || strings.Contains(lower, "edit"):
		return p.ToolWrite
	default:
		return p.ToolOther
	}
}

// ToolCall renders a tool call in the assistant message
type ToolCall struct {
	mu      sync.Mutex
	name    string
	args    string
	done    bool
	err     string
	output  strings.Builder
	expanded bool
}

// NewToolCall creates a ToolCall component for inline rendering
func NewToolCall(name, args string) *ToolCall {
	return &ToolCall{name: name, args: args}
}

// SetDone marks the tool execution as complete with full output
func (t *ToolCall) SetDone(output, errMsg string) {
	t.mu.Lock()
	t.done = true
	t.err = errMsg
	t.output.WriteString(output)
	t.mu.Unlock()
}

// ToggleExpand toggles the expanded state
func (t *ToolCall) ToggleExpand() {
	t.mu.Lock()
	t.expanded = !t.expanded
	t.mu.Unlock()
}

// SetExpanded sets the expanded state
func (t *ToolCall) SetExpanded(expanded bool) {
	t.mu.Lock()
	t.expanded = expanded
	t.mu.Unlock()
}

// IsExpanded returns whether the tool is expanded
func (t *ToolCall) IsExpanded() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.expanded
}

// Render draws the tool call with Claude-style border
func (t *ToolCall) Render(out *tui.RenderBuffer, w int) {
	t.mu.Lock()
	name := t.name
	args := t.args
	done := t.done
	errMsg := t.err
	output := t.output.String()
	expanded := t.expanded
	t.mu.Unlock()

	p := theme.Current().Palette
	nameColor := toolColor(name)

	// Status indicator
	var status string
	if done {
		if errMsg != "" {
			status = "✗"
		} else {
			status = "✓"
		}
	} else {
		spinner := SpinnerFrame()
		status = string(spinner)
	}

	// Tool info line
	toolInfo := fmt.Sprintf("%s %s %s", status, nameColor.Apply(name), args)
	toolInfo = strings.TrimSpace(toolInfo)

	// Claude-style border
	const borderChar = "─"
	const cornerTopLeft = "┌"
	const cornerTopRight = "┐"
	const cornerBottomLeft = "└"
	const cornerBottomRight = "┘"
	const verticalBorder = "│"

	header := " Code Generation "
	headerLen := len(header)
	totalWidth := w - 2
	availableForDashes := totalWidth - headerLen - 2
	dashesLeft := availableForDashes / 2
	dashesRight := availableForDashes - dashesLeft

	topBorder := cornerTopLeft
	if dashesLeft > 0 {
		topBorder += strings.Repeat(borderChar, dashesLeft)
	}
	topBorder += header
	if dashesRight > 0 {
		topBorder += strings.Repeat(borderChar, dashesRight)
	}
	topBorder += cornerTopRight

	out.WriteLine(topBorder)
	contentWidth := w - 2 - 4
	if contentWidth < 20 {
		contentWidth = 20
	}
	out.WriteLine(fmt.Sprintf("%s %-*s %s", verticalBorder, contentWidth, toolInfo, verticalBorder))

	// Add error if present
	if errMsg != "" {
		out.WriteLine(fmt.Sprintf("%s %-*s %s", verticalBorder, contentWidth, p.Error.Apply(errMsg), verticalBorder))
	}

	// Add output if expanded
	if expanded && output != "" {
		// Add separator between tool info and output
		out.WriteLine(fmt.Sprintf("%s %s", verticalBorder, strings.Repeat("─", w-4)))
		
		// Format output with proper indentation
		lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
		for _, line := range lines {
			out.WriteLine(fmt.Sprintf("%s %s", verticalBorder, line))
		}
		
		// Empty line before bottom border
		out.WriteLine(fmt.Sprintf("%s %s", verticalBorder, ""))
	}

	// Bottom border
	bottomBorder := cornerBottomLeft
	bottomBorder += strings.Repeat(borderChar, totalWidth-len(cornerBottomLeft)-len(cornerBottomRight))
	bottomBorder += cornerBottomRight
	out.WriteLine(bottomBorder)

	// Expand/collapse indicator
	if expanded {
		out.WriteLine(p.Dim.Apply("  Press Ctrl+O to collapse"))
	} else {
		out.WriteLine(p.Dim.Apply("  Press Ctrl+O to expand output"))
	}
	out.WriteLine("")
}

// Invalidate marks for re-render
func (t *ToolCall) Invalidate() {}
