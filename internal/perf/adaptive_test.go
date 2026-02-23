// ABOUTME: Tests for adaptive decisions engine: pure function from profile+tokens to params
// ABOUTME: Table-driven: local/fast/slow models produce different adaptive parameters

package perf

import (
	"testing"
)

func TestDecide(t *testing.T) {
	tests := []struct {
		name              string
		profile           ModelProfile
		inputTokens       int
		contextWindow     int
		wantMaxOutput     int
		wantCaching       bool
		wantCompact       bool
		wantStreamBuffer  int
		wantPreloadSkills bool
	}{
		{
			name: "local model with small input",
			profile: ModelProfile{
				ContextWindow:         32000,
				MaxOutputTokens:       4096,
				SupportsPromptCaching: false,
				Latency:               LatencyLocal,
			},
			inputTokens:       5000,
			contextWindow:     32000,
			wantMaxOutput:     4096,
			wantCaching:       false,
			wantCompact:       false,
			wantStreamBuffer:  4096,
			wantPreloadSkills: true,
		},
		{
			name: "local model with large input",
			profile: ModelProfile{
				ContextWindow:         32000,
				MaxOutputTokens:       4096,
				SupportsPromptCaching: false,
				Latency:               LatencyLocal,
			},
			inputTokens:       30000,
			contextWindow:     32000,
			wantMaxOutput:     1488, // 32000 - 30000 - 512
			wantCaching:       false,
			wantCompact:       true,
			wantStreamBuffer:  4096,
			wantPreloadSkills: true,
		},
		{
			name: "anthropic fast with moderate input",
			profile: ModelProfile{
				ContextWindow:         200000,
				MaxOutputTokens:       4096,
				SupportsPromptCaching: true,
				Latency:               LatencyFast,
			},
			inputTokens:       50000,
			contextWindow:     200000,
			wantMaxOutput:     4096,
			wantCaching:       true,
			wantCompact:       false,
			wantStreamBuffer:  2048,
			wantPreloadSkills: false,
		},
		{
			name: "slow frontier with large input",
			profile: ModelProfile{
				ContextWindow:         128000,
				MaxOutputTokens:       4096,
				SupportsPromptCaching: false,
				Latency:               LatencySlow,
			},
			inputTokens:       120000,
			contextWindow:     128000,
			wantMaxOutput:     4096, // clamped: 128000-120000-512=7488, but model max is 4096
			wantCaching:       false,
			wantCompact:       true,
			wantStreamBuffer:  512,
			wantPreloadSkills: false,
		},
		{
			name: "input exceeds context window",
			profile: ModelProfile{
				ContextWindow:         32000,
				MaxOutputTokens:       4096,
				SupportsPromptCaching: false,
				Latency:               LatencyFast,
			},
			inputTokens:       35000,
			contextWindow:     32000,
			wantMaxOutput:     1024, // minimum floor
			wantCaching:       false,
			wantCompact:       true,
			wantStreamBuffer:  2048,
			wantPreloadSkills: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := Decide(tt.profile, tt.inputTokens, tt.contextWindow)

			if params.MaxOutputTokens != tt.wantMaxOutput {
				t.Errorf("MaxOutputTokens = %d, want %d", params.MaxOutputTokens, tt.wantMaxOutput)
			}
			if params.UsePromptCaching != tt.wantCaching {
				t.Errorf("UsePromptCaching = %v, want %v", params.UsePromptCaching, tt.wantCaching)
			}
			if params.CompactBeforeCall != tt.wantCompact {
				t.Errorf("CompactBeforeCall = %v, want %v", params.CompactBeforeCall, tt.wantCompact)
			}
			if params.StreamBufferSize != tt.wantStreamBuffer {
				t.Errorf("StreamBufferSize = %d, want %d", params.StreamBufferSize, tt.wantStreamBuffer)
			}
			if params.PreloadSkills != tt.wantPreloadSkills {
				t.Errorf("PreloadSkills = %v, want %v", params.PreloadSkills, tt.wantPreloadSkills)
			}
		})
	}
}

func TestDecide_CompactThreshold(t *testing.T) {
	profile := ModelProfile{
		ContextWindow:   100000,
		MaxOutputTokens: 4096,
		Latency:         LatencyFast,
	}

	// At 80% of context window → should compact
	params := Decide(profile, 80001, 100000)
	if !params.CompactBeforeCall {
		t.Error("expected CompactBeforeCall at >80% utilization")
	}

	// At 79% of context window → should NOT compact
	params = Decide(profile, 79000, 100000)
	if params.CompactBeforeCall {
		t.Error("expected no compaction at <80% utilization")
	}
}
