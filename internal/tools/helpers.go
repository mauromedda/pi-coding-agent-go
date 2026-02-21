// ABOUTME: Shared helper functions for tool parameter extraction
// ABOUTME: Provides type-safe parameter accessors used by all tool implementations

package tools

import (
	"fmt"

	"github.com/mauromedda/pi-coding-agent-go/internal/agent"
)

// requireStringParam extracts a required string parameter from the args map.
func requireStringParam(params map[string]any, key string) (string, error) {
	v, ok := params[key]
	if !ok {
		return "", fmt.Errorf("missing required parameter %q", key)
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("parameter %q must be a string, got %T", key, v)
	}
	return s, nil
}

// stringParam extracts an optional string parameter with a default value.
func stringParam(params map[string]any, key, defaultVal string) string {
	v, ok := params[key]
	if !ok {
		return defaultVal
	}
	s, ok := v.(string)
	if !ok {
		return defaultVal
	}
	return s
}

// intParam extracts an optional integer parameter with a default value.
// Handles both float64 (from JSON unmarshal) and int types.
func intParam(params map[string]any, key string, defaultVal int) int {
	v, ok := params[key]
	if !ok {
		return defaultVal
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	default:
		return defaultVal
	}
}

// boolParam extracts an optional boolean parameter with a default value.
func boolParam(params map[string]any, key string, defaultVal bool) bool {
	v, ok := params[key]
	if !ok {
		return defaultVal
	}
	b, ok := v.(bool)
	if !ok {
		return defaultVal
	}
	return b
}

// errResult builds a ToolResult that signals an error.
func errResult(err error) agent.ToolResult {
	return agent.ToolResult{Content: err.Error(), IsError: true}
}
