// ABOUTME: Tests for the agent-to-Bubble Tea bridge goroutine
// ABOUTME: Verifies event mapping, ordering, and AgentDoneMsg on channel close

package btea

import (
	"sync"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mauromedda/pi-coding-agent-go/internal/agent"
	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

// mockSender collects messages sent via Send for assertion.
type mockSender struct {
	mu   sync.Mutex
	msgs []tea.Msg
}

func (s *mockSender) Send(msg tea.Msg) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.msgs = append(s.msgs, msg)
}

func (s *mockSender) Messages() []tea.Msg {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := make([]tea.Msg, len(s.msgs))
	copy(cp, s.msgs)
	return cp
}

func TestRunAgentBridge_EventMapping(t *testing.T) {
	tests := []struct {
		name  string
		event agent.AgentEvent
		check func(t *testing.T, msg tea.Msg)
	}{
		{
			name:  "assistant text maps to AgentTextMsg",
			event: agent.AgentEvent{Type: agent.EventAssistantText, Text: "hello"},
			check: func(t *testing.T, msg tea.Msg) {
				t.Helper()
				m, ok := msg.(AgentTextMsg)
				if !ok {
					t.Fatalf("got %T; want AgentTextMsg", msg)
				}
				if m.Text != "hello" {
					t.Errorf("Text = %q; want %q", m.Text, "hello")
				}
			},
		},
		{
			name:  "assistant thinking maps to AgentThinkingMsg",
			event: agent.AgentEvent{Type: agent.EventAssistantThinking, Text: "pondering"},
			check: func(t *testing.T, msg tea.Msg) {
				t.Helper()
				m, ok := msg.(AgentThinkingMsg)
				if !ok {
					t.Fatalf("got %T; want AgentThinkingMsg", msg)
				}
				if m.Text != "pondering" {
					t.Errorf("Text = %q; want %q", m.Text, "pondering")
				}
			},
		},
		{
			name: "tool start maps to AgentToolStartMsg",
			event: agent.AgentEvent{
				Type:     agent.EventToolStart,
				ToolID:   "t1",
				ToolName: "bash",
				ToolArgs: map[string]any{"cmd": "ls"},
			},
			check: func(t *testing.T, msg tea.Msg) {
				t.Helper()
				m, ok := msg.(AgentToolStartMsg)
				if !ok {
					t.Fatalf("got %T; want AgentToolStartMsg", msg)
				}
				if m.ToolID != "t1" {
					t.Errorf("ToolID = %q; want %q", m.ToolID, "t1")
				}
				if m.ToolName != "bash" {
					t.Errorf("ToolName = %q; want %q", m.ToolName, "bash")
				}
				if m.Args["cmd"] != "ls" {
					t.Errorf("Args[cmd] = %v; want ls", m.Args["cmd"])
				}
			},
		},
		{
			name:  "tool update maps to AgentToolUpdateMsg",
			event: agent.AgentEvent{Type: agent.EventToolUpdate, ToolID: "t2", Text: "progress"},
			check: func(t *testing.T, msg tea.Msg) {
				t.Helper()
				m, ok := msg.(AgentToolUpdateMsg)
				if !ok {
					t.Fatalf("got %T; want AgentToolUpdateMsg", msg)
				}
				if m.ToolID != "t2" || m.Text != "progress" {
					t.Errorf("got ToolID=%q Text=%q; want t2, progress", m.ToolID, m.Text)
				}
			},
		},
		{
			name: "tool end maps to AgentToolEndMsg",
			event: agent.AgentEvent{
				Type:       agent.EventToolEnd,
				ToolID:     "t3",
				Text:       "done",
				ToolResult: &agent.ToolResult{Content: "ok"},
			},
			check: func(t *testing.T, msg tea.Msg) {
				t.Helper()
				m, ok := msg.(AgentToolEndMsg)
				if !ok {
					t.Fatalf("got %T; want AgentToolEndMsg", msg)
				}
				if m.ToolID != "t3" || m.Text != "done" {
					t.Errorf("got ToolID=%q Text=%q; want t3, done", m.ToolID, m.Text)
				}
				if m.Result == nil || m.Result.Content != "ok" {
					t.Errorf("Result.Content = %v; want ok", m.Result)
				}
			},
		},
		{
			name:  "usage update maps to AgentUsageMsg",
			event: agent.AgentEvent{Type: agent.EventUsageUpdate, Usage: &ai.Usage{InputTokens: 100}},
			check: func(t *testing.T, msg tea.Msg) {
				t.Helper()
				m, ok := msg.(AgentUsageMsg)
				if !ok {
					t.Fatalf("got %T; want AgentUsageMsg", msg)
				}
				if m.Usage == nil || m.Usage.InputTokens != 100 {
					t.Errorf("Usage.InputTokens = %v; want 100", m.Usage)
				}
			},
		},
		{
			name:  "error maps to AgentErrorMsg",
			event: agent.AgentEvent{Type: agent.EventError, Error: errTest},
			check: func(t *testing.T, msg tea.Msg) {
				t.Helper()
				m, ok := msg.(AgentErrorMsg)
				if !ok {
					t.Fatalf("got %T; want AgentErrorMsg", msg)
				}
				if m.Err != errTest {
					t.Errorf("Err = %v; want %v", m.Err, errTest)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender := &mockSender{}
			ch := make(chan agent.AgentEvent, 1)
			ch <- tt.event
			close(ch)

			RunAgentBridge(sender, ch)

			msgs := sender.Messages()
			// Expect only the mapped message; AgentDoneMsg is sent by the caller
			if len(msgs) != 1 {
				t.Fatalf("got %d messages; want 1", len(msgs))
			}
			tt.check(t, msgs[0])
		})
	}
}

func TestRunAgentBridge_ChannelCloseOnly(t *testing.T) {
	sender := &mockSender{}
	ch := make(chan agent.AgentEvent)
	close(ch)

	RunAgentBridge(sender, ch)

	msgs := sender.Messages()
	// Bridge should NOT send AgentDoneMsg; the caller handles it
	if len(msgs) != 0 {
		t.Fatalf("got %d messages; want 0 (bridge should not send AgentDoneMsg)", len(msgs))
	}
}

func TestRunAgentBridge_MultipleEventsPreservesOrder(t *testing.T) {
	sender := &mockSender{}
	ch := make(chan agent.AgentEvent, 3)
	ch <- agent.AgentEvent{Type: agent.EventAssistantText, Text: "first"}
	ch <- agent.AgentEvent{Type: agent.EventAssistantText, Text: "second"}
	ch <- agent.AgentEvent{Type: agent.EventAssistantText, Text: "third"}
	close(ch)

	RunAgentBridge(sender, ch)

	msgs := sender.Messages()
	if len(msgs) != 3 { // 3 text; no AgentDoneMsg from bridge
		t.Fatalf("got %d messages; want 3", len(msgs))
	}

	expected := []string{"first", "second", "third"}
	for i, want := range expected {
		msg, ok := msgs[i].(AgentTextMsg)
		if !ok {
			t.Fatalf("msg[%d] = %T; want AgentTextMsg", i, msgs[i])
		}
		if msg.Text != want {
			t.Errorf("msg[%d].Text = %q; want %q", i, msg.Text, want)
		}
	}
}

func TestRunAgentBridge_IgnoresUnknownEventTypes(t *testing.T) {
	sender := &mockSender{}
	ch := make(chan agent.AgentEvent, 2)
	// EventAgentStart and EventAgentEnd are not mapped by the bridge
	ch <- agent.AgentEvent{Type: agent.EventAgentStart}
	ch <- agent.AgentEvent{Type: agent.EventAgentEnd}
	close(ch)

	RunAgentBridge(sender, ch)

	msgs := sender.Messages()
	// No messages; unknown events are ignored and AgentDoneMsg is the caller's job
	if len(msgs) != 0 {
		t.Fatalf("got %d messages; want 0", len(msgs))
	}
}

// errTest is a sentinel error for testing.
var errTest = &testError{msg: "test error"}

type testError struct{ msg string }

func (e *testError) Error() string { return e.msg }
