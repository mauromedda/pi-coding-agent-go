// ABOUTME: Shared helper functions for tool parameter extraction
// ABOUTME: Provides type-safe parameter accessors used by all tool implementations

package tools

import (
	"fmt"
	"math"

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
		if math.IsNaN(n) || math.IsInf(n, 0) || n > float64(math.MaxInt) || n < float64(math.MinInt) {
			return defaultVal
		}
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

// requireStringSliceParam extracts a required []string from a JSON-decoded []any.
func requireStringSliceParam(params map[string]any, key string) ([]string, error) {
	v, ok := params[key]
	if !ok {
		return nil, fmt.Errorf("missing required parameter %q", key)
	}
	raw, ok := v.([]any)
	if !ok {
		return nil, fmt.Errorf("parameter %q must be an array, got %T", key, v)
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("parameter %q must not be empty", key)
	}
	out := make([]string, 0, len(raw))
	for i, elem := range raw {
		s, ok := elem.(string)
		if !ok {
			return nil, fmt.Errorf("parameter %q[%d] must be a string, got %T", key, i, elem)
		}
		out = append(out, s)
	}
	return out, nil
}

// skipDirs contains directory names to skip during recursive file walks.
var skipDirs = map[string]bool{
	".git":         true,
	"vendor":       true,
	"node_modules": true,
	"__pycache__":  true,
	".venv":        true,
	".tox":         true,
	"dist":         true,
	"build":        true,
}

// shouldSkipDir reports whether a directory name should be skipped during walks.
func shouldSkipDir(name string) bool {
	return skipDirs[name]
}
