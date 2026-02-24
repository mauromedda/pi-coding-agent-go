// ABOUTME: Tests for the SDK public API with mock providers
// ABOUTME: Covers client creation, prompt/response, events, tool calls, and lifecycle

package sdk

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"

	"github.com/mauromedda/pi-coding-agent-go/internal/agent"
	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

// mockProvider replays canned responses for testing.
type mockProvider struct {
	responses []*ai.AssistantMessage
	callCount atomic.Int32
}

func (m *mockProvider) Api() ai.Api { return ai.ApiAnthropic }

func (m *mockProvider) Stream(_ context.Context, _ *ai.Model, _ *ai.Context, _ *ai.StreamOptions) *ai.EventStream {
	idx := int(m.callCount.Add(1)) - 1
	stream := ai.NewEventStream(16)

	go func() {
		if idx >= len(m.responses) {
			stream.FinishWithError(nil)
			return
		}
		msg := m.responses[idx]
		for _, c := range msg.Content {
			switch c.Type {
			case ai.ContentText:
				stream.Send(ai.StreamEvent{Type: ai.EventContentDelta, Text: c.Text})
			}
		}
		stream.Finish(msg)
	}()

	return stream
}

var testModel = &ai.Model{
	ID:              "test-model",
	Name:            "Test",
	Api:             ai.ApiAnthropic,
	MaxOutputTokens: 8192,
	SupportsTools:   true,
}

func TestNew_WithProvider(t *testing.T) {
	t.Parallel()

	provider := &mockProvider{}
	client, err := New(
		WithProvider(provider),
		WithModelDirect(testModel),
	)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer client.Close()

	if client.provider != provider {
		t.Error("provider not set correctly")
	}
}

func TestNew_UnknownModel(t *testing.T) {
	t.Parallel()

	_, err := New(
		WithModel("nonexistent-model-xyz"),
		WithProvider(&mockProvider{}),
	)
	if err == nil {
		t.Error("expected error for unknown model ID")
	}
}

func TestClient_Prompt_SimpleText(t *testing.T) {
	t.Parallel()

	provider := &mockProvider{
		responses: []*ai.AssistantMessage{
			{
				Content:    []ai.Content{{Type: ai.ContentText, Text: "Hello, world!"}},
				StopReason: ai.StopEndTurn,
			},
		},
	}

	client, err := New(
		WithProvider(provider),
		WithModelDirect(testModel),
		WithSystemPrompt("You are helpful."),
	)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer client.Close()

	result, err := client.Prompt(context.Background(), "hi")
	if err != nil {
		t.Fatalf("Prompt() error: %v", err)
	}

	text := result.Text()
	if text != "Hello, world!" {
		t.Errorf("Text() = %q, want %q", text, "Hello, world!")
	}
}

func TestClient_Prompt_WithToolCall(t *testing.T) {
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
				Content:    []ai.Content{{Type: ai.ContentText, Text: "File says: hello"}},
				StopReason: ai.StopEndTurn,
			},
		},
	}

	readTool := &agent.AgentTool{
		Name:     "read",
		ReadOnly: true,
		Execute: func(_ context.Context, _ string, _ map[string]any, _ func(agent.ToolUpdate)) (agent.ToolResult, error) {
			return agent.ToolResult{Content: "hello"}, nil
		},
	}

	client, err := New(
		WithProvider(provider),
		WithModelDirect(testModel),
		WithTool(readTool),
	)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer client.Close()

	result, err := client.Prompt(context.Background(), "read the file")
	if err != nil {
		t.Fatalf("Prompt() error: %v", err)
	}

	text := result.Text()
	if text != "File says: hello" {
		t.Errorf("Text() = %q, want %q", text, "File says: hello")
	}

	calls := result.ToolCalls()
	if len(calls) != 1 {
		t.Errorf("ToolCalls() length = %d, want 1", len(calls))
	}
	if len(calls) > 0 && calls[0].Name != "read" {
		t.Errorf("ToolCalls()[0].Name = %q, want %q", calls[0].Name, "read")
	}
}

func TestClient_OnEvent(t *testing.T) {
	t.Parallel()

	provider := &mockProvider{
		responses: []*ai.AssistantMessage{
			{
				Content:    []ai.Content{{Type: ai.ContentText, Text: "ok"}},
				StopReason: ai.StopEndTurn,
			},
		},
	}

	client, err := New(
		WithProvider(provider),
		WithModelDirect(testModel),
	)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer client.Close()

	var eventCount atomic.Int32
	client.OnEvent(func(evt agent.AgentEvent) {
		eventCount.Add(1)
	})

	_, err = client.Prompt(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Prompt() error: %v", err)
	}

	if eventCount.Load() == 0 {
		t.Error("expected at least one event to be delivered")
	}
}

func TestClient_Close_CancelsPrompt(t *testing.T) {
	t.Parallel()

	provider := &mockProvider{
		responses: []*ai.AssistantMessage{
			{
				Content:    []ai.Content{{Type: ai.ContentText, Text: "ok"}},
				StopReason: ai.StopEndTurn,
			},
		},
	}

	client, err := New(
		WithProvider(provider),
		WithModelDirect(testModel),
	)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	// Close before prompting should cause prompt to fail
	client.Close()

	_, err = client.Prompt(context.Background(), "hello")
	if err == nil {
		t.Error("expected error after Close()")
	}
}

func TestClient_ContextCancellation(t *testing.T) {
	t.Parallel()

	provider := &mockProvider{
		responses: []*ai.AssistantMessage{
			{
				Content:    []ai.Content{{Type: ai.ContentText, Text: "ok"}},
				StopReason: ai.StopEndTurn,
			},
		},
	}

	client, err := New(
		WithProvider(provider),
		WithModelDirect(testModel),
	)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = client.Prompt(ctx, "hello")
	if err == nil {
		t.Error("expected error from cancelled context")
	}
}

func TestResult_EmptyMessages(t *testing.T) {
	t.Parallel()

	r := &Result{Messages: nil}

	if text := r.Text(); text != "" {
		t.Errorf("Text() = %q, want empty", text)
	}
	if calls := r.ToolCalls(); len(calls) != 0 {
		t.Errorf("ToolCalls() length = %d, want 0", len(calls))
	}
}

func TestResult_MultipleAssistantMessages(t *testing.T) {
	t.Parallel()

	r := &Result{
		Messages: []ai.Message{
			{Role: ai.RoleAssistant, Content: []ai.Content{{Type: ai.ContentText, Text: "first "}}},
			{Role: ai.RoleUser, Content: []ai.Content{{Type: ai.ContentText, Text: "ignored"}}},
			{Role: ai.RoleAssistant, Content: []ai.Content{{Type: ai.ContentText, Text: "second"}}},
		},
	}

	text := r.Text()
	if text != "first second" {
		t.Errorf("Text() = %q, want %q", text, "first second")
	}
}

func TestWithOptions(t *testing.T) {
	t.Parallel()

	provider := &mockProvider{}
	client, err := New(
		WithProvider(provider),
		WithModelDirect(testModel),
		WithSystemPrompt("You are a test bot."),
		WithMaxTurns(5),
		WithAPIKey("test-key"),
	)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer client.Close()

	if client.systemPrompt != "You are a test bot." {
		t.Errorf("systemPrompt = %q", client.systemPrompt)
	}
	if client.maxTurns != 5 {
		t.Errorf("maxTurns = %d, want 5", client.maxTurns)
	}
	if client.apiKey != "test-key" {
		t.Errorf("apiKey = %q", client.apiKey)
	}
}
