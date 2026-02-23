// ABOUTME: Tests for manifest parsing and model compatibility checking
// ABOUTME: Validates YAML deserialization, glob matching, and edge cases

package prompts

import (
	"testing"
)

func TestLoadManifest_Valid(t *testing.T) {
	t.Parallel()

	data := []byte(`
version: "v1.0.0"
description: "Test manifest"
compatible_models:
  - "claude-*"
  - "gpt-*"
composition_order:
  - "system.md"
  - "modes/{{MODE}}.md"
variables:
  MODE: "execute"
  DATE: ""
`)

	m, err := LoadManifest(data)
	if err != nil {
		t.Fatalf("LoadManifest() error = %v", err)
	}

	if m.Version != "v1.0.0" {
		t.Errorf("Version = %q; want %q", m.Version, "v1.0.0")
	}
	if m.Description != "Test manifest" {
		t.Errorf("Description = %q; want %q", m.Description, "Test manifest")
	}
	if len(m.CompatibleModels) != 2 {
		t.Errorf("CompatibleModels len = %d; want 2", len(m.CompatibleModels))
	}
	if len(m.CompositionOrder) != 2 {
		t.Errorf("CompositionOrder len = %d; want 2", len(m.CompositionOrder))
	}
	if m.Variables["MODE"] != "execute" {
		t.Errorf("Variables[MODE] = %q; want %q", m.Variables["MODE"], "execute")
	}
}

func TestLoadManifest_InvalidYAML(t *testing.T) {
	t.Parallel()

	data := []byte(`{invalid yaml: [`)
	_, err := LoadManifest(data)
	if err == nil {
		t.Fatal("LoadManifest() expected error for invalid YAML; got nil")
	}
}

func TestManifest_IsCompatible(t *testing.T) {
	t.Parallel()

	m := &Manifest{
		CompatibleModels: []string{"claude-*", "gpt-*", "gemini-*"},
	}

	tests := []struct {
		name    string
		modelID string
		want    bool
	}{
		{"claude model matches", "claude-sonnet-4", true},
		{"gpt model matches", "gpt-4o", true},
		{"gemini model matches", "gemini-2.5-pro", true},
		{"unknown model fails", "llama-3-70b", false},
		{"empty model fails", "", false},
		{"exact prefix matches", "claude-", true},
		{"partial no wildcard", "claud", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := m.IsCompatible(tt.modelID)
			if got != tt.want {
				t.Errorf("IsCompatible(%q) = %v; want %v", tt.modelID, got, tt.want)
			}
		})
	}
}
