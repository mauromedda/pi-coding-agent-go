// ABOUTME: Tests for model spec parsing: thinking level extraction, alias detection
// ABOUTME: Covers colon-separated specs, case-insensitive matching, edge cases

package config

import (
	"testing"
)

func TestParseModelSpec_WithThinkingLevel(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		wantModel     string
		wantThinking  string
		wantErr       bool
	}{
		{"opus with high", "claude-opus-4-6:high", "claude-opus-4-6", "high", false},
		{"sonnet with low", "claude-sonnet-4-6:low", "claude-sonnet-4-6", "low", false},
		{"model with off", "claude-sonnet-4-6:off", "claude-sonnet-4-6", "off", false},
		{"model with medium", "some-model:medium", "some-model", "medium", false},
		{"no thinking level", "claude-opus-4-6", "claude-opus-4-6", "", false},
		{"provider prefix", "openai:gpt-4o", "openai:gpt-4o", "", false},
		{"provider with thinking", "openai:gpt-4o:high", "openai:gpt-4o", "high", false},
		{"empty input", "", "", "", false},
		{"case insensitive thinking", "model:HIGH", "model", "high", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modelID, thinking, err := ParseModelSpec(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseModelSpec(%q) error = %v; wantErr %v", tt.input, err, tt.wantErr)
			}
			if modelID != tt.wantModel {
				t.Errorf("modelID = %q; want %q", modelID, tt.wantModel)
			}
			if thinking != tt.wantThinking {
				t.Errorf("thinkingLevel = %q; want %q", thinking, tt.wantThinking)
			}
		})
	}
}

func TestIsAlias(t *testing.T) {
	tests := []struct {
		id   string
		want bool
	}{
		{"claude-opus-4-6", true},             // alias (no date suffix)
		{"claude-sonnet-4-6", true},           // alias (no date suffix)
		{"claude-haiku-4-5-20251001", false},  // has date suffix
		{"gpt-4o", true},                      // no date suffix
		{"claude-opus", true},                 // alias (no date)
		{"model-20250101", false},             // date suffix
		{"", true},                            // empty is alias
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			got := IsAlias(tt.id)
			if got != tt.want {
				t.Errorf("IsAlias(%q) = %v; want %v", tt.id, got, tt.want)
			}
		})
	}
}
