// ABOUTME: AssistantMsgModel is a Bubble Tea leaf that renders assistant responses
// ABOUTME: Port of components/assistant_msg.go; accumulates text, thinking, and tool calls

package btea

import (
	"encoding/json"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

	case tea.WindowSizeMsg:
		m.width = msg.Width
		for i := range m.toolCalls {
			updated, _ := m.toolCalls[i].Update(msg)
			m.toolCalls[i] = updated.(ToolCallModel)
		}
	}

	return m, nil
}

// View renders the assistant message with thinking indicator, text, and tool calls.
func (m *AssistantMsgModel) View() string {
	s := Styles()
	var b strings.Builder

	// Blank line before assistant content
	b.WriteString("\n")

	// Thinking indicator
	if m.thinking != "" {
		b.WriteString(fmt.Sprintf("%s %s\n", s.Info.Render("⠋"), s.Dim.Render("Thinking...")))
	}

	// Text content
	raw := m.text.String()
	if raw != "" {
		w := m.width
		if w <= 0 {
			w = 80
		}
		lines := width.WrapTextWithAnsi(raw, w)
		for _, line := range lines {
			b.WriteString(line + "\n")
		}
	}

	// Errors with dedicated styling
	for _, errText := range m.errors {
		errStyle := lipgloss.NewStyle().
			BorderLeft(true).
			BorderStyle(lipgloss.ThickBorder()).
			BorderForeground(lipgloss.Color("1")).
			PaddingLeft(1).
			Foreground(lipgloss.Color("1"))
		b.WriteString(errStyle.Render(fmt.Sprintf("✗ %s", errText)) + "\n")
	}

	// Tool calls
	for _, tc := range m.toolCalls {
		b.WriteString("\n")
		b.WriteString(tc.View())
		b.WriteString("\n")
	}

	return b.String()
}
