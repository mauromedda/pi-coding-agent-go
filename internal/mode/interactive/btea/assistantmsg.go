// ABOUTME: AssistantMsgModel is a Bubble Tea leaf that renders assistant responses
// ABOUTME: Uses ordered content blocks to preserve chronological text/tool interleaving

package btea

import (
	"encoding/json"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/width"
)

// blockKind distinguishes text blocks from tool call blocks in the content stream.
type blockKind int

const (
	blockText blockKind = iota
	blockTool
)

// contentBlock represents a single chronological unit in the assistant response:
// either a text segment or a reference to a tool call.
type contentBlock struct {
	kind blockKind

	// blockText fields
	text        string
	cachedLines []string
	cachedWidth int
	cachedLen   int

	// blockTool fields: index into AssistantMsgModel.toolCalls
	toolIdx int
}

// AssistantMsgModel renders an assistant's response with streamed text,
// thinking indicator, error messages, and inline tool call sub-models.
// Content blocks preserve chronological ordering of text and tool calls.
type AssistantMsgModel struct {
	blocks    []contentBlock
	curText   strings.Builder // accumulator for current text block
	thinking  string
	errors    []string
	toolCalls []ToolCallModel
	width     int

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

// Text returns all text content concatenated across blocks, preserving
// the public API that consumers use to extract the raw assistant text.
func (m *AssistantMsgModel) Text() string {
	var b strings.Builder
	for i := range m.blocks {
		if m.blocks[i].kind == blockText {
			b.WriteString(m.blocks[i].text)
		}
	}
	return b.String()
}

// hasText returns true if any text has been accumulated.
func (m *AssistantMsgModel) hasText() bool {
	for i := range m.blocks {
		if m.blocks[i].kind == blockText && m.blocks[i].text != "" {
			return true
		}
	}
	return false
}

// flushCurText ensures curText content is reflected in the last text block.
// If the last block is blockText, it updates its text field.
// Otherwise, it appends a new blockText block.
func (m *AssistantMsgModel) flushCurText() {
	if m.curText.Len() == 0 {
		return
	}
	text := m.curText.String()
	n := len(m.blocks)
	if n > 0 && m.blocks[n-1].kind == blockText {
		m.blocks[n-1].text = text
	} else {
		m.blocks = append(m.blocks, contentBlock{kind: blockText, text: text})
	}
}

// Update handles messages for text accumulation, thinking, and tool call routing.
func (m *AssistantMsgModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case AgentTextMsg:
		m.curText.WriteString(msg.Text)
		// Update or append the current text block
		m.flushCurText()

	case AgentThinkingMsg:
		m.thinking = msg.Text

	case AgentToolStartMsg:
		// Flush any pending text into its block, then start a new text accumulator
		m.flushCurText()
		m.curText.Reset()

		argsJSON, _ := json.Marshal(msg.Args)
		tc := NewToolCallModel(msg.ToolID, msg.ToolName, string(argsJSON))
		tc.width = m.width
		m.toolCalls = append(m.toolCalls, tc)
		m.blocks = append(m.blocks, contentBlock{
			kind:    blockTool,
			toolIdx: len(m.toolCalls) - 1,
		})

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

// wrapBlockLines returns cached wrapped lines for a content block,
// refreshing the cache when text or width changes.
func (m *AssistantMsgModel) wrapBlockLines(block *contentBlock) []string {
	w := m.width
	if w <= 0 {
		w = 80
	}
	// Account for left border prefix: "│ " = 2 chars
	contentWidth := max(w-2, 20)

	textLen := len(block.text)
	if textLen == block.cachedLen && w == block.cachedWidth && block.cachedLines != nil {
		return block.cachedLines
	}

	if block.text == "" {
		block.cachedLines = nil
		block.cachedWidth = w
		block.cachedLen = 0
		return nil
	}

	// Use markdown renderer for styled output
	rendered := m.ensureRenderer().Render(block.text, contentWidth)
	if rendered != "" {
		block.cachedLines = strings.Split(rendered, "\n")
	} else {
		block.cachedLines = width.WrapTextWithAnsi(block.text, contentWidth)
	}
	block.cachedWidth = w
	block.cachedLen = textLen
	return block.cachedLines
}

// View renders the assistant message with thinking indicator, text, and tool calls.
// Content blocks are rendered in chronological order to preserve interleaving.
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
	if m.thinking != "" && m.hasText() {
		divWidth := max(m.width-2, 1)
		divider := s.AssistantBorder.Render("─")
		b.WriteString(fmt.Sprintf("%s %s\n", borderChar, strings.Repeat(divider, divWidth)))
	}

	// Content blocks in chronological order
	for i := range m.blocks {
		block := &m.blocks[i]
		switch block.kind {
		case blockText:
			lines := m.wrapBlockLines(block)
			for _, line := range lines {
				b.WriteString(fmt.Sprintf("%s %s\n", borderChar, line))
			}
		case blockTool:
			if block.toolIdx < len(m.toolCalls) {
				b.WriteString("\n")
				b.WriteString(m.toolCalls[block.toolIdx].View())
				b.WriteString("\n")
			}
		}
	}

	// Errors (rendered after all blocks)
	for _, errText := range m.errors {
		b.WriteString(s.AssistantError.Render(fmt.Sprintf("✗ %s", errText)) + "\n")
	}

	return b.String()
}
