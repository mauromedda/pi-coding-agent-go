// ABOUTME: Tests for model resolution: built-in, custom providers, and VLLM
// ABOUTME: Covers ResolveModel with provider-prefixed IDs and default fallback

package config

import (
	"testing"

	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

func TestResolveModel_Default(t *testing.T) {
	t.Parallel()

	m, err := ResolveModel("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.ID != ai.ModelClaude4Sonnet.ID {
		t.Errorf("expected default model %q, got %q", ai.ModelClaude4Sonnet.ID, m.ID)
	}
}

func TestResolveModel_UnknownProvider(t *testing.T) {
	t.Parallel()

	_, err := ResolveModel("foobar:some-model")
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
}

func TestResolveModel_Ollama(t *testing.T) {
	t.Parallel()

	m, err := ResolveModel("ollama:llama3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Api != ai.ApiOpenAI {
		t.Errorf("expected ApiOpenAI for ollama, got %q", m.Api)
	}
	if m.ID != "llama3" {
		t.Errorf("expected model ID 'llama3', got %q", m.ID)
	}
}

func TestResolveModel_VLLM(t *testing.T) {
	t.Parallel()

	m, err := ResolveModel("vllm:Qwen/Qwen3-Coder-Next-FP8")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Api != ai.ApiOpenAI {
		t.Errorf("expected ApiOpenAI for vllm, got %q", m.Api)
	}
	if m.ID != "Qwen/Qwen3-Coder-Next-FP8" {
		t.Errorf("expected model ID 'Qwen/Qwen3-Coder-Next-FP8', got %q", m.ID)
	}
	if m.MaxTokens != 128000 {
		t.Errorf("expected MaxTokens 128000, got %d", m.MaxTokens)
	}
	if !m.SupportsTools {
		t.Error("expected SupportsTools=true for vllm custom model")
	}
}

func TestResolveModel_UnknownBareModel(t *testing.T) {
	t.Parallel()

	_, err := ResolveModel("nonexistent-model-xyz")
	if err == nil {
		t.Fatal("expected error for unknown bare model ID")
	}
}
