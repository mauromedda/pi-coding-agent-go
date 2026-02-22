// ABOUTME: Tests for ToolCallModel Bubble Tea leaf component
// ABOUTME: Verifies tool rendering, Update message routing, expand/collapse toggle

package btea

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mauromedda/pi-coding-agent-go/internal/agent"
)

// Compile-time check: ToolCallModel must satisfy tea.Model.
var _ tea.Model = ToolCallModel{}

func TestToolCallModel_Init(t *testing.T) {
	m := NewToolCallModel("t1", "Read", `{"path":"/tmp"}`)
	if cmd := m.Init(); cmd != nil {
		t.Errorf("Init() returned non-nil cmd")
	}
}

func TestToolCallModel_ViewContainsToolName(t *testing.T) {
	m := NewToolCallModel("t1", "Read", `{"path":"/tmp"}`)
	m.width = 80
	view := m.View()
	if !strings.Contains(view, "Read") {
		t.Errorf("View() missing tool name 'Read'; got %q", view)
	}
}

func TestToolCallModel_ViewShowsSpinnerWhenNotDone(t *testing.T) {
	m := NewToolCallModel("t1", "Bash", "ls")
	m.width = 80
	view := m.View()
	// When not done, should show the static spinner character
	if !strings.Contains(view, "⠋") {
		t.Errorf("View() missing spinner character when not done; got %q", view)
	}
}

func TestToolCallModel_UpdateAgentToolEndMsg(t *testing.T) {
	m := NewToolCallModel("t1", "Read", `{"path":"/tmp"}`)
	m.width = 80

	result := &agent.ToolResult{Content: "file contents here", IsError: false}
	updated, cmd := m.Update(AgentToolEndMsg{
		ToolID: "t1",
		Text:   "file contents here",
		Result: result,
	})
	if cmd != nil {
		t.Errorf("Update(AgentToolEndMsg) returned non-nil cmd")
	}

	tc := updated.(ToolCallModel)
	if !tc.done {
		t.Error("expected done=true after AgentToolEndMsg")
	}
	if tc.output != "file contents here" {
		t.Errorf("output = %q; want %q", tc.output, "file contents here")
	}
}

func TestToolCallModel_UpdateAgentToolEndMsgWithError(t *testing.T) {
	m := NewToolCallModel("t1", "Bash", "rm -rf /")
	m.width = 80

	result := &agent.ToolResult{Content: "permission denied", IsError: true}
	updated, _ := m.Update(AgentToolEndMsg{
		ToolID: "t1",
		Text:   "permission denied",
		Result: result,
	})

	tc := updated.(ToolCallModel)
	if !tc.done {
		t.Error("expected done=true")
	}
	if tc.errMsg != "permission denied" {
		t.Errorf("errMsg = %q; want %q", tc.errMsg, "permission denied")
	}
}

func TestToolCallModel_UpdateIgnoresWrongToolID(t *testing.T) {
	m := NewToolCallModel("t1", "Read", "args")
	m.width = 80

	updated, _ := m.Update(AgentToolEndMsg{
		ToolID: "other-id",
		Text:   "should not apply",
		Result: &agent.ToolResult{Content: "x"},
	})

	tc := updated.(ToolCallModel)
	if tc.done {
		t.Error("should not mark done for wrong ToolID")
	}
}

func TestToolCallModel_UpdateAgentToolUpdateMsg(t *testing.T) {
	m := NewToolCallModel("t1", "Bash", "ls")
	m.width = 80

	updated, _ := m.Update(AgentToolUpdateMsg{ToolID: "t1", Text: "chunk1"})
	tc := updated.(ToolCallModel)
	if tc.output != "chunk1" {
		t.Errorf("output = %q; want %q", tc.output, "chunk1")
	}

	updated2, _ := tc.Update(AgentToolUpdateMsg{ToolID: "t1", Text: "chunk2"})
	tc2 := updated2.(ToolCallModel)
	if tc2.output != "chunk1chunk2" {
		t.Errorf("output = %q; want %q", tc2.output, "chunk1chunk2")
	}
}

func TestToolCallModel_ToggleExpand(t *testing.T) {
	m := NewToolCallModel("t1", "Read", "args")
	m.width = 80
	m.done = true
	m.output = "some output"

	// Toggle expand via ctrl+o
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlO})
	tc := updated.(ToolCallModel)
	if !tc.expanded {
		t.Error("expected expanded=true after ctrl+o")
	}

	// View should contain output when expanded
	view := tc.View()
	if !strings.Contains(view, "some output") {
		t.Errorf("View() should contain output when expanded; got %q", view)
	}

	// Toggle again
	updated2, _ := tc.Update(tea.KeyMsg{Type: tea.KeyCtrlO})
	tc2 := updated2.(ToolCallModel)
	if tc2.expanded {
		t.Error("expected expanded=false after second ctrl+o")
	}
}

func TestToolCallModel_ToggleExpandIgnoredWhenNotDone(t *testing.T) {
	m := NewToolCallModel("t1", "Read", "args")
	m.width = 80
	// Not done yet

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlO})
	tc := updated.(ToolCallModel)
	if tc.expanded {
		t.Error("expanded should remain false when tool is not done")
	}
}

func TestToolCallModel_ViewDoneSuccess(t *testing.T) {
	m := NewToolCallModel("t1", "Read", "args")
	m.width = 80
	m.done = true
	m.output = "result"

	view := m.View()
	if !strings.Contains(view, "✓") {
		t.Errorf("View() missing success indicator '✓'; got %q", view)
	}
}

func TestToolCallModel_ViewDoneError(t *testing.T) {
	m := NewToolCallModel("t1", "Bash", "args")
	m.width = 80
	m.done = true
	m.errMsg = "failed"

	view := m.View()
	if !strings.Contains(view, "✗") {
		t.Errorf("View() missing error indicator '✗'; got %q", view)
	}
}

func TestToolCallModel_WindowSizeMsg(t *testing.T) {
	m := NewToolCallModel("t1", "Read", "args")
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	tc := updated.(ToolCallModel)
	if tc.width != 100 {
		t.Errorf("width = %d; want 100", tc.width)
	}
}

func TestToolColor(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
	}{
		{"read tool", "Read"},
		{"glob tool", "Glob"},
		{"grep tool", "Grep"},
		{"bash tool", "Bash"},
		{"exec tool", "Exec"},
		{"write tool", "Write"},
		{"edit tool", "Edit"},
		{"other tool", "CustomTool"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := toolColor(tt.toolName)
			// Verify it returns a usable style (no panic)
			rendered := s.Render("test")
			if rendered == "" {
				t.Errorf("toolColor(%q).Render('test') is empty", tt.toolName)
			}
		})
	}
}

func TestToolCallModel_ViewBorderBox(t *testing.T) {
	m := NewToolCallModel("t1", "Read", `{"path":"/tmp"}`)
	m.width = 80
	view := m.View()

	// Should have box-drawing characters for the bordered box
	boxParts := []string{"┌", "┐", "└", "┘", "│"}
	for _, part := range boxParts {
		if !strings.Contains(view, part) {
			t.Errorf("View() missing box character %q", part)
		}
	}
}

// Suppress unused import lint for lipgloss (used in compile-time type check above).
var _ = lipgloss.Style{}
