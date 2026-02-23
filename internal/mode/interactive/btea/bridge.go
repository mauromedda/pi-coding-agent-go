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

// RunAgentBridge reads agent events and sends them as tea.Msg to the program.
// Blocks until the events channel is closed. The caller is responsible for
// sending AgentDoneMsg with the final messages after the bridge returns.
func RunAgentBridge(program ProgramSender, events <-chan agent.AgentEvent) {
	for evt := range events {
		switch evt.Type {
		case agent.EventAssistantText:
			program.Send(AgentTextMsg{Text: evt.Text})
		case agent.EventAssistantThinking:
			program.Send(AgentThinkingMsg{Text: evt.Text})
		case agent.EventToolStart:
			program.Send(AgentToolStartMsg{
				ToolID:   evt.ToolID,
				ToolName: evt.ToolName,
				Args:     evt.ToolArgs,
			})
		case agent.EventToolUpdate:
			program.Send(AgentToolUpdateMsg{ToolID: evt.ToolID, Text: evt.Text})
		case agent.EventToolEnd:
			msg := AgentToolEndMsg{
				ToolID: evt.ToolID,
				Text:   evt.Text,
				Result: evt.ToolResult,
			}
			if evt.ToolResult != nil {
				msg.Images = evt.ToolResult.Images
			}
			program.Send(msg)
		case agent.EventUsageUpdate:
			program.Send(AgentUsageMsg{Usage: evt.Usage})
		case agent.EventError:
			program.Send(AgentErrorMsg{Err: evt.Error})
		}
	}
}
