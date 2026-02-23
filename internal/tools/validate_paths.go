// ABOUTME: Validate paths tool: checks existence and type for a list of file paths
// ABOUTME: Returns per-path status (file/dir/not-found) and a summary count

package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/mauromedda/pi-coding-agent-go/internal/agent"
)

// NewValidatePathsTool creates a read-only tool that checks path existence.
func NewValidatePathsTool() *agent.AgentTool {
	return &agent.AgentTool{
		Name:        "validate_paths",
		Label:       "Validate Paths",
		Description: "Check whether a list of file paths exist and report their type (file, dir) and size.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"required": ["paths"],
			"properties": {
				"paths": {
					"type": "array",
					"items": {"type": "string"},
					"description": "List of file or directory paths to check"
				}
			}
		}`),
		ReadOnly: true,
		Execute:  executeValidatePaths,
	}
}

func executeValidatePaths(_ context.Context, _ string, params map[string]any, _ func(agent.ToolUpdate)) (agent.ToolResult, error) {
	paths, err := requireStringSliceParam(params, "paths")
	if err != nil {
		return errResult(err), nil
	}

	var b strings.Builder
	found := 0
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			fmt.Fprintf(&b, "%s: not found\n", p)
			continue
		}
		found++
		kind := "file"
		if info.IsDir() {
			kind = "dir"
		}
		fmt.Fprintf(&b, "%s: %s (%d bytes)\n", p, kind, info.Size())
	}
	fmt.Fprintf(&b, "\n%d/%d paths exist", found, len(paths))
	return agent.ToolResult{Content: b.String()}, nil
}
