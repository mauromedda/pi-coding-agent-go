// ABOUTME: All custom tea.Msg types for the Bubble Tea TUI
// ABOUTME: Agent events, permission flow, user actions, and internal messages

package btea

import (
	"time"

	"github.com/mauromedda/pi-coding-agent-go/internal/agent"
	"github.com/mauromedda/pi-coding-agent-go/internal/perf"
	"github.com/mauromedda/pi-coding-agent-go/internal/types"
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
	Images []types.ImageBlock
}

// AgentUsageMsg carries token usage statistics.
type AgentUsageMsg struct{ Usage *ai.Usage }

// AgentDoneMsg signals the agent loop has finished.
type AgentDoneMsg struct{ Messages []ai.Message }

// AgentErrorMsg carries a non-recoverable agent error.
type AgentErrorMsg struct{ Err error }

// AgentCancelMsg signals that the agent was cancelled by the user.
type AgentCancelMsg struct{}

// RetryTickMsg drives the retry countdown timer.
type RetryTickMsg struct {
	Remaining time.Duration
}

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

// CompactDoneMsg carries the result of a completed compaction.
type CompactDoneMsg struct {
	Messages    []ai.Message // compacted message list
	Summary     string       // generated summary text
	TokensSaved int          // tokens freed
}

// ToggleImagesMsg signals all tool call models to show/hide images.
type ToggleImagesMsg struct{ Show bool }

// SpinnerTickMsg drives the spinner animation.
type SpinnerTickMsg struct{}

// ProbeResultMsg carries the TTFB probe result from the background probe.
type ProbeResultMsg struct {
	Profile perf.ModelProfile
}

// --- Phase 8: TUI enhancement messages ---

// ModeTransitionMsg signals that the intent classifier detected a mode change.
type ModeTransitionMsg struct {
	From   string // Previous intent name
	To     string // New intent name
	Reason string // Why the transition occurred
}

// SettingsChangedMsg signals that configuration was reloaded.
type SettingsChangedMsg struct {
	Section string // Which section changed (e.g., "personality", "intent", "prompts")
}

// PlanGeneratedMsg signals that a plan was generated and needs review.
type PlanGeneratedMsg struct {
	Plan string // The plan content
}

// --- Queue overlay messages ---

// QueueUpdatedMsg carries the updated queue items after the overlay closes.
type QueueUpdatedMsg struct {
	Items []string
}

// QueueEditMsg signals that a queue item should be popped into the editor for editing.
type QueueEditMsg struct {
	Text  string
	Index int
}

// --- Background task lifecycle ---

// BackgroundTaskDoneMsg signals that a background task has completed.
type BackgroundTaskDoneMsg struct {
	TaskID   string
	Prompt   string
	Messages []ai.Message
	Err      error
}

// BackgroundTaskReviewMsg requests that a completed background task be replayed into the content area.
type BackgroundTaskReviewMsg struct {
	TaskID string
}

// BackgroundTaskRemoveMsg requests removal of a background task from the manager.
type BackgroundTaskRemoveMsg struct {
	TaskID string
}

// BackgroundTaskCancelMsg requests cancellation of a running background task.
type BackgroundTaskCancelMsg struct {
	TaskID string
}

// --- Async I/O results ---

// BashDoneMsg carries the result of an asynchronous bash command execution.
type BashDoneMsg struct {
	Command  string
	Output   string
	ExitCode int
}

// gitCWDMsg carries the detected git working directory.
type gitCWDMsg struct{ cwd string }

// --- Session lifecycle messages ---

// SessionLoadedMsg signals that a session was resumed and its history loaded.
type SessionLoadedMsg struct {
	SessionID string
	Messages  []ai.Message
}

// SessionSavedMsg confirms that the session was persisted to disk.
type SessionSavedMsg struct {
	SessionID string
}
