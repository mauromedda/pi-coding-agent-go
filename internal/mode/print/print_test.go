// ABOUTME: Tests for SDK/headless print mode covering text, JSON, stream-JSON, turns, and budget
// ABOUTME: Uses a mock provider to simulate LLM responses without network calls

package print

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/mauromedda/pi-coding-agent-go/internal/agent"
	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

// mockProvider replays canned responses for testing print mode.
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
			stream.FinishWithError(fmt.Errorf("no more mock responses"))
			return
		}

		msg := m.responses[idx]
		for _, c := range msg.Content {
			if c.Type == ai.ContentText {
				stream.Send(ai.StreamEvent{Type: ai.EventContentDelta, Text: c.Text})
			}
		}
		stream.Finish(msg)
	}()

	return stream
}

func newTestModel() *ai.Model {
	return &ai.Model{
		ID:            "test-model",
		Name:          "Test",
		Api:           ai.ApiAnthropic,
		SupportsTools: true,
	}
}

// captureStderr runs fn and captures what it writes to os.Stderr.
// Tests using this helper must NOT run in parallel because os.Stderr is global.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating pipe: %v", err)
	}

	origStderr := os.Stderr
	os.Stderr = w

	fn()

	w.Close()
	os.Stderr = origStderr

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("reading captured stderr: %v", err)
	}
	r.Close()

	return buf.String()
}

// captureStdout runs fn and captures what it writes to os.Stdout.
// Tests using this helper must NOT run in parallel because os.Stdout is global.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating pipe: %v", err)
	}

	origStdout := os.Stdout
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = origStdout

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("reading captured output: %v", err)
	}
	r.Close()

	return buf.String()
}

func TestRunWithConfig_TextFormat(t *testing.T) {
	provider := &mockProvider{
		responses: []*ai.AssistantMessage{
			{
				Content:    []ai.Content{{Type: ai.ContentText, Text: "Hello world"}},
				StopReason: ai.StopEndTurn,
			},
		},
	}

	cfg := Config{
		OutputFormat: "text",
		SystemPrompt: "You are a test assistant.",
	}
	deps := Deps{
		Provider: provider,
		Model:    newTestModel(),
	}

	output := captureStdout(t, func() {
		err := RunWithConfig(context.Background(), cfg, deps, "say hello")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(output, "Hello world") {
		t.Errorf("expected output to contain 'Hello world', got %q", output)
	}
}

func TestRunWithConfig_MaxTurns(t *testing.T) {
	toolInput := json.RawMessage(`{"path":"/tmp/test.txt"}`)

	provider := &mockProvider{
		responses: []*ai.AssistantMessage{
			{
				Content: []ai.Content{
					{Type: ai.ContentToolUse, ID: "t1", Name: "read", Input: toolInput},
				},
				StopReason: ai.StopToolUse,
			},
			{
				Content: []ai.Content{
					{Type: ai.ContentToolUse, ID: "t2", Name: "read", Input: toolInput},
				},
				StopReason: ai.StopToolUse,
			},
			{
				Content:    []ai.Content{{Type: ai.ContentText, Text: "done after many turns"}},
				StopReason: ai.StopEndTurn,
			},
		},
	}

	var toolExecCount atomic.Int32
	readTool := &agent.AgentTool{
		Name:     "read",
		ReadOnly: true,
		Execute: func(_ context.Context, _ string, _ map[string]any, _ func(agent.ToolUpdate)) (agent.ToolResult, error) {
			toolExecCount.Add(1)
			return agent.ToolResult{Content: "file content"}, nil
		},
	}

	cfg := Config{
		OutputFormat: "text",
		MaxTurns:     1,
		SystemPrompt: "test",
	}
	deps := Deps{
		Provider: provider,
		Model:    newTestModel(),
		Tools:    []*agent.AgentTool{readTool},
	}

	// Capture stderr to count "[tool: read]" lines output by the text formatter.
	// The text formatter writes tool starts to stderr. runAgentLoop should stop
	// formatting events after MaxTurns is reached, even though the agent's
	// internal goroutine may complete more calls asynchronously.
	stderr := captureStderr(t, func() {
		_ = captureStdout(t, func() {
			err := RunWithConfig(context.Background(), cfg, deps, "read a file")
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	})

	// With MaxTurns=1, only 1 tool start should be formatted to the user.
	toolStartsFormatted := strings.Count(stderr, "[tool: read]")
	if toolStartsFormatted > 1 {
		t.Errorf("expected at most 1 tool start formatted with MaxTurns=1, got %d", toolStartsFormatted)
	}
}

func TestRunWithConfig_JSONFormat(t *testing.T) {
	provider := &mockProvider{
		responses: []*ai.AssistantMessage{
			{
				Content:    []ai.Content{{Type: ai.ContentText, Text: "JSON response"}},
				StopReason: ai.StopEndTurn,
			},
		},
	}

	cfg := Config{
		OutputFormat: "json",
		SystemPrompt: "test",
	}
	deps := Deps{
		Provider: provider,
		Model:    newTestModel(),
	}

	output := captureStdout(t, func() {
		err := RunWithConfig(context.Background(), cfg, deps, "hello")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	// Output should be valid JSON with a "text" field
	var result jsonOutput
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, output)
	}
	if result.Text != "JSON response" {
		t.Errorf("expected text 'JSON response', got %q", result.Text)
	}
}

func TestRunWithConfig_StreamJSON(t *testing.T) {
	provider := &mockProvider{
		responses: []*ai.AssistantMessage{
			{
				Content:    []ai.Content{{Type: ai.ContentText, Text: "streamed text"}},
				StopReason: ai.StopEndTurn,
			},
		},
	}

	cfg := Config{
		OutputFormat: "stream-json",
		SystemPrompt: "test",
	}
	deps := Deps{
		Provider: provider,
		Model:    newTestModel(),
	}

	output := captureStdout(t, func() {
		err := RunWithConfig(context.Background(), cfg, deps, "hello")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	// stream-json should output one JSON line per event: start, text, end
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 3 {
		t.Fatalf("expected at least 3 lines (start, text, end), got %d: %s", len(lines), output)
	}

	// First line: start event
	var startEvt streamEvent
	if err := json.Unmarshal([]byte(lines[0]), &startEvt); err != nil {
		t.Fatalf("line 0 is not valid JSON: %v", err)
	}
	if startEvt.Type != "start" {
		t.Errorf("expected first event type 'start', got %q", startEvt.Type)
	}

	// Last line: end event
	var endEvt streamEvent
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &endEvt); err != nil {
		t.Fatalf("last line is not valid JSON: %v", err)
	}
	if endEvt.Type != "end" {
		t.Errorf("expected last event type 'end', got %q", endEvt.Type)
	}
}

func TestRunWithConfig_MaxBudget(t *testing.T) {
	toolInput := json.RawMessage(`{"path":"/tmp/test.txt"}`)

	provider := &mockProvider{
		responses: []*ai.AssistantMessage{
			{
				Content: []ai.Content{
					{Type: ai.ContentToolUse, ID: "t1", Name: "read", Input: toolInput},
				},
				StopReason: ai.StopToolUse,
			},
			{
				Content: []ai.Content{
					{Type: ai.ContentToolUse, ID: "t2", Name: "read", Input: toolInput},
				},
				StopReason: ai.StopToolUse,
			},
			{
				Content:    []ai.Content{{Type: ai.ContentText, Text: "done"}},
				StopReason: ai.StopEndTurn,
			},
		},
	}

	readTool := &agent.AgentTool{
		Name:     "read",
		ReadOnly: true,
		Execute: func(_ context.Context, _ string, _ map[string]any, _ func(agent.ToolUpdate)) (agent.ToolResult, error) {
			return agent.ToolResult{Content: "file content"}, nil
		},
	}

	// Budget = $0.01. Default per-turn cost estimate:
	// 1000 input tokens * $0.000003 + 500 output tokens * $0.000015 = $0.003 + $0.0075 = $0.0105.
	// After 1 turn, cumulative cost ($0.0105) exceeds budget ($0.01), so the loop should stop.
	cfg := Config{
		OutputFormat: "text",
		MaxBudgetUSD: 0.01,
		SystemPrompt: "test",
	}
	deps := Deps{
		Provider: provider,
		Model:    newTestModel(),
		Tools:    []*agent.AgentTool{readTool},
	}

	// Count how many tool starts are formatted to the user via stderr.
	stderr := captureStderr(t, func() {
		_ = captureStdout(t, func() {
			err := RunWithConfig(context.Background(), cfg, deps, "read files")
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	})

	// Only 1 tool start should be formatted; the budget is exceeded after the first turn.
	toolStartsFormatted := strings.Count(stderr, "[tool: read]")
	if toolStartsFormatted > 1 {
		t.Errorf("expected at most 1 tool start formatted with MaxBudgetUSD=0.01, got %d", toolStartsFormatted)
	}
}

func TestEstimateTurnCost(t *testing.T) {
	t.Parallel()

	// 1000 input tokens * $0.000003 + 500 output tokens * $0.000015 = $0.0105
	cost := estimateTurnCost(1000, 500)
	expected := 0.0105
	if cost < expected-0.0001 || cost > expected+0.0001 {
		t.Errorf("expected cost ~%f, got %f", expected, cost)
	}

	// Zero tokens = zero cost
	if zeroCost := estimateTurnCost(0, 0); zeroCost != 0 {
		t.Errorf("expected zero cost for zero tokens, got %f", zeroCost)
	}
}

func TestShouldAbort(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     Config
		turns   int
		costUSD float64
		want    bool
	}{
		{"no limits", Config{}, 100, 100.0, false},
		{"under turn limit", Config{MaxTurns: 5}, 3, 0, false},
		{"at turn limit", Config{MaxTurns: 5}, 5, 0, true},
		{"over turn limit", Config{MaxTurns: 5}, 6, 0, true},
		{"under budget", Config{MaxBudgetUSD: 1.0}, 0, 0.5, false},
		{"at budget", Config{MaxBudgetUSD: 1.0}, 0, 1.0, true},
		{"over budget", Config{MaxBudgetUSD: 1.0}, 0, 1.5, true},
		{"both limits under", Config{MaxTurns: 5, MaxBudgetUSD: 1.0}, 3, 0.5, false},
		{"turns hit first", Config{MaxTurns: 5, MaxBudgetUSD: 10.0}, 5, 0.5, true},
		{"budget hit first", Config{MaxTurns: 50, MaxBudgetUSD: 1.0}, 3, 1.5, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := shouldAbort(tt.cfg, tt.turns, tt.costUSD)
			if got != tt.want {
				t.Errorf("shouldAbort(%+v, %d, %f) = %v; want %v", tt.cfg, tt.turns, tt.costUSD, got, tt.want)
			}
		})
	}
}

func TestRunWithConfig_DefaultOutputFormat(t *testing.T) {
	provider := &mockProvider{
		responses: []*ai.AssistantMessage{
			{
				Content:    []ai.Content{{Type: ai.ContentText, Text: "default format"}},
				StopReason: ai.StopEndTurn,
			},
		},
	}

	// Empty OutputFormat should default to "text"
	cfg := Config{
		OutputFormat: "",
		SystemPrompt: "test",
	}
	deps := Deps{
		Provider: provider,
		Model:    newTestModel(),
	}

	output := captureStdout(t, func() {
		err := RunWithConfig(context.Background(), cfg, deps, "hello")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	// Text formatter outputs text directly (not JSON)
	if !strings.Contains(output, "default format") {
		t.Errorf("expected output to contain 'default format', got %q", output)
	}

	// Should NOT be valid JSON (text mode, not JSON mode)
	var dummy jsonOutput
	if json.Unmarshal([]byte(strings.TrimSpace(output)), &dummy) == nil && dummy.Text != "" {
		t.Error("empty OutputFormat should default to text mode, not JSON")
	}
}
