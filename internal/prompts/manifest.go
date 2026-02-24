// ABOUTME: Manifest parsing for versioned prompt configuration
// ABOUTME: Defines composition order, variables, and model compatibility

package prompts

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Manifest describes a prompt version's configuration.
type Manifest struct {
	Version          string            `yaml:"version"`
	Description      string            `yaml:"description"`
	CompatibleModels []string          `yaml:"compatible_models"`
	CompositionOrder []string          `yaml:"composition_order"`
	Variables        map[string]string `yaml:"variables"`
}

// LoadManifest reads a manifest from YAML bytes.
func LoadManifest(data []byte) (*Manifest, error) {
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	return &m, nil
}

// IsCompatible checks if a model ID matches any compatible pattern.
// Patterns use simple suffix wildcard: "claude-*" matches any string starting with "claude-".
func (m *Manifest) IsCompatible(modelID string) bool {
	if modelID == "" {
		return false
	}
	for _, pattern := range m.CompatibleModels {
		if before, ok := strings.CutSuffix(pattern, "*"); ok {
			prefix := before
			if strings.HasPrefix(modelID, prefix) {
				return true
			}
		} else if pattern == modelID {
			return true
		}
	}
	return false
}
