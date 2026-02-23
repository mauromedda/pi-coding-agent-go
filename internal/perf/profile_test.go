// ABOUTME: Tests for model profile registry: BuildProfile and ProfileCache
// ABOUTME: Table-driven tests for Anthropic detection, latency-based config, cache behavior

package perf

import (
	"testing"

	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

func TestBuildProfile(t *testing.T) {
	tests := []struct {
		name                  string
		model                 *ai.Model
		probe                 ProbeResult
		wantCaching           bool
		wantTokensPerSec      float64
		wantLatency           LatencyClass
		wantContextWindow     int
		wantMaxOutputTokens   int
	}{
		{
			name: "anthropic local",
			model: &ai.Model{
				ID: "claude-opus-4-6", Api: ai.ApiAnthropic,
				ContextWindow: 200000, MaxOutputTokens: 4096,
			},
			probe:               ProbeResult{Latency: LatencyLocal},
			wantCaching:         true,
			wantTokensPerSec:    100.0,
			wantLatency:         LatencyLocal,
			wantContextWindow:   200000,
			wantMaxOutputTokens: 4096,
		},
		{
			name: "openai fast",
			model: &ai.Model{
				ID: "gpt-4o", Api: ai.ApiOpenAI,
				ContextWindow: 128000, MaxOutputTokens: 4096,
			},
			probe:               ProbeResult{Latency: LatencyFast},
			wantCaching:         false,
			wantTokensPerSec:    80.0,
			wantLatency:         LatencyFast,
			wantContextWindow:   128000,
			wantMaxOutputTokens: 4096,
		},
		{
			name: "google slow",
			model: &ai.Model{
				ID: "gemini-2.5-pro", Api: ai.ApiGoogle,
				MaxTokens: 1000000, MaxOutputTokens: 8192,
			},
			probe:               ProbeResult{Latency: LatencySlow},
			wantCaching:         false,
			wantTokensPerSec:    40.0,
			wantLatency:         LatencySlow,
			wantContextWindow:   1000000, // falls back to MaxTokens
			wantMaxOutputTokens: 8192,
		},
		{
			name: "vertex anthropic",
			model: &ai.Model{
				ID: "claude-3.5-sonnet", Api: ai.ApiVertex,
				ContextWindow: 200000, MaxOutputTokens: 4096,
			},
			probe:            ProbeResult{Latency: LatencyFast},
			wantCaching:      false, // Vertex uses different caching mechanism
			wantTokensPerSec: 80.0,
			wantLatency:      LatencyFast,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := BuildProfile(tt.model, tt.probe)

			if p.SupportsPromptCaching != tt.wantCaching {
				t.Errorf("SupportsPromptCaching = %v, want %v", p.SupportsPromptCaching, tt.wantCaching)
			}
			if p.TokensPerSecond != tt.wantTokensPerSec {
				t.Errorf("TokensPerSecond = %v, want %v", p.TokensPerSecond, tt.wantTokensPerSec)
			}
			if p.Latency != tt.wantLatency {
				t.Errorf("Latency = %v, want %v", p.Latency, tt.wantLatency)
			}
			if tt.wantContextWindow > 0 && p.ContextWindow != tt.wantContextWindow {
				t.Errorf("ContextWindow = %d, want %d", p.ContextWindow, tt.wantContextWindow)
			}
			if tt.wantMaxOutputTokens > 0 && p.MaxOutputTokens != tt.wantMaxOutputTokens {
				t.Errorf("MaxOutputTokens = %d, want %d", p.MaxOutputTokens, tt.wantMaxOutputTokens)
			}
		})
	}
}

func TestProfileCache_MissReturnsNil(t *testing.T) {
	c := &ProfileCache{}
	if got := c.Get("unknown", "http://localhost"); got != nil {
		t.Errorf("expected nil on cache miss, got %+v", got)
	}
}

func TestProfileCache_PutThenGet(t *testing.T) {
	c := &ProfileCache{}
	profile := ModelProfile{
		ContextWindow:         200000,
		MaxOutputTokens:       4096,
		SupportsPromptCaching: true,
		Latency:               LatencyFast,
		TokensPerSecond:       80.0,
	}

	c.Put("claude-opus-4-6", "https://api.anthropic.com", profile)
	got := c.Get("claude-opus-4-6", "https://api.anthropic.com")

	if got == nil {
		t.Fatal("expected non-nil profile from cache")
	}
	if got.ContextWindow != 200000 {
		t.Errorf("ContextWindow = %d, want 200000", got.ContextWindow)
	}
	if !got.SupportsPromptCaching {
		t.Error("expected SupportsPromptCaching = true")
	}
}

func TestProfileCache_DifferentKeysIndependent(t *testing.T) {
	c := &ProfileCache{}
	p1 := ModelProfile{ContextWindow: 100}
	p2 := ModelProfile{ContextWindow: 200}

	c.Put("model-a", "url-a", p1)
	c.Put("model-b", "url-b", p2)

	got1 := c.Get("model-a", "url-a")
	got2 := c.Get("model-b", "url-b")

	if got1 == nil || got1.ContextWindow != 100 {
		t.Errorf("model-a: got %+v", got1)
	}
	if got2 == nil || got2.ContextWindow != 200 {
		t.Errorf("model-b: got %+v", got2)
	}

	// Cross-key miss
	if got := c.Get("model-a", "url-b"); got != nil {
		t.Error("expected nil for mismatched key")
	}
}
