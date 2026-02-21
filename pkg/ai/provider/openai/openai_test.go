// ABOUTME: Tests for the OpenAI provider: text streaming, tool calls, and error handling
// ABOUTME: Uses httptest.NewServer to mock OpenAI Chat Completions SSE responses

package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

func TestProviderApi(t *testing.T) {
	t.Parallel()
	p := New("key", "")
	if got := p.Api(); got != ai.ApiOpenAI {
		t.Errorf("Api() = %q, want %q", got, ai.ApiOpenAI)
	}
}

func TestProviderStreamTextContent(t *testing.T) {
	t.Parallel()

	sseBody := buildSSETextResponse("Hello from OpenAI!")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("got Authorization %q, want %q", r.Header.Get("Authorization"), "Bearer test-key")
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("got Content-Type %q, want %q", r.Header.Get("Content-Type"), "application/json")
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decoding request body: %v", err)
		}
		if body["model"] != "gpt-4o" {
			t.Errorf("got model %q, want %q", body["model"], "gpt-4o")
		}
		if body["stream"] != true {
			t.Errorf("got stream %v, want true", body["stream"])
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(sseBody))
	}))
	t.Cleanup(srv.Close)

	provider := New("test-key", srv.URL)
	model := &ai.ModelGPT4o
	ctx := &ai.Context{
		System:   "You are helpful.",
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
	if result.Content[0].Text != "Hello from OpenAI!" {
		t.Errorf("got text %q, want %q", result.Content[0].Text, "Hello from OpenAI!")
	}
	if result.Usage.InputTokens != 10 {
		t.Errorf("got InputTokens %d, want 10", result.Usage.InputTokens)
	}
	if result.Usage.OutputTokens != 5 {
		t.Errorf("got OutputTokens %d, want 5", result.Usage.OutputTokens)
	}
}

func TestProviderStreamToolCalls(t *testing.T) {
	t.Parallel()

	sseBody := buildSSEToolCallResponse("call_abc", "get_weather", `{"city":"London"}`)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(sseBody))
	}))
	t.Cleanup(srv.Close)

	provider := New("test-key", srv.URL)
	model := &ai.ModelGPT4o
	ctx := &ai.Context{
		Messages: []ai.Message{ai.NewTextMessage(ai.RoleUser, "Weather?")},
		Tools: []ai.Tool{{
			Name:        "get_weather",
			Description: "Get the weather",
			Parameters:  json.RawMessage(`{"type":"object","properties":{"city":{"type":"string"}}}`),
		}},
	}

	stream := provider.Stream(context.Background(), model, ctx, nil)

	var toolStarted bool
	var toolDeltas []string
	for ev := range stream.Events() {
		switch ev.Type {
		case ai.EventToolUseStart:
			toolStarted = true
			if ev.ToolName != "get_weather" {
				t.Errorf("got ToolName %q, want %q", ev.ToolName, "get_weather")
			}
		case ai.EventToolUseDelta:
			toolDeltas = append(toolDeltas, ev.ToolInput)
		case ai.EventError:
			t.Fatalf("unexpected error: %v", ev.Error)
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

	var found bool
	for _, c := range result.Content {
		if c.Type == ai.ContentToolUse {
			found = true
			if c.ID != "call_abc" {
				t.Errorf("got tool ID %q, want %q", c.ID, "call_abc")
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
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)

	provider := New("bad-key", srv.URL)
	stream := provider.Stream(context.Background(), &ai.ModelGPT4o, &ai.Context{
		Messages: []ai.Message{ai.NewTextMessage(ai.RoleUser, "Hi")},
	}, nil)

	var gotError bool
	for ev := range stream.Events() {
		if ev.Type == ai.EventError {
			gotError = true
		}
	}
	if !gotError {
		t.Error("expected error event for unauthorized response")
	}
	if result := stream.Result(); result != nil {
		t.Errorf("expected nil result on error, got %v", result)
	}
}

func TestConvertMessages(t *testing.T) {
	t.Parallel()

	ctx := &ai.Context{
		System: "Be helpful.",
		Messages: []ai.Message{
			ai.NewTextMessage(ai.RoleUser, "Hello"),
			ai.NewTextMessage(ai.RoleAssistant, "Hi there"),
		},
	}

	msgs := convertMessages(ctx)
	if len(msgs) != 3 {
		t.Fatalf("got %d messages, want 3", len(msgs))
	}
	if msgs[0].Role != "system" {
		t.Errorf("got role %q, want %q", msgs[0].Role, "system")
	}
	if msgs[1].Role != "user" {
		t.Errorf("got role %q, want %q", msgs[1].Role, "user")
	}
	if msgs[2].Role != "assistant" {
		t.Errorf("got role %q, want %q", msgs[2].Role, "assistant")
	}
}

func TestConvertTools(t *testing.T) {
	t.Parallel()

	tools := []ai.Tool{
		{
			Name:        "read_file",
			Description: "Read a file",
			Parameters:  json.RawMessage(`{"type":"object"}`),
		},
	}

	defs := convertTools(tools)
	if len(defs) != 1 {
		t.Fatalf("got %d tool defs, want 1", len(defs))
	}
	if defs[0].Type != "function" {
		t.Errorf("got type %q, want %q", defs[0].Type, "function")
	}
	if defs[0].Function.Name != "read_file" {
		t.Errorf("got name %q, want %q", defs[0].Function.Name, "read_file")
	}
}

func TestMapFinishReason(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  ai.StopReason
	}{
		{"stop", ai.StopEndTurn},
		{"length", ai.StopMaxTokens},
		{"tool_calls", ai.StopToolUse},
		{"unknown", ai.StopStop},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			if got := mapFinishReason(tt.input); got != tt.want {
				t.Errorf("mapFinishReason(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestProviderNormalizesBaseURLWithV1(t *testing.T) {
	t.Parallel()

	sseBody := buildSSETextResponse("ok")
	var gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(sseBody))
	}))
	t.Cleanup(srv.Close)

	// Pass baseURL with /v1 suffix — the provider should strip it to avoid /v1/v1/...
	provider := New("test-key", srv.URL+"/v1")
	stream := provider.Stream(context.Background(), &ai.ModelGPT4o, &ai.Context{
		Messages: []ai.Message{ai.NewTextMessage(ai.RoleUser, "Hi")},
	}, nil)

	for range stream.Events() {
	}

	if gotPath != "/v1/chat/completions" {
		t.Errorf("request path = %q, want %q", gotPath, "/v1/chat/completions")
	}
}

func TestAppendTextContent(t *testing.T) {
	t.Parallel()

	msg := &ai.AssistantMessage{}
	appendTextContent(msg, "Hello")
	appendTextContent(msg, " world")

	if len(msg.Content) != 1 {
		t.Fatalf("got %d content blocks, want 1", len(msg.Content))
	}
	if msg.Content[0].Text != "Hello world" {
		t.Errorf("got text %q, want %q", msg.Content[0].Text, "Hello world")
	}
}

// buildSSETextResponse builds an OpenAI-style SSE text streaming response.
func buildSSETextResponse(text string) string {
	return fmt.Sprintf(`data: {"id":"chatcmpl-test","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"role":"assistant","content":""},"finish_reason":null}]}

data: {"id":"chatcmpl-test","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"%s"},"finish_reason":null}]}

data: {"id":"chatcmpl-test","object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}

data: [DONE]

`, escapeJSON(text))
}

// buildSSEToolCallResponse builds an OpenAI-style SSE tool call response.
func buildSSEToolCallResponse(callID, funcName, args string) string {
	return fmt.Sprintf(`data: {"id":"chatcmpl-test","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"role":"assistant","content":null,"tool_calls":[{"index":0,"id":"%s","type":"function","function":{"name":"%s","arguments":""}}]},"finish_reason":null}]}

data: {"id":"chatcmpl-test","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"%s"}}]},"finish_reason":null}]}

data: {"id":"chatcmpl-test","object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}

data: [DONE]

`, callID, funcName, escapeJSON(args))
}

func escapeJSON(s string) string {
	b, _ := json.Marshal(s)
	return string(b[1 : len(b)-1])
}
