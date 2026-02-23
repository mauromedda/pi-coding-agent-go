// ABOUTME: ToolCallModel is a Bubble Tea leaf that renders a tool invocation box
// ABOUTME: Port of components/tool_call.go; handles streaming updates, expand/collapse

package btea

import (
	"encoding/json"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/width"
)

// toolColor returns a lipgloss.Style for the tool name header based on tool type.
// This is a convenience wrapper that calls Styles() internally.
func toolColor(name string) lipgloss.Style {
	return toolColorFromStyles(name, Styles())
}

// toolColorFromStyles returns the style for a tool name using a pre-fetched ThemeStyles,
// avoiding a redundant Styles() call when the caller already has the styles.
func toolColorFromStyles(name string, s ThemeStyles) lipgloss.Style {
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
	id             string
	name           string
	args           string
	done           bool
	errMsg         string
	output         string
	expanded       bool
	width          int
	images         []ImageViewModel
	showImages     bool
	cachedFilePath string // extracted once at creation, not per View()
}

// NewToolCallModel creates a ToolCallModel for the given tool invocation.
// File path is extracted once here rather than on every View() call.
func NewToolCallModel(id, name, args string) ToolCallModel {
	return ToolCallModel{
		id:             id,
		name:           name,
		args:           args,
		showImages:     true,
		cachedFilePath: extractFilePath(args),
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

	case ToggleImagesMsg:
		m.showImages = msg.Show

	case tea.WindowSizeMsg:
		m.width = msg.Width
		// Invalidate cached image renders so they re-render at new width
		for i := range m.images {
			m.images[i] = NewImageViewModel(m.images[i].data, m.images[i].mimeType, msg.Width)
		}

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

// padRight pads s with spaces to targetWidth visible columns.
// Uses ANSI-aware width measurement so escape codes do not inflate the count.
func padRight(s string, targetWidth int) string {
	vis := width.VisibleWidth(s)
	if vis >= targetWidth {
		return s
	}
	return s + strings.Repeat(" ", targetWidth-vis)
}

// writeBoxLine writes a bordered content line: │ <content padded to contentWidth> │
func writeBoxLine(b *strings.Builder, border string, content string, contentWidth int) {
	b.WriteString(border)
	b.WriteByte(' ')
	b.WriteString(padRight(content, contentWidth))
	b.WriteByte(' ')
	b.WriteString(border)
	b.WriteByte('\n')
}

// View renders the tool call as a bordered box with status indicator.
// The border color matches the tool type; file paths appear as subtitles.
func (m ToolCallModel) View() string {
	if m.width <= 0 {
		return ""
	}

	s := Styles()
	nameStyle := toolColorFromStyles(m.name, s)

	// Compute border style from current theme (supports mid-session theme changes)
	bs := lipgloss.NewStyle().Foreground(nameStyle.GetForeground())

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

	// Border characters (each is 1 visible column wide)
	const (
		borderChar        = "─"
		cornerTopLeft     = "┌"
		cornerTopRight    = "┐"
		cornerBottomLeft  = "└"
		cornerBottomRight = "┘"
		verticalBorder    = "│"
	)

	// Box geometry: outer width = m.width
	// Top/bottom borders: corner(1) + dashes + corner(1) = m.width visible cols
	// Content lines: │(1) + space(1) + content(contentWidth) + space(1) + │(1)
	contentWidth := max(m.width-4, 20)

	// Build top border with header
	header := " Code Generation "
	headerLen := width.VisibleWidth(header)
	innerWidth := max(m.width-2, 0) // columns between corners
	availableForDashes := max(innerWidth-headerLen, 0)
	dashesLeft := availableForDashes / 2
	dashesRight := availableForDashes - dashesLeft

	var b strings.Builder
	border := bs.Render(verticalBorder)

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
	b.WriteByte('\n')

	// Tool info line (ANSI-aware padding)
	writeBoxLine(&b, border, toolInfo, contentWidth)

	// File path subtitle when available (cached at creation)
	if m.cachedFilePath != "" {
		writeBoxLine(&b, border, s.Dim.Render(m.cachedFilePath), contentWidth)
	}

	// Image blocks
	if m.showImages {
		for i := range m.images {
			imgView := m.images[i].View()
			if imgView != "" {
				imgLines := strings.SplitSeq(imgView, "\n")
				for il := range imgLines {
					writeBoxLine(&b, border, il, contentWidth)
				}
			}
		}
	} else if len(m.images) > 0 {
		placeholder := fmt.Sprintf("[%d image(s) hidden - Alt+I to show]", len(m.images))
		writeBoxLine(&b, border, s.Dim.Render(placeholder), contentWidth)
	}

	// Error line
	if m.errMsg != "" {
		writeBoxLine(&b, border, s.Error.Render(m.errMsg), contentWidth)
	}

	// Expanded output
	if m.expanded && m.output != "" {
		// Separator line inside box
		writeBoxLine(&b, border, bs.Render(strings.Repeat(borderChar, contentWidth)), contentWidth)

		// Color edit tool output as a diff
		outputText := m.output
		if IsEditTool(m.name) {
			outputText = RenderDiff(outputText, s)
		}

		lines := strings.SplitSeq(strings.TrimRight(outputText, "\n"), "\n")
		for line := range lines {
			writeBoxLine(&b, border, line, contentWidth)
		}

		writeBoxLine(&b, border, "", contentWidth)
	}

	// Bottom border: same innerWidth as top, using visual column count (1 per corner)
	b.WriteString(bs.Render(cornerBottomLeft))
	b.WriteString(bs.Render(strings.Repeat(borderChar, innerWidth)))
	b.WriteString(bs.Render(cornerBottomRight))
	b.WriteByte('\n')

	// Expand/collapse hint
	if m.expanded {
		b.WriteString(s.Dim.Render("  Press Ctrl+O to collapse"))
	} else {
		b.WriteString(s.Dim.Render("  Press Ctrl+O to expand output"))
	}

	return b.String()
}
