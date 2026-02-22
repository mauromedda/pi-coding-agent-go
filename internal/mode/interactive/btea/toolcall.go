// ABOUTME: ToolCallModel is a Bubble Tea leaf that renders a tool invocation box
// ABOUTME: Port of components/tool_call.go; handles streaming updates, expand/collapse

package btea

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// toolColor returns a lipgloss.Style for the tool name header based on tool type.
func toolColor(name string) lipgloss.Style {
	s := Styles()
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "read") || strings.Contains(lower, "glob") || strings.Contains(lower, "grep"):
		return s.ToolRead
	case strings.Contains(lower, "bash") || strings.Contains(lower, "exec") || strings.Contains(lower, "shell"):
		return s.ToolBash
	case strings.Contains(lower, "write") || strings.Contains(lower, "edit"):
		return s.ToolWrite
	default:
		return s.ToolOther
	}
}

// ToolCallModel renders a tool invocation with Claude-style bordered box,
// status indicator, and expand/collapse support for output.
type ToolCallModel struct {
	id       string
	name     string
	args     string
	done     bool
	errMsg   string
	output   string
	expanded bool
	width    int
}

// NewToolCallModel creates a ToolCallModel for the given tool invocation.
func NewToolCallModel(id, name, args string) ToolCallModel {
	return ToolCallModel{
		id:   id,
		name: name,
		args: args,
	}
}

// Init returns nil; no commands needed for a leaf model.
func (m ToolCallModel) Init() tea.Cmd {
	return nil
}

// Update handles messages relevant to this tool call.
func (m ToolCallModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case AgentToolUpdateMsg:
		if msg.ToolID == m.id {
			m.output += msg.Text
		}

	case AgentToolEndMsg:
		if msg.ToolID == m.id {
			m.done = true
			m.output = msg.Text
			if msg.Result != nil && msg.Result.IsError {
				m.errMsg = msg.Result.Content
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width

	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlO && m.done {
			m.expanded = !m.expanded
		}
	}

	return m, nil
}

// View renders the tool call as a bordered box with status indicator.
func (m ToolCallModel) View() string {
	if m.width <= 0 {
		return ""
	}

	s := Styles()
	nameStyle := toolColor(m.name)

	// Status indicator
	var status string
	switch {
	case m.done && m.errMsg != "":
		status = "✗"
	case m.done:
		status = "✓"
	default:
		status = "⠋"
	}

	// Tool info line
	toolInfo := fmt.Sprintf("%s %s %s", status, nameStyle.Render(m.name), m.args)
	toolInfo = strings.TrimSpace(toolInfo)

	// Border characters
	const (
		borderChar        = "─"
		cornerTopLeft     = "┌"
		cornerTopRight    = "┐"
		cornerBottomLeft  = "└"
		cornerBottomRight = "┘"
		verticalBorder    = "│"
	)

	// Build top border with header
	header := " Code Generation "
	headerLen := len(header)
	totalWidth := m.width - 2
	if totalWidth < 0 {
		totalWidth = 0
	}
	availableForDashes := totalWidth - headerLen - 2
	if availableForDashes < 0 {
		availableForDashes = 0
	}
	dashesLeft := availableForDashes / 2
	dashesRight := availableForDashes - dashesLeft

	var b strings.Builder

	// Top border
	b.WriteString(cornerTopLeft)
	if dashesLeft > 0 {
		b.WriteString(strings.Repeat(borderChar, dashesLeft))
	}
	b.WriteString(header)
	if dashesRight > 0 {
		b.WriteString(strings.Repeat(borderChar, dashesRight))
	}
	b.WriteString(cornerTopRight)
	b.WriteString("\n")

	// Content width
	contentWidth := m.width - 4
	if contentWidth < 20 {
		contentWidth = 20
	}

	// Tool info line
	b.WriteString(fmt.Sprintf("%s %-*s %s\n", verticalBorder, contentWidth, toolInfo, verticalBorder))

	// Error line
	if m.errMsg != "" {
		b.WriteString(fmt.Sprintf("%s %-*s %s\n", verticalBorder, contentWidth, s.Error.Render(m.errMsg), verticalBorder))
	}

	// Expanded output
	if m.expanded && m.output != "" {
		b.WriteString(fmt.Sprintf("%s %s\n", verticalBorder, strings.Repeat(borderChar, m.width-4)))

		lines := strings.Split(strings.TrimRight(m.output, "\n"), "\n")
		for _, line := range lines {
			b.WriteString(fmt.Sprintf("%s %s\n", verticalBorder, line))
		}

		b.WriteString(fmt.Sprintf("%s %s\n", verticalBorder, ""))
	}

	// Bottom border
	bottomWidth := totalWidth - len(cornerBottomLeft) - len(cornerBottomRight)
	if bottomWidth < 0 {
		bottomWidth = 0
	}
	b.WriteString(cornerBottomLeft)
	b.WriteString(strings.Repeat(borderChar, bottomWidth))
	b.WriteString(cornerBottomRight)
	b.WriteString("\n")

	// Expand/collapse hint
	if m.expanded {
		b.WriteString(s.Dim.Render("  Press Ctrl+O to collapse"))
	} else {
		b.WriteString(s.Dim.Render("  Press Ctrl+O to expand output"))
	}

	return b.String()
}
