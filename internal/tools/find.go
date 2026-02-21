// ABOUTME: Find tool: discovers files matching a glob pattern in a directory tree
// ABOUTME: Uses ripgrep --files with glob when available; falls back to stdlib

package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/mauromedda/pi-coding-agent-go/internal/agent"
)

// NewFindTool creates a read-only tool for discovering files by glob pattern.
func NewFindTool(hasRg bool) *agent.AgentTool {
	return &agent.AgentTool{
		Name:        "find",
		Label:       "Find Files",
		Description: "Find files matching a glob pattern. Uses ripgrep if available.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"required": ["pattern"],
			"properties": {
				"pattern": {"type": "string", "description": "Glob pattern to match file names"},
				"path":    {"type": "string", "description": "Directory to search (default: current dir)"}
			}
		}`),
		ReadOnly: true,
		Execute:  makeFindExecutor(hasRg),
	}
}

func makeFindExecutor(hasRg bool) func(context.Context, string, map[string]any, func(agent.ToolUpdate)) (agent.ToolResult, error) {
	return func(ctx context.Context, _ string, params map[string]any, _ func(agent.ToolUpdate)) (agent.ToolResult, error) {
		pattern, err := requireStringParam(params, "pattern")
		if err != nil {
			return errResult(err), nil
		}

		path := stringParam(params, "path", ".")

		var output string
		if hasRg {
			output, err = findWithRg(ctx, pattern, path)
		} else {
			output, err = findBuiltin(pattern, path)
		}
		if err != nil {
			return errResult(fmt.Errorf("find: %w", err)), nil
		}

		output = truncateOutput(output, maxReadOutput)
		return agent.ToolResult{Content: output}, nil
	}
}

// findWithRg uses ripgrep's --files mode with a glob filter.
func findWithRg(ctx context.Context, pattern, path string) (string, error) {
	cmd := exec.CommandContext(ctx, "rg", "--files", "--glob", pattern, path)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		if cmd.ProcessState != nil && cmd.ProcessState.ExitCode() == 1 {
			return "no files found", nil
		}
		return "", fmt.Errorf("ripgrep --files failed: %s: %w", stderr.String(), err)
	}

	if stdout.Len() == 0 {
		return "no files found", nil
	}
	return stdout.String(), nil
}
