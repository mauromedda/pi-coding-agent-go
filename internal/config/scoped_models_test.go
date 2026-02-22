// ABOUTME: Tests for scoped models

package config

import (
	"path/filepath"
	"testing"
)

func TestThinkingLevel_String(t *testing.T) {
	tests := []struct {
		level    ThinkingLevel
		expected string
	}{
		{ThinkingOff, "off"},
		{ThinkingMinimal, "minimal"},
		{ThinkingLow, "low"},
		{ThinkingMedium, "medium"},
		{ThinkingHigh, "high"},
		{ThinkingXHigh, "xhigh"},
	}

	for _, test := range tests {
		if test.level.String() != test.expected {
			t.Errorf("ThinkingLevel(%d).String() = %s, want %s", test.level, test.level.String(), test.expected)
		}
	}
}

func TestThinkingLevelFromString(t *testing.T) {
	tests := []struct {
		input    string
		expected ThinkingLevel
	}{
		{"off", ThinkingOff},
		{"minimal", ThinkingMinimal},
		{"low", ThinkingLow},
		{"medium", ThinkingMedium},
		{"high", ThinkingHigh},
		{"xhigh", ThinkingXHigh},
		{"invalid", ThinkingOff}, // Default
	}

	for _, test := range tests {
		if result := ThinkingLevelFromString(test.input); result != test.expected {
			t.Errorf("ThinkingLevelFromString(%q) = %d, want %d", test.input, result, test.expected)
		}
	}
}

func TestScopedModelsConfig_New(t *testing.T) {
	cfg := NewScopedModelsConfig()
	if cfg == nil {
		t.Fatal("NewScopedModelsConfig returned nil")
	}
	if len(cfg.Models) == 0 {
		t.Error("Expected default models")
	}
}

func TestScopedModelsConfig_SaveLoad(t *testing.T) {
	cfg := NewScopedModelsConfig()
	cfg.Models = append(cfg.Models, ScopedModel{
		Name:       "test-model",
		Thinking:   ThinkingHigh,
		Provider:   "openai",
		Capabilities: []string{"tools", "vision"},
	})

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "scoped-models.json")

	if err := cfg.SaveScopedModels(path); err != nil {
		t.Fatalf("SaveScopedModels failed: %v", err)
	}

	loaded, err := LoadScopedModels(path)
	if err != nil {
		t.Fatalf("LoadScopedModels failed: %v", err)
	}

	if len(loaded.Models) != len(cfg.Models) {
		t.Errorf("Expected %d models, got %d", len(cfg.Models), len(loaded.Models))
	}

	// Check model with capabilities
	found := false
	for _, m := range loaded.Models {
		if m.Name == "test-model" {
			found = true
			if len(m.Capabilities) != 2 {
				t.Errorf("Expected 2 capabilities, got %d", len(m.Capabilities))
			}
		}
	}
	if !found {
		t.Error("Expected test-model to be saved")
	}
}

func TestScopedModelsConfig_GetModelForLevel(t *testing.T) {
	cfg := NewScopedModelsConfig()
	cfg.Models = []ScopedModel{
		{Name: "low-model", Thinking: ThinkingLow},
		{Name: "high-model", Thinking: ThinkingHigh},
	}

	model := cfg.GetModelForLevel(ThinkingLow)
	if model != "low-model" {
		t.Errorf("Expected 'low-model', got '%s'", model)
	}

	model = cfg.GetModelForLevel(ThinkingHigh)
	if model != "high-model" {
		t.Errorf("Expected 'high-model', got '%s'", model)
	}

	model = cfg.GetModelForLevel(ThinkingXHigh) // No match
	if model != cfg.Default {
		t.Errorf("Expected default '%s', got '%s'", cfg.Default, model)
	}
}

func TestScopedModelsConfig_CycleModels(t *testing.T) {
	cfg := NewScopedModelsConfig()
	cfg.Models = []ScopedModel{
		{Name: "first"},
		{Name: "second"},
		{Name: "third"},
	}

	// Forward cycle
	next := cfg.CycleModels("first", 1)
	if next != "second" {
		t.Errorf("Expected 'second', got '%s'", next)
	}

	next = cfg.CycleModels("second", 1)
	if next != "third" {
		t.Errorf("Expected 'third', got '%s'", next)
	}

	// Wrap around
	next = cfg.CycleModels("third", 1)
	if next != "first" {
		t.Errorf("Expected 'first' after wrap, got '%s'", next)
	}

	// Backward cycle
	prev := cfg.CycleModels("first", -1)
	if prev != "third" {
		t.Errorf("Expected 'third' for prev, got '%s'", prev)
	}
}

func TestScopedModelsConfig_GetCapabilities(t *testing.T) {
	cfg := NewScopedModelsConfig()
	cfg.Models = append(cfg.Models, ScopedModel{
		Name:       "test",
		Capabilities: []string{"tool-use", "multi-turn"},
	})

	caps := cfg.GetCapabilities("test")
	if len(caps) != 2 {
		t.Errorf("Expected 2 capabilities, got %d", len(caps))
	}

	caps = cfg.GetCapabilities("nonexistent")
	if caps != nil {
		t.Errorf("Expected nil for nonexistent model, got %v", caps)
	}
}

func TestScopedModelsConfig_GlobalPath(t *testing.T) {
	path := GlobalScopedModelsFile()
	if path == "" {
		t.Error("Expected non-empty path")
	}
}

func TestScopedModelsConfig_LocalPath(t *testing.T) {
	path := LocalScopedModelsFile("/test/project")
	expected := filepath.Join("/test/project", ".pi-go", "agent", "scoped-models.json")
	if path != expected {
		t.Errorf("Expected %s, got %s", expected, path)
	}
}

func TestScopedModelsConfig_InvalidPath(t *testing.T) {
	_, err := LoadScopedModels("/nonexistent/path.json")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}
