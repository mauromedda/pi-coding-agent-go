// ABOUTME: Model spec parser: extracts thinking level from colon-separated model IDs
// ABOUTME: Handles provider:model:thinking format with recursive last-colon split

package config

import (
	"regexp"
	"strings"

	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

// validThinkingLevels lists recognized thinking level suffixes.
var validThinkingLevels = map[string]bool{
	"off":    true,
	"low":    true,
	"medium": true,
	"high":   true,
}

// dateSuffixRe matches model IDs ending with a YYYYMMDD date suffix.
var dateSuffixRe = regexp.MustCompile(`-\d{8}$`)

// ParseModelSpec splits a model input into model ID and optional thinking level.
// Format: "model-id" or "model-id:thinking" or "provider:model:thinking".
// Uses recursive last-colon split: if the part after the last colon is a valid
// thinking level, it is extracted; otherwise the full string is the model ID.
func ParseModelSpec(input string) (modelID string, thinkingLevel string, err error) {
	if input == "" {
		return "", "", nil
	}

	lastColon := strings.LastIndex(input, ":")
	if lastColon < 0 {
		return input, "", nil
	}

	suffix := strings.ToLower(input[lastColon+1:])
	if validThinkingLevels[suffix] {
		return input[:lastColon], suffix, nil
	}

	return input, "", nil
}

// ResolveModelWithSpec wraps ResolveModel and ParseModelSpec: resolves the
// model and extracts an optional thinking level from the input string.
func ResolveModelWithSpec(input string) (*ai.Model, string, error) {
	modelID, thinking, err := ParseModelSpec(input)
	if err != nil {
		return nil, "", err
	}

	model, err := ResolveModel(modelID)
	if err != nil {
		return nil, "", err
	}

	return model, thinking, nil
}

// IsAlias returns true if the model ID does not end with a YYYYMMDD date suffix.
// Aliased models (like "gpt-4o") are typically short names without pinned versions.
func IsAlias(id string) bool {
	return !dateSuffixRe.MatchString(id)
}
