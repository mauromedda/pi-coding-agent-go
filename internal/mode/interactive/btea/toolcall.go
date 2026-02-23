// ABOUTME: ToolCallModel is a Bubble Tea leaf that renders a tool invocation box
// ABOUTME: Port of components/tool_call.go; handles streaming updates, expand/collapse

package btea

import (
	"encoding/json"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// toolColor returns a lipgloss.Style for the tool name header based on tool type.
// Colors: Read/Glob -> cyan, Write -> green, Edit -> yellow,
// Bash/Exec/Shell -> orange, Grep -> magenta, other -> magenta.
func toolColor(name string) lipgloss.Style {
	s := Styles()
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "grep"):
		return s.ToolGrep
	case strings.Contains(lower, "edit"):
		return s.ToolEdit
	case strings.Contains(lower, "read") || strings.Contains(lower, "glob"):
		return s.ToolRead
	case strings.Contains(lower, "bash") || strings.Contains(lower, "exec") || strings.Contains(lower, "shell"):
		return s.ToolBash
	case strings.Contains(lower, "write"):
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
	images   []ImageViewModel
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
			for _, img := range msg.Images {
				vm := NewImageViewModel(img.Data, img.MimeType, m.width)
				m.images = append(m.images, vm)
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

// extractFilePath attempts to extract a file path from the tool args JSON.
// It checks common keys: file_path, path, filename.
func extractFilePath(argsJSON string) string {
	if argsJSON == "" {
		return ""
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return ""
	}
	for _, key := range []string{"file_path", "path", "filename"} {
		if v, ok := args[key]; ok {
			if s, ok := v.(string); ok && s != "" {
				return s
			}
		}
	}
	return ""
}

// View renders the tool call as a bordered box with status indicator.
// The border color matches the tool type; file paths appear as subtitles.
func (m ToolCallModel) View() string {
	if m.width <= 0 {
		return ""
	}

	s := Styles()
	nameStyle := toolColor(m.name)
	borderColor := nameStyle.GetForeground()

	// Border style with tool-specific color
	bs := lipgloss.NewStyle().Foreground(borderColor)

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

	// Top border (colored)
	b.WriteString(bs.Render(cornerTopLeft))
	if dashesLeft > 0 {
		b.WriteString(bs.Render(strings.Repeat(borderChar, dashesLeft)))
	}
	b.WriteString(header)
	if dashesRight > 0 {
		b.WriteString(bs.Render(strings.Repeat(borderChar, dashesRight)))
	}
	b.WriteString(bs.Render(cornerTopRight))
	b.WriteString("\n")

	// Content width
	contentWidth := m.width - 4
	if contentWidth < 20 {
		contentWidth = 20
	}

	// Tool info line
	b.WriteString(fmt.Sprintf("%s %-*s %s\n", bs.Render(verticalBorder), contentWidth, toolInfo, bs.Render(verticalBorder)))

	// File path subtitle when available
	if filePath := extractFilePath(m.args); filePath != "" {
		pathLine := s.Dim.Render(filePath)
		b.WriteString(fmt.Sprintf("%s %-*s %s\n", bs.Render(verticalBorder), contentWidth, pathLine, bs.Render(verticalBorder)))
	}

	// Image blocks
	for i := range m.images {
		imgView := m.images[i].View()
		if imgView != "" {
			imgLines := strings.Split(imgView, "\n")
			for _, il := range imgLines {
				b.WriteString(fmt.Sprintf("%s %s\n", bs.Render(verticalBorder), il))
			}
		}
	}

	// Error line
	if m.errMsg != "" {
		b.WriteString(fmt.Sprintf("%s %-*s %s\n", bs.Render(verticalBorder), contentWidth, s.Error.Render(m.errMsg), bs.Render(verticalBorder)))
	}

	// Expanded output
	if m.expanded && m.output != "" {
		b.WriteString(fmt.Sprintf("%s %s\n", bs.Render(verticalBorder), bs.Render(strings.Repeat(borderChar, m.width-4))))

		lines := strings.Split(strings.TrimRight(m.output, "\n"), "\n")
		for _, line := range lines {
			b.WriteString(fmt.Sprintf("%s %s\n", bs.Render(verticalBorder), line))
		}

		b.WriteString(fmt.Sprintf("%s %s\n", bs.Render(verticalBorder), ""))
	}

	// Bottom border (colored)
	bottomWidth := totalWidth - len(cornerBottomLeft) - len(cornerBottomRight)
	if bottomWidth < 0 {
		bottomWidth = 0
	}
	b.WriteString(bs.Render(cornerBottomLeft))
	b.WriteString(bs.Render(strings.Repeat(borderChar, bottomWidth)))
	b.WriteString(bs.Render(cornerBottomRight))
	b.WriteString("\n")

	// Expand/collapse hint
	if m.expanded {
		b.WriteString(s.Dim.Render("  Press Ctrl+O to collapse"))
	} else {
		b.WriteString(s.Dim.Render("  Press Ctrl+O to expand output"))
	}

	return b.String()
}
