// ABOUTME: Find tool: discovers files matching a glob pattern in a directory tree
// ABOUTME: Supports ** globs, mod-time sorting (newest first), head_limit; uses rg or stdlib

package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/mauromedda/pi-coding-agent-go/internal/agent"
)

// NewFindTool creates a read-only tool for discovering files by glob pattern.
func NewFindTool(hasRg bool) *agent.AgentTool {
	return &agent.AgentTool{
		Name:  "find",
		Label: "Find Files",
		Description: `Fast file pattern matching tool that works with any codebase size.

Usage:
- Supports glob patterns like "**/*.js" or "src/**/*.ts"
- Returns matching file paths sorted by modification time (newest first)
- Use this tool when you need to find files by name patterns

Features:
- ** glob patterns: Match files in any subdirectory depth
- Modification time sorting: Results are sorted newest-first
- head_limit: Limit the number of results returned
- Automatically skips common non-source directories (.git, node_modules, vendor, etc.)`,
		Parameters: json.RawMessage(`{
			"type": "object",
			"required": ["pattern"],
			"properties": {
				"pattern":    {"type": "string", "description": "Glob pattern to match file names (supports ** for recursive matching)"},
				"path":       {"type": "string", "description": "Directory to search (default: current dir)"},
				"head_limit": {"type": "integer", "description": "Limit output to first N files (0 = unlimited)"}
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
		headLimit := intParam(params, "head_limit", 0)

		var output string
		if hasRg {
			output, err = findWithRg(ctx, pattern, path, headLimit)
		} else {
			output, err = findBuiltin(pattern, path, headLimit)
		}
		if err != nil {
			return errResult(fmt.Errorf("find: %w", err)), nil
		}

		output = truncateOutput(output, maxReadOutput)
		return agent.ToolResult{Content: output}, nil
	}
}

// findWithRg uses ripgrep's --files mode with a glob filter, sorted by mod time.
func findWithRg(ctx context.Context, pattern, path string, headLimit int) (string, error) {
	args := []string{"--files", "--glob", pattern, "--sortr", "modified", path}
	cmd := exec.CommandContext(ctx, "rg", args...)
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

	raw := strings.TrimRight(stdout.String(), "\n")
	if headLimit > 0 {
		lines := strings.Split(raw, "\n")
		if headLimit < len(lines) {
			lines = lines[:headLimit]
		}
		return strings.Join(lines, "\n") + "\n", nil
	}

	return raw + "\n", nil
}
