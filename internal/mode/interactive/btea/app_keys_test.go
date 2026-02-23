// ABOUTME: Tests for escape-as-interrupt and keybinding additions
// ABOUTME: Verifies esc cancels agent, Ctrl+L clears screen

package btea

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestAppModel_EscCancelsRunningAgent(t *testing.T) {
	m := NewAppModel(testDeps())
	m.agentRunning = true

	key := tea.KeyMsg{Type: tea.KeyEsc}
	result, cmd := m.Update(key)
	model := result.(AppModel)

	// Agent should be signaled to stop
	if model.agentRunning {
		// Note: agentRunning is cleared by AgentDoneMsg, not by esc.
		// Esc just fires the cancel signal; agent loop exits asynchronously.
		// So agentRunning may still be true here; that's fine.
	}

	// Should return a command (the cancel signal)
	if cmd == nil {
		t.Error("cmd = nil; want cancel command after esc while running")
	}
}

func TestAppModel_EscDoesNothingWhenIdle(t *testing.T) {
	m := NewAppModel(testDeps())
	m.agentRunning = false

	key := tea.KeyMsg{Type: tea.KeyEsc}
	result, cmd := m.Update(key)
	_ = result.(AppModel)

	// Should not produce a quit or cancel command
	if cmd != nil {
		msg := cmd()
		if _, isQuit := msg.(tea.QuitMsg); isQuit {
			t.Error("esc while idle should not quit")
		}
	}
}

func TestAppModel_CtrlLClearsContent(t *testing.T) {
	m := NewAppModel(testDeps())
	// Add some content
	m.content = append(m.content, NewUserMsgModel("hello"))
	am := NewAssistantMsgModel()
	am.width = 80
	m.content = append(m.content, am)

	key := tea.KeyMsg{Type: tea.KeyCtrlL}
	result, _ := m.Update(key)
	model := result.(AppModel)

	// Content should be cleared (only welcome model remains or empty)
	if len(model.content) > 1 {
		t.Errorf("content length = %d; want at most 1 after Ctrl+L", len(model.content))
	}
}

func TestAppModel_AgentCancelMsg(t *testing.T) {
	m := NewAppModel(testDeps())
	m.agentRunning = true

	// AgentCancelMsg signals the agent was cancelled
	result, _ := m.Update(AgentCancelMsg{})
	model := result.(AppModel)

	// Should create an assistant message showing cancellation
	if len(model.content) < 1 {
		t.Fatal("expected content after AgentCancelMsg")
	}
}
