// ABOUTME: Tests for the agent loop with a mock provider
// ABOUTME: Covers text responses, tool calls, parallel read-only execution, and abort

package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

// mockProvider is a configurable mock that replays canned responses.
type mockProvider struct {
	responses []*ai.AssistantMessage
	callCount atomic.Int32
}

func (m *mockProvider) Api() ai.Api { return ai.ApiAnthropic }

func (m *mockProvider) Stream(model *ai.Model, ctx *ai.Context, opts *ai.StreamOptions) *ai.EventStream {
	idx := int(m.callCount.Add(1)) - 1
	stream := ai.NewEventStream(16)

	go func() {
		if idx >= len(m.responses) {
			stream.FinishWithError(fmt.Errorf("no more mock responses"))
			return
		}

		msg := m.responses[idx]
		for _, c := range msg.Content {
			switch c.Type {
			case ai.ContentText:
				stream.Send(ai.StreamEvent{Type: ai.EventContentDelta, Text: c.Text})
			case ai.ContentThinking:
				stream.Send(ai.StreamEvent{Type: ai.EventThinkingDelta, Text: c.Thinking})
			}
		}
		stream.Finish(msg)
	}()

	return stream
}

func collectEvents(ch <-chan AgentEvent) []AgentEvent {
	var events []AgentEvent
	for evt := range ch {
		events = append(events, evt)
	}
	return events
}

func newTestModel() *ai.Model {
	return &ai.Model{
		ID:            "test-model",
		Name:          "Test",
		Api:           ai.ApiAnthropic,
		SupportsTools: true,
	}
}

func newTestContext() *ai.Context {
	return &ai.Context{
		System:   "You are a test assistant.",
		Messages: []ai.Message{ai.NewTextMessage(ai.RoleUser, "hello")},
	}
}

func TestAgent_SimpleTextResponse(t *testing.T) {
	t.Parallel()

	provider := &mockProvider{
		responses: []*ai.AssistantMessage{
			{
				Content:    []ai.Content{{Type: ai.ContentText, Text: "Hello!"}},
				StopReason: ai.StopEndTurn,
			},
		},
	}

	ag := New(provider, newTestModel(), nil)
	events := collectEvents(ag.Prompt(context.Background(), newTestContext(), &ai.StreamOptions{}))

	var hasStart, hasEnd, hasText bool
	for _, evt := range events {
		switch evt.Type {
		case EventAgentStart:
			hasStart = true
		case EventAgentEnd:
			hasEnd = true
		case EventAssistantText:
			hasText = true
			if evt.Text != "Hello!" {
				t.Errorf("expected text 'Hello!', got %q", evt.Text)
			}
		}
	}

	if !hasStart {
		t.Error("missing EventAgentStart")
	}
	if !hasEnd {
		t.Error("missing EventAgentEnd")
	}
	if !hasText {
		t.Error("missing EventAssistantText")
	}
}

func TestAgent_SingleToolCall(t *testing.T) {
	t.Parallel()

	toolInput := json.RawMessage(`{"path":"/tmp/test.txt"}`)

	provider := &mockProvider{
		responses: []*ai.AssistantMessage{
			{
				Content: []ai.Content{
					{Type: ai.ContentToolUse, ID: "tool_1", Name: "read", Input: toolInput},
				},
				StopReason: ai.StopToolUse,
			},
			{
				Content:    []ai.Content{{Type: ai.ContentText, Text: "File content is: hello"}},
				StopReason: ai.StopEndTurn,
			},
		},
	}

	readTool := &AgentTool{
		Name:     "read",
		ReadOnly: true,
		Execute: func(_ context.Context, _ string, params map[string]any, _ func(ToolUpdate)) (ToolResult, error) {
			return ToolResult{Content: "hello"}, nil
		},
	}

	ag := New(provider, newTestModel(), []*AgentTool{readTool})
	events := collectEvents(ag.Prompt(context.Background(), newTestContext(), &ai.StreamOptions{}))

	var toolStart, toolEnd bool
	for _, evt := range events {
		switch evt.Type {
		case EventToolStart:
			toolStart = true
			if evt.ToolName != "read" {
				t.Errorf("expected tool name 'read', got %q", evt.ToolName)
			}
		case EventToolEnd:
			toolEnd = true
			if evt.ToolResult == nil {
				t.Fatal("expected ToolResult, got nil")
			}
			if evt.ToolResult.Content != "hello" {
				t.Errorf("expected result 'hello', got %q", evt.ToolResult.Content)
			}
		}
	}

	if !toolStart {
		t.Error("missing EventToolStart")
	}
	if !toolEnd {
		t.Error("missing EventToolEnd")
	}
}

func TestAgent_MultipleToolCalls(t *testing.T) {
	t.Parallel()

	provider := &mockProvider{
		responses: []*ai.AssistantMessage{
			{
				Content: []ai.Content{
					{Type: ai.ContentToolUse, ID: "t1", Name: "read", Input: json.RawMessage(`{"path":"a"}`)},
					{Type: ai.ContentToolUse, ID: "t2", Name: "write", Input: json.RawMessage(`{"path":"b","content":"x"}`)},
				},
				StopReason: ai.StopToolUse,
			},
			{
				Content:    []ai.Content{{Type: ai.ContentText, Text: "done"}},
				StopReason: ai.StopEndTurn,
			},
		},
	}

	readTool := &AgentTool{
		Name:     "read",
		ReadOnly: true,
		Execute: func(_ context.Context, _ string, _ map[string]any, _ func(ToolUpdate)) (ToolResult, error) {
			return ToolResult{Content: "content-a"}, nil
		},
	}

	writeTool := &AgentTool{
		Name:     "write",
		ReadOnly: false,
		Execute: func(_ context.Context, _ string, _ map[string]any, _ func(ToolUpdate)) (ToolResult, error) {
			return ToolResult{Content: "written"}, nil
		},
	}

	ag := New(provider, newTestModel(), []*AgentTool{readTool, writeTool})
	events := collectEvents(ag.Prompt(context.Background(), newTestContext(), &ai.StreamOptions{}))

	toolEnds := 0
	for _, evt := range events {
		if evt.Type == EventToolEnd {
			toolEnds++
		}
	}
	if toolEnds != 2 {
		t.Errorf("expected 2 tool end events, got %d", toolEnds)
	}
}

func TestAgent_ReadOnlyToolsRunInParallel(t *testing.T) {
	t.Parallel()

	provider := &mockProvider{
		responses: []*ai.AssistantMessage{
			{
				Content: []ai.Content{
					{Type: ai.ContentToolUse, ID: "t1", Name: "slow_read_a", Input: json.RawMessage(`{}`)},
					{Type: ai.ContentToolUse, ID: "t2", Name: "slow_read_b", Input: json.RawMessage(`{}`)},
				},
				StopReason: ai.StopToolUse,
			},
			{
				Content:    []ai.Content{{Type: ai.ContentText, Text: "done"}},
				StopReason: ai.StopEndTurn,
			},
		},
	}

	var running atomic.Int32
	var maxConcurrent atomic.Int32
	var mu sync.Mutex

	makeTool := func(name string) *AgentTool {
		return &AgentTool{
			Name:     name,
			ReadOnly: true,
			Execute: func(_ context.Context, _ string, _ map[string]any, _ func(ToolUpdate)) (ToolResult, error) {
				cur := running.Add(1)
				mu.Lock()
				if cur > maxConcurrent.Load() {
					maxConcurrent.Store(cur)
				}
				mu.Unlock()
				time.Sleep(50 * time.Millisecond)
				running.Add(-1)
				return ToolResult{Content: "ok"}, nil
			},
		}
	}

	ag := New(provider, newTestModel(), []*AgentTool{makeTool("slow_read_a"), makeTool("slow_read_b")})
	_ = collectEvents(ag.Prompt(context.Background(), newTestContext(), &ai.StreamOptions{}))

	if maxConcurrent.Load() < 2 {
		t.Errorf("expected concurrent execution (max concurrent >= 2), got %d", maxConcurrent.Load())
	}
}

func TestAgent_ToolBlockedByPermission(t *testing.T) {
	t.Parallel()

	toolInput := json.RawMessage(`{"path":"/tmp/test.txt","content":"x"}`)

	provider := &mockProvider{
		responses: []*ai.AssistantMessage{
			{
				Content: []ai.Content{
					{Type: ai.ContentToolUse, ID: "tool_1", Name: "write", Input: toolInput},
				},
				StopReason: ai.StopToolUse,
			},
			{
				Content:    []ai.Content{{Type: ai.ContentText, Text: "Permission denied"}},
				StopReason: ai.StopEndTurn,
			},
		},
	}

	writeTool := &AgentTool{
		Name:     "write",
		ReadOnly: false,
		Execute: func(_ context.Context, _ string, _ map[string]any, _ func(ToolUpdate)) (ToolResult, error) {
			return ToolResult{Content: "written"}, nil
		},
	}

	// permCheck always denies
	permCheck := func(tool string, args map[string]any) error {
		return fmt.Errorf("tool %q denied by permission checker", tool)
	}

	ag := NewWithPermissions(provider, newTestModel(), []*AgentTool{writeTool}, permCheck)
	events := collectEvents(ag.Prompt(context.Background(), newTestContext(), &ai.StreamOptions{}))

	var toolEndWithError bool
	for _, evt := range events {
		if evt.Type == EventToolEnd && evt.ToolResult != nil && evt.ToolResult.IsError {
			toolEndWithError = true
			break
		}
	}
	if !toolEndWithError {
		t.Error("expected tool to be blocked by permission checker with IsError result")
	}
}

func TestAgent_AbortCancelsExecution(t *testing.T) {
	t.Parallel()

	provider := &mockProvider{
		responses: []*ai.AssistantMessage{
			{
				Content: []ai.Content{
					{Type: ai.ContentToolUse, ID: "t1", Name: "slow", Input: json.RawMessage(`{}`)},
				},
				StopReason: ai.StopToolUse,
			},
		},
	}

	slowTool := &AgentTool{
		Name:     "slow",
		ReadOnly: false,
		Execute: func(ctx context.Context, _ string, _ map[string]any, _ func(ToolUpdate)) (ToolResult, error) {
			select {
			case <-ctx.Done():
				return ToolResult{Content: "cancelled", IsError: true}, fmt.Errorf("tool cancelled: %w", ctx.Err())
			case <-time.After(5 * time.Second):
				return ToolResult{Content: "done"}, nil
			}
		},
	}

	ag := New(provider, newTestModel(), []*AgentTool{slowTool})
	ch := ag.Prompt(context.Background(), newTestContext(), &ai.StreamOptions{})

	// Wait for the tool to start, then abort.
	time.Sleep(50 * time.Millisecond)
	ag.Abort()

	events := collectEvents(ch)

	if ag.State() != StateCancelled {
		t.Errorf("expected StateCancelled, got %d", ag.State())
	}

	// Should see either an error event or a tool end with error.
	var hasError bool
	for _, evt := range events {
		if evt.Type == EventError || (evt.Type == EventToolEnd && evt.ToolResult != nil && evt.ToolResult.IsError) {
			hasError = true
			break
		}
	}
	if !hasError {
		t.Error("expected an error event after abort")
	}
}
