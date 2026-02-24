// ABOUTME: Agent-to-Bubble Tea bridge goroutine that converts agent events to tea.Msg
// ABOUTME: Reads from <-chan agent.AgentEvent, sends typed messages via ProgramSender

package btea

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mauromedda/pi-coding-agent-go/internal/agent"
)

// ProgramSender is the interface for sending messages to Bubble Tea.
// Matches *tea.Program's Send method.
type ProgramSender interface {
	Send(msg tea.Msg)
}

// bridgeEventToMsg converts a single agent event to its corresponding tea.Msg.
// Returns nil for unmapped event types (e.g., EventAgentStart, EventAgentEnd).
func bridgeEventToMsg(evt agent.AgentEvent) tea.Msg {
	switch evt.Type {
	case agent.EventAssistantText:
		return AgentTextMsg{Text: evt.Text}
	case agent.EventAssistantThinking:
		return AgentThinkingMsg{Text: evt.Text}
	case agent.EventToolStart:
		return AgentToolStartMsg{
			ToolID:   evt.ToolID,
			ToolName: evt.ToolName,
			Args:     evt.ToolArgs,
		}
	case agent.EventToolUpdate:
		return AgentToolUpdateMsg{ToolID: evt.ToolID, Text: evt.Text}
	case agent.EventToolEnd:
		msg := AgentToolEndMsg{
			ToolID: evt.ToolID,
			Text:   evt.Text,
			Result: evt.ToolResult,
		}
		if evt.ToolResult != nil {
			msg.Images = evt.ToolResult.Images
		}
		return msg
	case agent.EventUsageUpdate:
		return AgentUsageMsg{Usage: evt.Usage}
	case agent.EventError:
		return AgentErrorMsg{Err: evt.Error}
	default:
		return nil
	}
}

// RunAgentBridge reads agent events and sends them as tea.Msg to the program.
// Blocks until the events channel is closed. The caller is responsible for
// sending AgentDoneMsg with the final messages after the bridge returns.
func RunAgentBridge(program ProgramSender, events <-chan agent.AgentEvent) {
	for evt := range events {
		if msg := bridgeEventToMsg(evt); msg != nil {
			program.Send(msg)
		}
	}
}
