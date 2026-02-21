// ABOUTME: Write-file tool: creates or overwrites files with given content
// ABOUTME: Automatically creates parent directories via os.MkdirAll

package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mauromedda/pi-coding-agent-go/internal/agent"
)

// NewWriteTool creates a tool that writes content to a file.
func NewWriteTool() *agent.AgentTool {
	return &agent.AgentTool{
		Name:        "write",
		Label:       "Write File",
		Description: "Write content to a file, creating parent directories if needed.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"required": ["path", "content"],
			"properties": {
				"path":    {"type": "string", "description": "Absolute path to the file"},
				"content": {"type": "string", "description": "Content to write"}
			}
		}`),
		ReadOnly: false,
		Execute:  executeWrite,
	}
}

func executeWrite(_ context.Context, _ string, params map[string]any, _ func(agent.ToolUpdate)) (agent.ToolResult, error) {
	path, err := requireStringParam(params, "path")
	if err != nil {
		return errResult(err), nil
	}

	content, err := requireStringParam(params, "content")
	if err != nil {
		return errResult(err), nil
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return errResult(fmt.Errorf("creating directory %s: %w", dir, err)), nil
	}

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return errResult(fmt.Errorf("writing file %s: %w", path, err)), nil
	}

	return agent.ToolResult{Content: fmt.Sprintf("wrote %d bytes to %s", len(content), path)}, nil
}
