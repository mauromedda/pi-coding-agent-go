// ABOUTME: AssistantMsgModel is a Bubble Tea leaf that renders assistant responses
// ABOUTME: Port of components/assistant_msg.go; accumulates text, thinking, and tool calls

package btea

import (
	"encoding/json"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/width"
)

// AssistantMsgModel renders an assistant's response with streamed text,
// thinking indicator, error messages, and inline tool call sub-models.
type AssistantMsgModel struct {
	text      strings.Builder
	thinking  string
	errors    []string
	toolCalls []ToolCallModel
	width     int

	// Cached text wrapping: only rewrap when width or text changes.
	cachedLines []string
	cachedWidth int
	cachedLen   int // length of text at last wrap

	// Markdown rendering (lazily initialized)
	mdRenderer *MarkdownRenderer
}

// NewAssistantMsgModel creates an empty AssistantMsgModel.
func NewAssistantMsgModel() *AssistantMsgModel {
	return &AssistantMsgModel{}
}

// Init returns nil; no commands needed for a leaf model.
func (m *AssistantMsgModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for text accumulation, thinking, and tool call routing.
func (m *AssistantMsgModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case AgentTextMsg:
		m.text.WriteString(msg.Text)

	case AgentThinkingMsg:
		m.thinking = msg.Text

	case AgentToolStartMsg:
		argsJSON, _ := json.Marshal(msg.Args)
		tc := NewToolCallModel(msg.ToolID, msg.ToolName, string(argsJSON))
		tc.width = m.width
		m.toolCalls = append(m.toolCalls, tc)

	case AgentToolUpdateMsg:
		for i := range m.toolCalls {
			if m.toolCalls[i].id == msg.ToolID {
				updated, _ := m.toolCalls[i].Update(msg)
				m.toolCalls[i] = updated.(ToolCallModel)
				break
			}
		}

	case AgentToolEndMsg:
		for i := range m.toolCalls {
			if m.toolCalls[i].id == msg.ToolID {
				updated, _ := m.toolCalls[i].Update(msg)
				m.toolCalls[i] = updated.(ToolCallModel)
				break
			}
		}

	case AgentErrorMsg:
		m.errors = append(m.errors, msg.Err.Error())

	case tea.KeyMsg:
		for i := range m.toolCalls {
			updated, _ := m.toolCalls[i].Update(msg)
			m.toolCalls[i] = updated.(ToolCallModel)
		}

	case ToggleImagesMsg:
		for i := range m.toolCalls {
			updated, _ := m.toolCalls[i].Update(msg)
			m.toolCalls[i] = updated.(ToolCallModel)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		for i := range m.toolCalls {
			updated, _ := m.toolCalls[i].Update(msg)
			m.toolCalls[i] = updated.(ToolCallModel)
		}
	}

	return m, nil
}

// ensureRenderer lazily creates the markdown renderer.
func (m *AssistantMsgModel) ensureRenderer() *MarkdownRenderer {
	if m.mdRenderer == nil {
		m.mdRenderer = NewMarkdownRenderer()
	}
	return m.mdRenderer
}

// wrapLines returns cached wrapped lines, refreshing the cache when text or width changes.
// Uses glamour markdown rendering for content, falling back to plain text wrapping.
func (m *AssistantMsgModel) wrapLines() []string {
	raw := m.text.String()
	w := m.width
	if w <= 0 {
		w = 80
	}
	// Account for left border prefix: "│ " = 2 chars
	contentWidth := max(w-2, 20)

	textLen := len(raw)
	if textLen == m.cachedLen && w == m.cachedWidth && m.cachedLines != nil {
		return m.cachedLines
	}

	if raw == "" {
		m.cachedLines = nil
		m.cachedWidth = w
		m.cachedLen = 0
		return nil
	}

	// Use markdown renderer for styled output
	rendered := m.ensureRenderer().Render(raw, contentWidth)
	if rendered != "" {
		m.cachedLines = strings.Split(rendered, "\n")
	} else {
		m.cachedLines = width.WrapTextWithAnsi(raw, contentWidth)
	}
	m.cachedWidth = w
	m.cachedLen = textLen
	return m.cachedLines
}

// View renders the assistant message with thinking indicator, text, and tool calls.
func (m *AssistantMsgModel) View() string {
	s := Styles()
	var b strings.Builder

	borderChar := s.AssistantBorder.Render("│")

	// Blank line before assistant content
	b.WriteString("\n")

	// Thinking indicator
	if m.thinking != "" {
		b.WriteString(fmt.Sprintf("%s %s %s\n", borderChar, s.Info.Render("⠋"), s.Dim.Render("Thinking...")))
	}

	// Divider between thinking and text when both present
	if m.thinking != "" && m.text.Len() > 0 {
		divWidth := max(m.width-2, 1)
		divider := s.AssistantBorder.Render("─")
		b.WriteString(fmt.Sprintf("%s %s\n", borderChar, strings.Repeat(divider, divWidth)))
	}

	// Text content with left border and cached wrapping
	lines := m.wrapLines()
	for _, line := range lines {
		b.WriteString(fmt.Sprintf("%s %s\n", borderChar, line))
	}

	// Errors with pre-computed style
	for _, errText := range m.errors {
		b.WriteString(s.AssistantError.Render(fmt.Sprintf("✗ %s", errText)) + "\n")
	}

	// Tool calls
	for i := range m.toolCalls {
		b.WriteString("\n")
		b.WriteString(m.toolCalls[i].View())
		b.WriteString("\n")
	}

	return b.String()
}
