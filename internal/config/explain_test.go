// ABOUTME: Tests for human-readable config explanation rendering
// ABOUTME: Covers empty, full, and partial settings scenarios

package config

import (
	"strings"
	"testing"
)

func TestExplain_EmptySettings(t *testing.T) {
	t.Parallel()

	s := &Settings{}
	result := Explain(s)

	if result == "" {
		t.Error("Explain should return non-empty string for empty settings")
	}
	// Should contain section headers even for empty settings
	if !strings.Contains(result, "General") {
		t.Error("should contain General section header")
	}
}

func TestExplain_FullSettings(t *testing.T) {
	t.Parallel()

	tr := true
	f := false
	s := &Settings{
		Model:       "claude-opus-4",
		BaseURL:     "https://api.example.com",
		Temperature: 0.7,
		MaxTokens:   4096,
		Yolo:        true,
		Thinking:    true,
		DefaultMode: "acceptEdits",
		Allow:       []string{"read", "write"},
		Deny:        []string{"rm*"},
		Intent: &IntentSettings{
			Enabled:            &tr,
			HeuristicThreshold: 0.85,
			AutoPlanFileCount:  10,
		},
		Prompts: &PromptsSettings{
			ActiveVersion:         "v2.0.0",
			OverridesDir:          "/custom/prompts",
			MaxSystemPromptTokens: 8192,
		},
		Personality: &PersonalitySettings{
			Profile: "concise",
			Checks: map[string]PersonalityCheck{
				"humor": {Enabled: &tr, Level: "minimal"},
			},
		},
		Telemetry: &TelemetrySettings{
			Enabled:   &tr,
			BudgetUSD: 50.0,
			WarnAtPct: 90,
		},
		Safety: &SafetySettings{
			NeverModify: []string{"*.env", ".git/*"},
			LockedKeys:  []string{"model"},
		},
		Compaction: &CompactionSettings{
			Enabled:       &f,
			ReserveTokens: 8192,
		},
		Retry: &RetrySettings{
			MaxRetries: 5,
			BaseDelay:  500,
		},
		Terminal: &TerminalSettings{
			LineWidth: 120,
			Pager:     true,
		},
	}

	result := Explain(s)

	// Check key sections appear
	sections := []string{"General", "Permissions", "Intent", "Prompts", "Personality", "Telemetry", "Safety", "Compaction", "Retry", "Terminal"}
	for _, sec := range sections {
		if !strings.Contains(result, sec) {
			t.Errorf("should contain %q section", sec)
		}
	}

	// Check key values appear
	values := []string{"claude-opus-4", "https://api.example.com", "0.85", "v2.0.0", "concise", "50.00", "*.env"}
	for _, v := range values {
		if !strings.Contains(result, v) {
			t.Errorf("should contain value %q", v)
		}
	}
}

func TestExplain_PartialSettings(t *testing.T) {
	t.Parallel()

	s := &Settings{
		Model: "claude-sonnet",
		Intent: &IntentSettings{
			HeuristicThreshold: 0.9,
		},
		Safety: &SafetySettings{
			NeverModify: []string{"*.env"},
		},
	}

	result := Explain(s)

	if !strings.Contains(result, "claude-sonnet") {
		t.Error("should contain model name")
	}
	if !strings.Contains(result, "Intent") {
		t.Error("should contain Intent section")
	}
	if !strings.Contains(result, "Safety") {
		t.Error("should contain Safety section")
	}
	if !strings.Contains(result, "0.9") {
		t.Error("should contain threshold value")
	}
}

func TestExplain_NilSettings(t *testing.T) {
	t.Parallel()

	result := Explain(nil)

	if result == "" {
		t.Error("Explain(nil) should return non-empty string")
	}
}
