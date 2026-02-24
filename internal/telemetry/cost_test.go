// ABOUTME: Tests for per-model pricing lookup and cost estimation
// ABOUTME: Covers exact match, prefix match, fallback, and edge cases

package telemetry

import (
	"math"
	"testing"
)

func TestLookupPricing_ExactMatch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		modelID string
		wantIn  float64
		wantOut float64
	}{
		{"claude-opus-4", "claude-opus-4", 15.0, 75.0},
		{"claude-sonnet-4", "claude-sonnet-4", 3.0, 15.0},
		{"claude-haiku-3.5", "claude-haiku-3.5", 0.80, 4.0},
		{"claude-3-5-sonnet", "claude-3-5-sonnet", 3.0, 15.0},
		{"claude-3-opus", "claude-3-opus", 15.0, 75.0},
		{"claude-3-haiku", "claude-3-haiku", 0.25, 1.25},
		{"gpt-4o", "gpt-4o", 2.50, 10.0},
		{"gpt-4o-mini", "gpt-4o-mini", 0.15, 0.60},
		{"gpt-4-turbo", "gpt-4-turbo", 10.0, 30.0},
		{"o1", "o1", 15.0, 60.0},
		{"o1-mini", "o1-mini", 3.0, 12.0},
		{"o3-mini", "o3-mini", 1.10, 4.40},
		{"gemini-2.0-flash", "gemini-2.0-flash", 0.10, 0.40},
		{"gemini-1.5-pro", "gemini-1.5-pro", 1.25, 5.0},
		{"gemini-1.5-flash", "gemini-1.5-flash", 0.075, 0.30},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := LookupPricing(tt.modelID)
			if got.InputPerMillion != tt.wantIn {
				t.Errorf("LookupPricing(%q).InputPerMillion = %v, want %v", tt.modelID, got.InputPerMillion, tt.wantIn)
			}
			if got.OutputPerMillion != tt.wantOut {
				t.Errorf("LookupPricing(%q).OutputPerMillion = %v, want %v", tt.modelID, got.OutputPerMillion, tt.wantOut)
			}
		})
	}
}

func TestLookupPricing_PrefixMatch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		modelID string
		wantIn  float64
		wantOut float64
	}{
		{"claude-sonnet-4-6", "claude-sonnet-4-6", 3.0, 15.0},
		{"claude-opus-4-6", "claude-opus-4-6", 15.0, 75.0},
		{"claude-haiku-3.5-20250101", "claude-haiku-3.5-20250101", 0.80, 4.0},
		{"gpt-4o-2024-08-06", "gpt-4o-2024-08-06", 2.50, 10.0},
		{"gpt-4o-mini-2024-07-18", "gpt-4o-mini-2024-07-18", 0.15, 0.60},
		{"gemini-2.0-flash-exp", "gemini-2.0-flash-exp", 0.10, 0.40},
		{"o1-2024-12-17", "o1-2024-12-17", 15.0, 60.0},
		{"o1-mini-2024-09-12", "o1-mini-2024-09-12", 3.0, 12.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := LookupPricing(tt.modelID)
			if got.InputPerMillion != tt.wantIn {
				t.Errorf("LookupPricing(%q).InputPerMillion = %v, want %v", tt.modelID, got.InputPerMillion, tt.wantIn)
			}
			if got.OutputPerMillion != tt.wantOut {
				t.Errorf("LookupPricing(%q).OutputPerMillion = %v, want %v", tt.modelID, got.OutputPerMillion, tt.wantOut)
			}
		})
	}
}

func TestLookupPricing_LongestPrefixWins(t *testing.T) {
	t.Parallel()

	// "gpt-4o-mini" should match "gpt-4o-mini" (longer), not "gpt-4o" (shorter prefix).
	got := LookupPricing("gpt-4o-mini-2024-07-18")
	if got.InputPerMillion != 0.15 {
		t.Errorf("expected gpt-4o-mini pricing (0.15), got %v", got.InputPerMillion)
	}

	// "o1-mini" should match "o1-mini", not "o1".
	got = LookupPricing("o1-mini-2024-09-12")
	if got.InputPerMillion != 3.0 {
		t.Errorf("expected o1-mini pricing (3.0), got %v", got.InputPerMillion)
	}
}

func TestLookupPricing_Fallback(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		modelID string
	}{
		{"unknown-model", "unknown-model"},
		{"llama-3.1-70b", "llama-3.1-70b"},
		{"empty-string", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := LookupPricing(tt.modelID)
			if got.InputPerMillion != fallbackPricing.InputPerMillion {
				t.Errorf("LookupPricing(%q).InputPerMillion = %v, want fallback %v", tt.modelID, got.InputPerMillion, fallbackPricing.InputPerMillion)
			}
			if got.OutputPerMillion != fallbackPricing.OutputPerMillion {
				t.Errorf("LookupPricing(%q).OutputPerMillion = %v, want fallback %v", tt.modelID, got.OutputPerMillion, fallbackPricing.OutputPerMillion)
			}
		})
	}
}

func TestEstimateCost_KnownModel(t *testing.T) {
	t.Parallel()

	// claude-sonnet-4: $3/M input, $15/M output
	// 1000 input + 500 output = (1000/1M)*3 + (500/1M)*15 = 0.003 + 0.0075 = 0.0105
	got := EstimateCost("claude-sonnet-4", 1000, 500)
	want := 0.0105
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("EstimateCost(claude-sonnet-4, 1000, 500) = %v, want %v", got, want)
	}
}

func TestEstimateCost_ZeroTokens(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  int
		output int
	}{
		{"both-zero", 0, 0},
		{"zero-input", 0, 100},
		{"zero-output", 100, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := EstimateCost("claude-sonnet-4", tt.input, tt.output)
			// With zero input tokens, cost should only come from output and vice versa.
			if tt.input == 0 && tt.output == 0 && got != 0 {
				t.Errorf("EstimateCost with zero tokens = %v, want 0", got)
			}
			if got < 0 {
				t.Errorf("EstimateCost should never be negative, got %v", got)
			}
		})
	}
}

func TestEstimateCost_LargeTokenCounts(t *testing.T) {
	t.Parallel()

	// 1 million input + 1 million output for claude-opus-4: $15 + $75 = $90
	got := EstimateCost("claude-opus-4", 1_000_000, 1_000_000)
	want := 90.0
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("EstimateCost(claude-opus-4, 1M, 1M) = %v, want %v", got, want)
	}
}
