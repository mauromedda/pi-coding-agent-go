// ABOUTME: Tests for AssistantMsgModel Bubble Tea leaf component
// ABOUTME: Verifies text accumulation, thinking indicator, tool call routing, View output

package btea

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mauromedda/pi-coding-agent-go/internal/agent"
)

// Compile-time check: *AssistantMsgModel must satisfy tea.Model.
var _ tea.Model = &AssistantMsgModel{}

func TestAssistantMsgModel_Init(t *testing.T) {
	m := &AssistantMsgModel{}
	if cmd := m.Init(); cmd != nil {
		t.Errorf("Init() returned non-nil cmd")
	}
}

func TestAssistantMsgModel_EmptyView(t *testing.T) {
	m := &AssistantMsgModel{}
	view := m.View()
	// Empty model should produce at least a blank line
	if !strings.HasPrefix(view, "\n") {
		t.Errorf("View() should start with blank line; got %q", view[:min(20, len(view))])
	}
}

func TestAssistantMsgModel_AgentTextMsg(t *testing.T) {
	m := &AssistantMsgModel{}

	updated, _ := m.Update(AgentTextMsg{Text: "Hello "})
	m1 := updated.(*AssistantMsgModel)

	updated2, _ := m1.Update(AgentTextMsg{Text: "world"})
	m2 := updated2.(*AssistantMsgModel)

	view := m2.View()
	if !strings.Contains(view, "Hello world") {
		t.Errorf("View() missing accumulated text; got %q", view)
	}
}

func TestAssistantMsgModel_AgentThinkingMsg(t *testing.T) {
	m := &AssistantMsgModel{}

	updated, _ := m.Update(AgentThinkingMsg{Text: "reasoning about the problem"})
	m1 := updated.(*AssistantMsgModel)

	view := m1.View()
	if !strings.Contains(view, "Thinking") {
		t.Errorf("View() missing thinking indicator; got %q", view)
	}
}

func TestAssistantMsgModel_AgentToolStartMsg(t *testing.T) {
	m := &AssistantMsgModel{}
	m.width = 80

	updated, _ := m.Update(AgentToolStartMsg{
		ToolID:   "t1",
		ToolName: "Read",
		Args:     map[string]any{"path": "/tmp"},
	})
	m1 := updated.(*AssistantMsgModel)

	if len(m1.toolCalls) != 1 {
		t.Fatalf("toolCalls length = %d; want 1", len(m1.toolCalls))
	}
	if m1.toolCalls[0].name != "Read" {
		t.Errorf("toolCalls[0].name = %q; want %q", m1.toolCalls[0].name, "Read")
	}
}

func TestAssistantMsgModel_ToolUpdateRouting(t *testing.T) {
	m := &AssistantMsgModel{}
	m.width = 80

	// Add a tool call first
	updated, _ := m.Update(AgentToolStartMsg{
		ToolID:   "t1",
		ToolName: "Bash",
		Args:     map[string]any{"command": "ls"},
	})
	m1 := updated.(*AssistantMsgModel)

	// Send update to that tool
	updated2, _ := m1.Update(AgentToolUpdateMsg{ToolID: "t1", Text: "output chunk"})
	m2 := updated2.(*AssistantMsgModel)

	if m2.toolCalls[0].output != "output chunk" {
		t.Errorf("tool output = %q; want %q", m2.toolCalls[0].output, "output chunk")
	}
}

func TestAssistantMsgModel_ToolEndRouting(t *testing.T) {
	m := &AssistantMsgModel{}
	m.width = 80

	// Add a tool call
	updated, _ := m.Update(AgentToolStartMsg{
		ToolID:   "t1",
		ToolName: "Read",
		Args:     map[string]any{},
	})
	m1 := updated.(*AssistantMsgModel)

	// End the tool
	result := &agent.ToolResult{Content: "done", IsError: false}
	updated2, _ := m1.Update(AgentToolEndMsg{
		ToolID: "t1",
		Text:   "done",
		Result: result,
	})
	m2 := updated2.(*AssistantMsgModel)

	if !m2.toolCalls[0].done {
		t.Error("tool should be done after AgentToolEndMsg")
	}
}

func TestAssistantMsgModel_AgentErrorMsg(t *testing.T) {
	m := &AssistantMsgModel{}
	m.width = 80

	updated, _ := m.Update(AgentErrorMsg{Err: fmt.Errorf("connection lost")})
	m1 := updated.(*AssistantMsgModel)

	if len(m1.errors) != 1 {
		t.Fatalf("errors length = %d; want 1", len(m1.errors))
	}
	if m1.errors[0] != "connection lost" {
		t.Errorf("errors[0] = %q; want %q", m1.errors[0], "connection lost")
	}

	view := m1.View()
	if !strings.Contains(view, "connection lost") {
		t.Errorf("View() should contain error text; got %q", view)
	}
	// Should NOT be embedded as plain text in the text accumulator
	if strings.Contains(m1.Text(), "connection lost") {
		t.Error("error should not be in text; should be in errors slice")
	}
}

func TestAssistantMsgModel_MultipleErrors(t *testing.T) {
	m := &AssistantMsgModel{}
	m.width = 80

	updated, _ := m.Update(AgentErrorMsg{Err: fmt.Errorf("error one")})
	m1 := updated.(*AssistantMsgModel)
	updated2, _ := m1.Update(AgentErrorMsg{Err: fmt.Errorf("error two")})
	m2 := updated2.(*AssistantMsgModel)

	if len(m2.errors) != 2 {
		t.Fatalf("errors length = %d; want 2", len(m2.errors))
	}

	view := m2.View()
	if !strings.Contains(view, "error one") {
		t.Error("View() missing first error")
	}
	if !strings.Contains(view, "error two") {
		t.Error("View() missing second error")
	}
}

func TestAssistantMsgModel_WindowSizeMsg(t *testing.T) {
	m := &AssistantMsgModel{}

	// Add a tool call to verify propagation
	updated, _ := m.Update(AgentToolStartMsg{
		ToolID:   "t1",
		ToolName: "Read",
		Args:     map[string]any{},
	})
	m1 := updated.(*AssistantMsgModel)

	// Send window size
	updated2, _ := m1.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m2 := updated2.(*AssistantMsgModel)

	if m2.width != 100 {
		t.Errorf("width = %d; want 100", m2.width)
	}
	if m2.toolCalls[0].width != 100 {
		t.Errorf("toolCalls[0].width = %d; want 100", m2.toolCalls[0].width)
	}
}

func TestAssistantMsgModel_MultipleToolCalls(t *testing.T) {
	m := &AssistantMsgModel{}
	m.width = 80

	// Add two tool calls
	updated, _ := m.Update(AgentToolStartMsg{ToolID: "t1", ToolName: "Read", Args: map[string]any{}})
	m1 := updated.(*AssistantMsgModel)

	updated2, _ := m1.Update(AgentToolStartMsg{ToolID: "t2", ToolName: "Bash", Args: map[string]any{}})
	m2 := updated2.(*AssistantMsgModel)

	if len(m2.toolCalls) != 2 {
		t.Fatalf("toolCalls length = %d; want 2", len(m2.toolCalls))
	}

	// Update should route to correct tool
	updated3, _ := m2.Update(AgentToolUpdateMsg{ToolID: "t2", Text: "bash output"})
	m3 := updated3.(*AssistantMsgModel)

	if m3.toolCalls[0].output != "" {
		t.Errorf("toolCalls[0].output should be empty; got %q", m3.toolCalls[0].output)
	}
	if m3.toolCalls[1].output != "bash output" {
		t.Errorf("toolCalls[1].output = %q; want %q", m3.toolCalls[1].output, "bash output")
	}
}

func TestAssistantMsgModel_ViewWithToolCalls(t *testing.T) {
	m := &AssistantMsgModel{}
	m.width = 80

	// Add text and a tool call
	updated, _ := m.Update(AgentTextMsg{Text: "Let me read that file."})
	m1 := updated.(*AssistantMsgModel)

	updated2, _ := m1.Update(AgentToolStartMsg{
		ToolID:   "t1",
		ToolName: "Read",
		Args:     map[string]any{"path": "/tmp/test"},
	})
	m2 := updated2.(*AssistantMsgModel)

	view := m2.View()
	if !strings.Contains(view, "Let me read that file.") {
		t.Errorf("View() missing text content")
	}
	if !strings.Contains(view, "Read") {
		t.Errorf("View() missing tool call render")
	}
}

// --- Phase 5A: Visual hierarchy tests ---

func TestAssistantMsgModel_ViewHasLeftBorder(t *testing.T) {
	m := &AssistantMsgModel{}
	m.width = 80

	updated, _ := m.Update(AgentTextMsg{Text: "Hello world"})
	m1 := updated.(*AssistantMsgModel)

	view := m1.View()
	// Text lines should have a left border character
	if !strings.Contains(view, "│") {
		t.Errorf("View() missing left border character; got %q", view)
	}
}

func TestAssistantMsgModel_ThinkingTextDivider(t *testing.T) {
	m := &AssistantMsgModel{}
	m.width = 80

	// Add thinking then text
	updated, _ := m.Update(AgentThinkingMsg{Text: "reasoning"})
	m1 := updated.(*AssistantMsgModel)
	updated2, _ := m1.Update(AgentTextMsg{Text: "Here is my answer"})
	m2 := updated2.(*AssistantMsgModel)

	view := m2.View()
	// Should have a divider between thinking and text sections
	if !strings.Contains(view, "─") {
		t.Errorf("View() missing divider between thinking and text; got %q", view)
	}
}

func TestAssistantMsgModel_CachedWrapInvalidatesOnWidthChange(t *testing.T) {
	m := &AssistantMsgModel{}
	m.width = 80

	updated, _ := m.Update(AgentTextMsg{Text: "Hello world this is some text"})
	m1 := updated.(*AssistantMsgModel)

	// First render caches (per-block)
	_ = m1.View()
	if len(m1.blocks) == 0 {
		t.Fatal("expected at least one content block")
	}
	if m1.blocks[0].cachedWidth != 80 {
		t.Errorf("cachedWidth = %d; want 80", m1.blocks[0].cachedWidth)
	}
	if len(m1.blocks[0].cachedLines) == 0 {
		t.Error("cachedLines should be populated after View()")
	}

	// Change width: cache should invalidate on next View()
	updated2, _ := m1.Update(tea.WindowSizeMsg{Width: 40, Height: 24})
	m2 := updated2.(*AssistantMsgModel)
	_ = m2.View()
	if m2.blocks[0].cachedWidth != 40 {
		t.Errorf("cachedWidth after resize = %d; want 40", m2.blocks[0].cachedWidth)
	}
}

func TestAssistantMsgModel_CachedWrapInvalidatesOnNewText(t *testing.T) {
	m := &AssistantMsgModel{}
	m.width = 80

	updated, _ := m.Update(AgentTextMsg{Text: "First"})
	m1 := updated.(*AssistantMsgModel)
	_ = m1.View()

	// Add more text: cache should update
	updated2, _ := m1.Update(AgentTextMsg{Text: " Second"})
	m2 := updated2.(*AssistantMsgModel)
	_ = m2.View()

	// New text should produce different or longer cached lines
	if m2.Text() != "First Second" {
		t.Errorf("text = %q; want %q", m2.Text(), "First Second")
	}
	// Cache should have been refreshed (per-block)
	if len(m2.blocks) == 0 {
		t.Fatal("expected at least one content block")
	}
	if len(m2.blocks[0].cachedLines) == 0 {
		t.Error("cachedLines should not be empty after new text")
	}
}

func TestAssistantMsgModel_KeyMsgForwardedToToolCalls(t *testing.T) {
	m := &AssistantMsgModel{}
	m.width = 80

	// Add a completed tool call
	updated, _ := m.Update(AgentToolStartMsg{ToolID: "t1", ToolName: "Read", Args: map[string]any{}})
	m1 := updated.(*AssistantMsgModel)
	updated2, _ := m1.Update(AgentToolEndMsg{
		ToolID: "t1",
		Text:   "file contents",
		Result: &agent.ToolResult{Content: "file contents"},
	})
	m2 := updated2.(*AssistantMsgModel)

	if m2.toolCalls[0].expanded {
		t.Fatal("tool call should start collapsed")
	}

	// Send Ctrl+O key message; should propagate and toggle expand
	updated3, _ := m2.Update(tea.KeyMsg{Type: tea.KeyCtrlO})
	m3 := updated3.(*AssistantMsgModel)

	if !m3.toolCalls[0].expanded {
		t.Error("Ctrl+O should toggle tool call expanded via KeyMsg forwarding")
	}
}

func TestAssistantMsgModel_InterleavingOrder(t *testing.T) {
	m := &AssistantMsgModel{}
	m.width = 80

	// Sequence: text → tool → text
	updated, _ := m.Update(AgentTextMsg{Text: "Before tool."})
	m1 := updated.(*AssistantMsgModel)

	updated2, _ := m1.Update(AgentToolStartMsg{
		ToolID:   "t1",
		ToolName: "Read",
		Args:     map[string]any{"path": "/tmp"},
	})
	m2 := updated2.(*AssistantMsgModel)

	updated3, _ := m2.Update(AgentToolEndMsg{
		ToolID: "t1",
		Text:   "file contents",
		Result: &agent.ToolResult{Content: "file contents"},
	})
	m3 := updated3.(*AssistantMsgModel)

	updated4, _ := m3.Update(AgentTextMsg{Text: "After tool."})
	m4 := updated4.(*AssistantMsgModel)

	view := m4.View()

	// "Before tool." must appear BEFORE the tool call box
	// "After tool." must appear AFTER the tool call box
	beforeIdx := strings.Index(view, "Before tool.")
	toolIdx := strings.Index(view, "Read")
	afterIdx := strings.Index(view, "After tool.")

	if beforeIdx == -1 {
		t.Fatal("View() missing 'Before tool.' text")
	}
	if toolIdx == -1 {
		t.Fatal("View() missing tool call render")
	}
	if afterIdx == -1 {
		t.Fatal("View() missing 'After tool.' text")
	}

	if beforeIdx >= toolIdx {
		t.Errorf("'Before tool.' (idx %d) should appear before tool call (idx %d)", beforeIdx, toolIdx)
	}
	if afterIdx <= toolIdx {
		t.Errorf("'After tool.' (idx %d) should appear after tool call (idx %d)", afterIdx, toolIdx)
	}
}

func TestAssistantMsgModel_TextMethodConcatenatesBlocks(t *testing.T) {
	m := &AssistantMsgModel{}
	m.width = 80

	updated, _ := m.Update(AgentTextMsg{Text: "Part1 "})
	m1 := updated.(*AssistantMsgModel)

	updated2, _ := m1.Update(AgentToolStartMsg{
		ToolID: "t1", ToolName: "Bash", Args: map[string]any{},
	})
	m2 := updated2.(*AssistantMsgModel)

	updated3, _ := m2.Update(AgentTextMsg{Text: "Part2"})
	m3 := updated3.(*AssistantMsgModel)

	got := m3.Text()
	if got != "Part1 Part2" {
		t.Errorf("Text() = %q; want %q", got, "Part1 Part2")
	}
}
