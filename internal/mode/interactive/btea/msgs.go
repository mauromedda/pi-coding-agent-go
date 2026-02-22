// ABOUTME: All custom tea.Msg types for the Bubble Tea TUI
// ABOUTME: Agent events, permission flow, user actions, and internal messages

package btea

import (
	"github.com/mauromedda/pi-coding-agent-go/internal/agent"
	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

// --- Agent events (sent by bridge goroutine via Program.Send) ---

// AgentTextMsg carries streamed text from the model.
type AgentTextMsg struct{ Text string }

// AgentThinkingMsg carries extended thinking output.
type AgentThinkingMsg struct{ Text string }

// AgentToolStartMsg signals that a tool execution has begun.
type AgentToolStartMsg struct {
	ToolID   string
	ToolName string
	Args     map[string]any
}

// AgentToolUpdateMsg carries incremental tool output.
type AgentToolUpdateMsg struct {
	ToolID string
	Text   string
}

// AgentToolEndMsg signals that a tool execution has completed.
type AgentToolEndMsg struct {
	ToolID string
	Text   string
	Result *agent.ToolResult
}

// AgentUsageMsg carries token usage statistics.
type AgentUsageMsg struct{ Usage *ai.Usage }

// AgentDoneMsg signals the agent loop has finished.
type AgentDoneMsg struct{ Messages []ai.Message }

// AgentErrorMsg carries a non-recoverable agent error.
type AgentErrorMsg struct{ Err error }

// --- Permission flow ---

// PermissionReply is the user's response to a permission request.
type PermissionReply struct {
	Allowed bool
	Always  bool
}

// PermissionRequestMsg asks the user to approve or deny a tool invocation.
// The agent goroutine blocks on ReplyCh until the user responds.
type PermissionRequestMsg struct {
	Tool    string
	Args    map[string]any
	ReplyCh chan<- PermissionReply
}

// PermissionResponseMsg carries the user's reply back to the agent bridge.
type PermissionResponseMsg struct{ Reply PermissionReply }

// --- User actions ---

// SubmitPromptMsg is sent when the user submits a prompt.
type SubmitPromptMsg struct{ Text string }

// --- Internal ---

// AutoCompactMsg triggers automatic context compaction.
type AutoCompactMsg struct{}

// SpinnerTickMsg drives the spinner animation.
type SpinnerTickMsg struct{}
