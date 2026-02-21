// ABOUTME: Tests for the Google Generative AI provider: text streaming, tool calls, error handling
// ABOUTME: Uses httptest.NewServer to mock Gemini API JSON streaming responses

package google

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

func TestProviderApi(t *testing.T) {
	t.Parallel()
	p := New("key", "")
	if got := p.Api(); got != ai.ApiGoogle {
		t.Errorf("Api() = %q, want %q", got, ai.ApiGoogle)
	}
}

func TestProviderStreamAPIKeyInHeader(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// C4: API key must be in header, NOT in URL
		if r.URL.Query().Get("key") != "" {
			t.Errorf("API key should NOT be in URL query, got key=%q", r.URL.Query().Get("key"))
		}
		if got := r.Header.Get("X-Goog-Api-Key"); got != "secret-key" {
			t.Errorf("got X-Goog-Api-Key %q, want %q", got, "secret-key")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := geminiResponse{
			Candidates: []geminiCandidate{{
				Content:      geminiContent{Role: "model", Parts: []geminiPart{{Text: "ok"}}},
				FinishReason: "STOP",
			}},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(srv.Close)

	provider := New("secret-key", srv.URL)
	stream := provider.Stream(context.Background(), &ai.ModelGemini25Pro, &ai.Context{
		Messages: []ai.Message{ai.NewTextMessage(ai.RoleUser, "Hi")},
	}, nil)

	for range stream.Events() {
	}
	if result := stream.Result(); result == nil {
		t.Error("expected non-nil result")
	}
}

func TestProviderStreamTextContent(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// After fix: key is in header, not URL
		if got := r.Header.Get("X-Goog-Api-Key"); got != "test-key" {
			t.Errorf("got X-Goog-Api-Key %q, want %q", got, "test-key")
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("got Content-Type %q, want %q", r.Header.Get("Content-Type"), "application/json")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Gemini streams as JSON array or newline-delimited JSON objects
		resp := geminiResponse{
			Candidates: []geminiCandidate{{
				Content:      geminiContent{Role: "model", Parts: []geminiPart{{Text: "Hello from Gemini!"}}},
				FinishReason: "STOP",
			}},
			UsageMetadata: &geminiUsage{
				PromptTokenCount:     8,
				CandidatesTokenCount: 4,
				TotalTokenCount:      12,
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(srv.Close)

	provider := New("test-key", srv.URL)
	model := &ai.ModelGemini25Pro
	ctx := &ai.Context{
		System:   "Be helpful.",
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
	if result.Content[0].Text != "Hello from Gemini!" {
		t.Errorf("got text %q, want %q", result.Content[0].Text, "Hello from Gemini!")
	}
	if result.Usage.InputTokens != 8 {
		t.Errorf("got InputTokens %d, want 8", result.Usage.InputTokens)
	}
	if result.Usage.OutputTokens != 4 {
		t.Errorf("got OutputTokens %d, want 4", result.Usage.OutputTokens)
	}
}

func TestProviderStreamToolCall(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := geminiResponse{
			Candidates: []geminiCandidate{{
				Content: geminiContent{
					Role: "model",
					Parts: []geminiPart{{
						FunctionCall: &geminiFunctionCall{
							Name: "get_weather",
							Args: map[string]string{"city": "Paris"},
						},
					}},
				},
				FinishReason: "STOP",
			}},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(srv.Close)

	provider := New("test-key", srv.URL)
	ctx := &ai.Context{
		Messages: []ai.Message{ai.NewTextMessage(ai.RoleUser, "Weather?")},
		Tools: []ai.Tool{{
			Name:        "get_weather",
			Description: "Get weather",
			Parameters:  json.RawMessage(`{"type":"object"}`),
		}},
	}

	stream := provider.Stream(context.Background(), &ai.ModelGemini25Pro, ctx, nil)

	var toolStarted bool
	for ev := range stream.Events() {
		switch ev.Type {
		case ai.EventToolUseStart:
			toolStarted = true
			if ev.ToolName != "get_weather" {
				t.Errorf("got ToolName %q, want %q", ev.ToolName, "get_weather")
			}
		case ai.EventError:
			t.Fatalf("unexpected error: %v", ev.Error)
		}
	}

	if !toolStarted {
		t.Error("did not receive tool use start event")
	}

	result := stream.Result()
	if result == nil {
		t.Fatal("Result() returned nil")
	}

	var found bool
	for _, c := range result.Content {
		if c.Type == ai.ContentToolUse && c.Name == "get_weather" {
			found = true
		}
	}
	if !found {
		t.Error("no tool_use content block in result")
	}
}

func TestProviderStreamErrorResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"message":"invalid api key"}}`))
	}))
	t.Cleanup(srv.Close)

	provider := New("bad-key", srv.URL)
	stream := provider.Stream(context.Background(), &ai.ModelGemini25Pro, &ai.Context{
		Messages: []ai.Message{ai.NewTextMessage(ai.RoleUser, "Hi")},
	}, nil)

	var gotError bool
	for ev := range stream.Events() {
		if ev.Type == ai.EventError {
			gotError = true
		}
	}
	if !gotError {
		t.Error("expected error event for forbidden response")
	}
	if result := stream.Result(); result != nil {
		t.Errorf("expected nil result on error, got %v", result)
	}
}

func TestMapRole(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input ai.Role
		want  string
	}{
		{ai.RoleUser, "user"},
		{ai.RoleAssistant, "model"},
		{ai.RoleSystem, "user"},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			t.Parallel()
			if got := mapRole(tt.input); got != tt.want {
				t.Errorf("mapRole(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestMapGeminiFinishReason(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  ai.StopReason
	}{
		{"STOP", ai.StopEndTurn},
		{"MAX_TOKENS", ai.StopMaxTokens},
		{"OTHER", ai.StopStop},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			if got := mapGeminiFinishReason(tt.input); got != tt.want {
				t.Errorf("mapGeminiFinishReason(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestBuildGeminiRequestBody(t *testing.T) {
	t.Parallel()

	ctx := &ai.Context{
		System: "Be concise.",
		Messages: []ai.Message{
			ai.NewTextMessage(ai.RoleUser, "Hello"),
		},
		Tools: []ai.Tool{{
			Name:        "read",
			Description: "Read file",
			Parameters:  json.RawMessage(`{"type":"object"}`),
		}},
	}
	opts := &ai.StreamOptions{MaxTokens: 512, Temperature: 0.7}

	req := buildGeminiRequestBody(ctx, opts)

	if req.SystemInstruction == nil {
		t.Fatal("SystemInstruction is nil")
	}
	if req.SystemInstruction.Parts[0].Text != "Be concise." {
		t.Errorf("got system %q, want %q", req.SystemInstruction.Parts[0].Text, "Be concise.")
	}
	if len(req.Contents) != 1 {
		t.Fatalf("got %d contents, want 1", len(req.Contents))
	}
	if req.Contents[0].Role != "user" {
		t.Errorf("got role %q, want %q", req.Contents[0].Role, "user")
	}
	if len(req.Tools) != 1 {
		t.Fatalf("got %d tool groups, want 1", len(req.Tools))
	}
	if len(req.Tools[0].FunctionDeclarations) != 1 {
		t.Fatalf("got %d decls, want 1", len(req.Tools[0].FunctionDeclarations))
	}
	if req.GenerationConfig == nil {
		t.Fatal("GenerationConfig is nil")
	}
	if req.GenerationConfig.MaxOutputTokens != 512 {
		t.Errorf("got MaxOutputTokens %d, want 512", req.GenerationConfig.MaxOutputTokens)
	}
}
