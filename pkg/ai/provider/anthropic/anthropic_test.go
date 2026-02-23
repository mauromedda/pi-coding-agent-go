// ABOUTME: Tests for the Anthropic provider: text streaming, tool use, and error handling
// ABOUTME: Uses httptest.NewServer to mock the Anthropic Messages API SSE responses

package anthropic

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

func TestProviderStreamTextContent(t *testing.T) {
	t.Parallel()

	sseResponse := buildSSETextResponse("Hello, world!")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("got api key %q, want %q", r.Header.Get("x-api-key"), "test-key")
		}
		if r.Header.Get("anthropic-version") != "2023-06-01" {
			t.Errorf("got version %q, want %q", r.Header.Get("anthropic-version"), "2023-06-01")
		}
		if r.Header.Get("content-type") != "application/json" {
			t.Errorf("got content-type %q, want %q", r.Header.Get("content-type"), "application/json")
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(sseResponse))
	}))
	t.Cleanup(srv.Close)

	provider := New("test-key", srv.URL)
	if provider.Api() != ai.ApiAnthropic {
		t.Errorf("got Api %q, want %q", provider.Api(), ai.ApiAnthropic)
	}

	model := &ai.ModelClaude4Sonnet
	ctx := &ai.Context{
		System:   "You are a helpful assistant.",
		Messages: []ai.Message{ai.NewTextMessage(ai.RoleUser, "Hi")},
	}
	opts := &ai.StreamOptions{MaxTokens: 1024}

	stream := provider.Stream(context.Background(), model, ctx, opts)

	var texts []string
	for ev := range stream.Events() {
		switch ev.Type {
		case ai.EventContentDelta:
			texts = append(texts, ev.Text)
		case ai.EventError:
			t.Fatalf("unexpected error event: %v", ev.Error)
		}
	}

	result := stream.Result()
	if result == nil {
		t.Fatal("Result() returned nil")
	}
	if result.StopReason != ai.StopEndTurn {
		t.Errorf("got StopReason %q, want %q", result.StopReason, ai.StopEndTurn)
	}
	if len(result.Content) == 0 {
		t.Fatal("expected at least one content block")
	}
	if result.Content[0].Type != ai.ContentText {
		t.Errorf("got content type %q, want %q", result.Content[0].Type, ai.ContentText)
	}
	if result.Content[0].Text != "Hello, world!" {
		t.Errorf("got text %q, want %q", result.Content[0].Text, "Hello, world!")
	}
}

func TestProviderStreamToolUse(t *testing.T) {
	t.Parallel()

	sseResponse := buildSSEToolUseResponse("tool_123", "get_weather", `{"city":"Paris"}`)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(sseResponse))
	}))
	t.Cleanup(srv.Close)

	provider := New("test-key", srv.URL)
	model := &ai.ModelClaude4Sonnet
	ctx := &ai.Context{
		Messages: []ai.Message{ai.NewTextMessage(ai.RoleUser, "Weather in Paris?")},
		Tools:    []ai.Tool{{Name: "get_weather", Description: "Get weather"}},
	}
	opts := &ai.StreamOptions{MaxTokens: 1024}

	stream := provider.Stream(context.Background(), model, ctx, opts)

	var toolStarted bool
	var toolDeltas []string
	for ev := range stream.Events() {
		switch ev.Type {
		case ai.EventToolUseStart:
			toolStarted = true
			if ev.ToolID != "tool_123" {
				t.Errorf("got ToolID %q, want %q", ev.ToolID, "tool_123")
			}
			if ev.ToolName != "get_weather" {
				t.Errorf("got ToolName %q, want %q", ev.ToolName, "get_weather")
			}
		case ai.EventToolUseDelta:
			toolDeltas = append(toolDeltas, ev.ToolInput)
		case ai.EventError:
			t.Fatalf("unexpected error event: %v", ev.Error)
		}
	}

	if !toolStarted {
		t.Error("did not receive tool use start event")
	}
	if len(toolDeltas) == 0 {
		t.Error("did not receive tool use delta events")
	}

	result := stream.Result()
	if result == nil {
		t.Fatal("Result() returned nil")
	}
	if result.StopReason != ai.StopToolUse {
		t.Errorf("got StopReason %q, want %q", result.StopReason, ai.StopToolUse)
	}

	// Find tool_use content block.
	var found bool
	for _, c := range result.Content {
		if c.Type == ai.ContentToolUse {
			found = true
			if c.ID != "tool_123" {
				t.Errorf("got tool ID %q, want %q", c.ID, "tool_123")
			}
			if c.Name != "get_weather" {
				t.Errorf("got tool name %q, want %q", c.Name, "get_weather")
			}
		}
	}
	if !found {
		t.Error("no tool_use content block in result")
	}
}

func TestProviderStreamErrorResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		resp := map[string]any{
			"type": "error",
			"error": map[string]any{
				"type":    "authentication_error",
				"message": "invalid x-api-key",
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(srv.Close)

	provider := New("bad-key", srv.URL)
	model := &ai.ModelClaude4Sonnet
	ctx := &ai.Context{
		Messages: []ai.Message{ai.NewTextMessage(ai.RoleUser, "Hi")},
	}
	opts := &ai.StreamOptions{MaxTokens: 1024}

	stream := provider.Stream(context.Background(), model, ctx, opts)

	var gotError bool
	for ev := range stream.Events() {
		if ev.Type == ai.EventError {
			gotError = true
		}
	}

	if !gotError {
		t.Error("expected error event for unauthorized response")
	}

	result := stream.Result()
	if result != nil {
		t.Errorf("expected nil result on error, got %v", result)
	}
}

func TestMessageStartPayload_EasyjsonRoundTrip(t *testing.T) {
	t.Parallel()

	input := `{"message":{"model":"claude-sonnet-4-20250514","usage":{"input_tokens":42,"output_tokens":0}}}`
	var payload messageStartPayload
	if err := payload.UnmarshalJSON([]byte(input)); err != nil {
		t.Fatalf("UnmarshalJSON error: %v", err)
	}
	if payload.Message.Model != "claude-sonnet-4-20250514" {
		t.Errorf("model = %q; want %q", payload.Message.Model, "claude-sonnet-4-20250514")
	}
	if payload.Message.Usage.InputTokens != 42 {
		t.Errorf("input_tokens = %d; want 42", payload.Message.Usage.InputTokens)
	}

	out, err := payload.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON error: %v", err)
	}

	var roundTrip messageStartPayload
	if err := roundTrip.UnmarshalJSON(out); err != nil {
		t.Fatalf("round-trip UnmarshalJSON error: %v", err)
	}
	if roundTrip.Message.Model != payload.Message.Model {
		t.Errorf("round-trip model mismatch: %q vs %q", roundTrip.Message.Model, payload.Message.Model)
	}
}

func TestContentBlockDeltaPayload_EasyjsonRoundTrip(t *testing.T) {
	t.Parallel()

	input := `{"delta":{"type":"text_delta","text":"Hello"}}`
	var payload contentBlockDeltaPayload
	if err := payload.UnmarshalJSON([]byte(input)); err != nil {
		t.Fatalf("UnmarshalJSON error: %v", err)
	}
	if payload.Delta.Type != "text_delta" {
		t.Errorf("type = %q; want %q", payload.Delta.Type, "text_delta")
	}
	if payload.Delta.Text != "Hello" {
		t.Errorf("text = %q; want %q", payload.Delta.Text, "Hello")
	}
}

// buildSSETextResponse constructs a realistic Anthropic SSE text streaming response.
func buildSSETextResponse(text string) string {
	return fmt.Sprintf(`event: message_start
data: {"type":"message_start","message":{"id":"msg_test","type":"message","role":"assistant","content":[],"model":"claude-sonnet-4-20250514","stop_reason":null,"usage":{"input_tokens":10,"output_tokens":0}}}

event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}

event: ping
data: {"type":"ping"}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"%s"}}

event: content_block_stop
data: {"type":"content_block_stop","index":0}

event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":5}}

event: message_stop
data: {"type":"message_stop"}

`, escapeJSON(text))
}

// buildSSEToolUseResponse constructs a realistic Anthropic SSE tool use response.
func buildSSEToolUseResponse(toolID, toolName, toolInput string) string {
	return fmt.Sprintf(`event: message_start
data: {"type":"message_start","message":{"id":"msg_tool","type":"message","role":"assistant","content":[],"model":"claude-sonnet-4-20250514","stop_reason":null,"usage":{"input_tokens":10,"output_tokens":0}}}

event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"%s","name":"%s","input":{}}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"%s"}}

event: content_block_stop
data: {"type":"content_block_stop","index":0}

event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"tool_use"},"usage":{"output_tokens":15}}

event: message_stop
data: {"type":"message_stop"}

`, toolID, toolName, escapeJSON(toolInput))
}

// escapeJSON escapes a string for embedding in a JSON string value.
func escapeJSON(s string) string {
	b, _ := json.Marshal(s)
	// Remove surrounding quotes.
	return string(b[1 : len(b)-1])
}
