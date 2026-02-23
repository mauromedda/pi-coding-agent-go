// ABOUTME: Compile-time verification that all custom messages satisfy tea.Msg
// ABOUTME: Also validates message field accessibility for key types

package btea

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mauromedda/pi-coding-agent-go/internal/agent"
	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

// Compile-time interface checks: every custom message must satisfy tea.Msg.
var (
	_ tea.Msg = AgentTextMsg{}
	_ tea.Msg = AgentThinkingMsg{}
	_ tea.Msg = AgentToolStartMsg{}
	_ tea.Msg = AgentToolUpdateMsg{}
	_ tea.Msg = AgentToolEndMsg{}
	_ tea.Msg = AgentUsageMsg{}
	_ tea.Msg = AgentDoneMsg{}
	_ tea.Msg = AgentErrorMsg{}

	_ tea.Msg = PermissionRequestMsg{}
	_ tea.Msg = PermissionResponseMsg{}

	_ tea.Msg = SubmitPromptMsg{}

	_ tea.Msg = AutoCompactMsg{}
	_ tea.Msg = SpinnerTickMsg{}

	_ tea.Msg = ModeTransitionMsg{}
	_ tea.Msg = SettingsChangedMsg{}
	_ tea.Msg = PlanGeneratedMsg{}
)

func TestAgentTextMsg(t *testing.T) {
	msg := AgentTextMsg{Text: "hello"}
	if msg.Text != "hello" {
		t.Errorf("AgentTextMsg.Text = %q; want %q", msg.Text, "hello")
	}
}

func TestAgentToolStartMsg(t *testing.T) {
	args := map[string]any{"path": "/tmp"}
	msg := AgentToolStartMsg{
		ToolID:   "t1",
		ToolName: "Read",
		Args:     args,
	}
	if msg.ToolID != "t1" {
		t.Errorf("ToolID = %q; want %q", msg.ToolID, "t1")
	}
	if msg.ToolName != "Read" {
		t.Errorf("ToolName = %q; want %q", msg.ToolName, "Read")
	}
	if msg.Args["path"] != "/tmp" {
		t.Errorf("Args[path] = %v; want /tmp", msg.Args["path"])
	}
}

func TestAgentToolEndMsg(t *testing.T) {
	result := &agent.ToolResult{Content: "ok", IsError: false}
	msg := AgentToolEndMsg{
		ToolID: "t1",
		Text:   "done",
		Result: result,
	}
	if msg.Result.Content != "ok" {
		t.Errorf("Result.Content = %q; want %q", msg.Result.Content, "ok")
	}
	if msg.Result.IsError {
		t.Error("Result.IsError = true; want false")
	}
}

func TestAgentUsageMsg(t *testing.T) {
	usage := &ai.Usage{InputTokens: 100, OutputTokens: 50}
	msg := AgentUsageMsg{Usage: usage}
	if msg.Usage.InputTokens != 100 {
		t.Errorf("InputTokens = %d; want 100", msg.Usage.InputTokens)
	}
}

func TestAgentDoneMsg(t *testing.T) {
	msgs := []ai.Message{{Role: ai.RoleAssistant}}
	msg := AgentDoneMsg{Messages: msgs}
	if len(msg.Messages) != 1 {
		t.Errorf("len(Messages) = %d; want 1", len(msg.Messages))
	}
}

func TestAgentErrorMsg(t *testing.T) {
	msg := AgentErrorMsg{Err: nil}
	if msg.Err != nil {
		t.Errorf("Err = %v; want nil", msg.Err)
	}
}

func TestPermissionRequestMsg(t *testing.T) {
	ch := make(chan PermissionReply, 1)
	msg := PermissionRequestMsg{
		Tool:    "Bash",
		Args:    map[string]any{"command": "ls"},
		ReplyCh: ch,
	}
	if msg.Tool != "Bash" {
		t.Errorf("Tool = %q; want %q", msg.Tool, "Bash")
	}

	// Verify the channel works for sending a reply.
	ch <- PermissionReply{Allowed: true, Always: true}
	reply := <-ch
	if !reply.Allowed || !reply.Always {
		t.Errorf("reply = %+v; want Allowed=true, Always=true", reply)
	}
}

func TestSubmitPromptMsg(t *testing.T) {
	msg := SubmitPromptMsg{Text: "hello world"}
	if msg.Text != "hello world" {
		t.Errorf("Text = %q; want %q", msg.Text, "hello world")
	}
}
