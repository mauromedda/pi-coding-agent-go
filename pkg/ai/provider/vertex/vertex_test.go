// ABOUTME: Tests for the Vertex AI provider: text streaming and error handling
// ABOUTME: Uses httptest.NewServer to mock Vertex AI JSON streaming responses

package vertex

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
	"github.com/mauromedda/pi-coding-agent-go/pkg/ai/provider/gemini"
)

func TestProviderApi(t *testing.T) {
	t.Parallel()
	p := New("proj", "us-central1", "")
	if got := p.Api(); got != ai.ApiVertex {
		t.Errorf("Api() = %q, want %q", got, ai.ApiVertex)
	}
}

func TestNewDefaults(t *testing.T) {
	t.Parallel()
	p := New("", "", "")
	if p.location != "us-central1" {
		t.Errorf("got location %q, want %q", p.location, "us-central1")
	}
}

func TestProviderStreamTextContent(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the URL contains the expected path pattern
		if !strings.Contains(r.URL.Path, "/publishers/google/models/") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if !strings.Contains(r.URL.Path, ":streamGenerateContent") {
			t.Errorf("expected streamGenerateContent in path: %s", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("got Content-Type %q, want %q", r.Header.Get("Content-Type"), "application/json")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := vertexResponse{
			Candidates: []vertexCandidate{{
				Content: gemini.Content{
					Role:  "model",
					Parts: []gemini.Part{{Text: "Hello from Vertex!"}},
				},
			}},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(srv.Close)

	provider := New("test-project", "us-central1", srv.URL)
	model := &ai.Model{
		ID:   "gemini-2.5-pro",
		Name: "Gemini 2.5 Pro",
		Api:  ai.ApiVertex,
	}
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
	if len(result.Content) == 0 {
		t.Fatal("expected at least one content block")
	}
	if result.Content[0].Text != "Hello from Vertex!" {
		t.Errorf("got text %q, want %q", result.Content[0].Text, "Hello from Vertex!")
	}
}

func TestProviderStreamErrorResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"message":"permission denied"}}`))
	}))
	t.Cleanup(srv.Close)

	provider := New("proj", "us-central1", srv.URL)
	model := &ai.Model{ID: "gemini-2.5-pro", Api: ai.ApiVertex}
	stream := provider.Stream(context.Background(), model, &ai.Context{
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

func TestVertexRole(t *testing.T) {
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
			if got := vertexRole(tt.input); got != tt.want {
				t.Errorf("vertexRole(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestBuildVertexRequestBody(t *testing.T) {
	t.Parallel()

	ctx := &ai.Context{
		System: "System prompt.",
		Messages: []ai.Message{
			ai.NewTextMessage(ai.RoleUser, "Hello"),
			ai.NewTextMessage(ai.RoleAssistant, "Hi"),
		},
		Tools: []ai.Tool{{
			Name:        "search",
			Description: "Search files",
			Parameters:  json.RawMessage(`{"type":"object"}`),
		}},
	}
	opts := &ai.StreamOptions{MaxTokens: 2048, Temperature: 0.5}

	req := buildVertexRequestBody(ctx, opts)

	if req.SystemInstruction == nil {
		t.Fatal("SystemInstruction is nil")
	}
	if req.SystemInstruction.Parts[0].Text != "System prompt." {
		t.Errorf("got system %q, want %q", req.SystemInstruction.Parts[0].Text, "System prompt.")
	}
	if len(req.Contents) != 2 {
		t.Fatalf("got %d contents, want 2", len(req.Contents))
	}
	if req.Contents[0].Role != "user" {
		t.Errorf("got first role %q, want %q", req.Contents[0].Role, "user")
	}
	if req.Contents[1].Role != "model" {
		t.Errorf("got second role %q, want %q", req.Contents[1].Role, "model")
	}
	if len(req.Tools) != 1 || len(req.Tools[0].FunctionDeclarations) != 1 {
		t.Fatal("expected 1 tool with 1 declaration")
	}
	if req.GenerationConfig == nil {
		t.Fatal("GenerationConfig is nil")
	}
	if req.GenerationConfig.MaxOutputTokens != 2048 {
		t.Errorf("got MaxOutputTokens %d, want 2048", req.GenerationConfig.MaxOutputTokens)
	}
}
