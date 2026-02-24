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
	if m.ID != ai.ModelClaudeSonnet.ID {
		t.Errorf("expected default model %q, got %q", ai.ModelClaudeSonnet.ID, m.ID)
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

func TestApplyModelOverrides_GlobalBaseURL(t *testing.T) {
	t.Parallel()

	m := &ai.Model{ID: "test-model", MaxTokens: 100000}
	settings := &Settings{BaseURL: "https://proxy.example.com"}

	ApplyModelOverrides(m, settings)

	if m.BaseURL != "https://proxy.example.com" {
		t.Errorf("BaseURL = %q, want proxy URL", m.BaseURL)
	}
}

func TestApplyModelOverrides_PerModel(t *testing.T) {
	t.Parallel()

	m := &ai.Model{ID: "test-model", MaxTokens: 100000, MaxOutputTokens: 4096}
	settings := &Settings{
		ModelOverrides: map[string]ModelOverride{
			"test-model": {
				BaseURL:         "https://custom.example.com",
				MaxOutputTokens: 8192,
				ContextWindow:   200000,
				CustomHeaders:   map[string]string{"X-Custom": "value"},
			},
		},
	}

	ApplyModelOverrides(m, settings)

	if m.BaseURL != "https://custom.example.com" {
		t.Errorf("BaseURL = %q, want custom URL", m.BaseURL)
	}
	if m.MaxOutputTokens != 8192 {
		t.Errorf("MaxOutputTokens = %d, want 8192", m.MaxOutputTokens)
	}
	if m.ContextWindow != 200000 {
		t.Errorf("ContextWindow = %d, want 200000", m.ContextWindow)
	}
	if m.CustomHeaders["X-Custom"] != "value" {
		t.Errorf("CustomHeaders[X-Custom] = %q, want 'value'", m.CustomHeaders["X-Custom"])
	}
}

func TestApplyModelOverrides_PerModelOverridesGlobalBaseURL(t *testing.T) {
	t.Parallel()

	m := &ai.Model{ID: "test-model"}
	settings := &Settings{
		BaseURL: "https://global.example.com",
		ModelOverrides: map[string]ModelOverride{
			"test-model": {BaseURL: "https://specific.example.com"},
		},
	}

	ApplyModelOverrides(m, settings)

	// Per-model should win: global sets it first, then per-model overrides
	if m.BaseURL != "https://specific.example.com" {
		t.Errorf("BaseURL = %q, want specific URL", m.BaseURL)
	}
}

func TestApplyModelOverrides_NilSettings(t *testing.T) {
	t.Parallel()

	m := &ai.Model{ID: "test-model", MaxTokens: 100000}
	ApplyModelOverrides(m, nil) // should not panic

	if m.MaxTokens != 100000 {
		t.Error("model should be unchanged with nil settings")
	}
}

func TestModel_EffectiveContextWindow(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		contextWindow int
		maxTokens     int
		want          int
	}{
		{"uses ContextWindow when set", 200000, 128000, 200000},
		{"falls back to MaxTokens", 0, 128000, 128000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &ai.Model{ContextWindow: tt.contextWindow, MaxTokens: tt.maxTokens}
			if got := m.EffectiveContextWindow(); got != tt.want {
				t.Errorf("EffectiveContextWindow() = %d, want %d", got, tt.want)
			}
		})
	}
}
