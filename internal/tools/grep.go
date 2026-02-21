// ABOUTME: Grep tool: searches file contents using ripgrep or built-in fallback
// ABOUTME: Returns matching lines with context; auto-selects rg vs stdlib regex

package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/mauromedda/pi-coding-agent-go/internal/agent"
)

// NewGrepTool creates a read-only tool that searches file contents.
func NewGrepTool(hasRg bool) *agent.AgentTool {
	return &agent.AgentTool{
		Name:        "grep",
		Label:       "Search File Contents",
		Description: "Search for a regex pattern in file contents. Uses ripgrep if available.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"required": ["pattern"],
			"properties": {
				"pattern": {"type": "string", "description": "Regex pattern to search for"},
				"path":    {"type": "string", "description": "Directory or file to search (default: current dir)"},
				"include": {"type": "string", "description": "Glob pattern to filter files (e.g. *.go)"}
			}
		}`),
		ReadOnly: true,
		Execute:  makeGrepExecutor(hasRg),
	}
}

func makeGrepExecutor(hasRg bool) func(context.Context, string, map[string]any, func(agent.ToolUpdate)) (agent.ToolResult, error) {
	return func(ctx context.Context, _ string, params map[string]any, _ func(agent.ToolUpdate)) (agent.ToolResult, error) {
		pattern, err := requireStringParam(params, "pattern")
		if err != nil {
			return errResult(err), nil
		}

		path := stringParam(params, "path", ".")
		include := stringParam(params, "include", "")

		var output string
		if hasRg {
			output, err = grepWithRg(ctx, pattern, path, include)
		} else {
			output, err = grepBuiltin(pattern, path, include)
		}
		if err != nil {
			return errResult(fmt.Errorf("grep: %w", err)), nil
		}

		output = truncateOutput(output, maxReadOutput)
		return agent.ToolResult{Content: output}, nil
	}
}

// grepWithRg runs ripgrep with JSON output for structured results.
func grepWithRg(ctx context.Context, pattern, path, include string) (string, error) {
	args := []string{"--json", "-e", pattern}
	if include != "" {
		args = append(args, "--glob", include)
	}
	args = append(args, path)

	cmd := exec.CommandContext(ctx, "rg", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// rg exits 1 when no matches found; that's not an error.
		if cmd.ProcessState != nil && cmd.ProcessState.ExitCode() == 1 {
			return "no matches found", nil
		}
		return "", fmt.Errorf("ripgrep failed: %s: %w", stderr.String(), err)
	}

	return stdout.String(), nil
}
