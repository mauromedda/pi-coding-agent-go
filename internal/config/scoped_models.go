// ABOUTME: Scoped models configuration and cycling
// ABOUTME: Supports cycling between models based on scope (off, minimal, low, medium, high, xhigh)

package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// ThinkingLevel represents the thinking level for model behavior
type ThinkingLevel int

const (
	ThinkingOff     ThinkingLevel = iota // Off - minimal thinking
	ThinkingMinimal                      // Minimal
	ThinkingLow                          // Low
	ThinkingMedium                       // Medium
	ThinkingHigh                         // High
	ThinkingXHigh                        // XHigh
)

func (tl ThinkingLevel) String() string {
	switch tl {
	case ThinkingOff:
		return "off"
	case ThinkingMinimal:
		return "minimal"
	case ThinkingLow:
		return "low"
	case ThinkingMedium:
		return "medium"
	case ThinkingHigh:
		return "high"
	case ThinkingXHigh:
		return "xhigh"
	default:
		return "off"
	}
}

func ThinkingLevelFromString(s string) ThinkingLevel {
	switch s {
	case "off":
		return ThinkingOff
	case "minimal":
		return ThinkingMinimal
	case "low":
		return ThinkingLow
	case "medium":
		return ThinkingMedium
	case "high":
		return ThinkingHigh
	case "xhigh":
		return ThinkingXHigh
	default:
		return ThinkingOff
	}
}

// Index returns the index for this thinking level (0-5)
func (tl ThinkingLevel) Index() int {
	return int(tl)
}

// FromIndex converts an index to ThinkingLevel
func ThinkingLevelFromIndex(idx int) ThinkingLevel {
	switch idx {
	case 0:
		return ThinkingOff
	case 1:
		return ThinkingMinimal
	case 2:
		return ThinkingLow
	case 3:
		return ThinkingMedium
	case 4:
		return ThinkingHigh
	case 5:
		return ThinkingXHigh
	default:
		return ThinkingOff
	}
}

// ScopedModel represents a model configuration with scope level
type ScopedModel struct {
	Name       string        `json:"name"`
	Thinking   ThinkingLevel `json:"thinking,omitempty"`
	Provider   string        `json:"provider,omitempty"`
	Capabilities []string    `json:"capabilities,omitempty"`
}

// ScopedModelsConfig holds scoped model configurations
type ScopedModelsConfig struct {
	Models   []ScopedModel `json:"models"`
	Default  string        `json:"default,omitempty"`
}

// NewScopedModelsConfig creates a new config with defaults
func NewScopedModelsConfig() *ScopedModelsConfig {
	return &ScopedModelsConfig{
		Models: []ScopedModel{
			{Name: "claude-3-5-sonnet", Thinking: ThinkingLow},
			{Name: "claude-3-opus", Thinking: ThinkingMedium},
			{Name: "gpt-4o", Thinking: ThinkingHigh},
			{Name: "gemini-1.5-pro", Thinking: ThinkingMedium},
		},
		Default: "claude-3-5-sonnet",
	}
}

// LoadScopedModels loads scoped models from a file
func LoadScopedModels(path string) (*ScopedModelsConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg ScopedModelsConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// SaveScopedModels saves scoped models to a file
func (cfg *ScopedModelsConfig) SaveScopedModels(path string) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o644)
}

// GetModelForLevel returns the model name for a given thinking level
func (cfg *ScopedModelsConfig) GetModelForLevel(level ThinkingLevel) string {
	for _, m := range cfg.Models {
		if m.Thinking == level {
			return m.Name
		}
	}
	return cfg.Default
}

// CycleModels cycles through models based on level
func (cfg *ScopedModelsConfig) CycleModels(current string, direction int) string {
	if len(cfg.Models) == 0 {
		return current
	}

	// Find current index
	currentIdx := -1
	for i, m := range cfg.Models {
		if m.Name == current {
			currentIdx = i
			break
		}
	}

	if currentIdx == -1 {
		return cfg.Default
	}

	// Cycle with wrap-around
	newIdx := currentIdx + direction
	if newIdx < 0 {
		newIdx = len(cfg.Models) - 1
	} else if newIdx >= len(cfg.Models) {
		newIdx = 0
	}

	return cfg.Models[newIdx].Name
}

// GetCapabilities returns capabilities for a model
func (cfg *ScopedModelsConfig) GetCapabilities(modelName string) []string {
	for _, m := range cfg.Models {
		if m.Name == modelName {
			return m.Capabilities
		}
	}
	return nil
}

// GlobalScopedModelsFile returns the path to global scoped models file
func GlobalScopedModelsFile() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".pi-go", "agent", "scoped-models.json")
}

// LocalScopedModelsFile returns the path to local scoped models file
func LocalScopedModelsFile(projectRoot string) string {
	return filepath.Join(projectRoot, ".pi-go", "agent", "scoped-models.json")
}
