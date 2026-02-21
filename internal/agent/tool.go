// ABOUTME: Tool argument validation and parsing utilities
// ABOUTME: Validates required parameters and deserialises raw JSON into maps

package agent

import (
	"encoding/json"
	"fmt"
)

// ValidateToolArgs checks that the provided args satisfy the tool's JSON Schema.
// Currently validates required fields defined in the schema; returns an error
// listing any missing required parameter.
func ValidateToolArgs(tool *AgentTool, args map[string]any) error {
	if tool.Parameters == nil {
		return nil
	}

	var schema struct {
		Required []string `json:"required"`
	}
	if err := json.Unmarshal(tool.Parameters, &schema); err != nil {
		return fmt.Errorf("parsing tool %s schema: %w", tool.Name, err)
	}

	for _, req := range schema.Required {
		if _, ok := args[req]; !ok {
			return fmt.Errorf("missing required parameter %q for tool %s", req, tool.Name)
		}
	}

	return nil
}

// ParseToolArgs deserialises raw JSON into a string-keyed map.
// Returns an empty map (not nil) when raw is empty or null.
func ParseToolArgs(raw json.RawMessage) (map[string]any, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return make(map[string]any), nil
	}

	var args map[string]any
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, fmt.Errorf("parsing tool arguments: %w", err)
	}

	return args, nil
}
