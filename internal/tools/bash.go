// ABOUTME: Bash tool: executes shell commands via /bin/bash -c
// ABOUTME: Captures combined stdout+stderr; respects configurable timeout

package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/mauromedda/pi-coding-agent-go/internal/agent"
)

const defaultBashTimeoutMs = 120_000

// NewBashTool creates a tool that executes shell commands.
func NewBashTool() *agent.AgentTool {
	return &agent.AgentTool{
		Name:        "bash",
		Label:       "Run Shell Command",
		Description: "Execute a shell command via /bin/bash -c. Captures stdout and stderr.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"required": ["command"],
			"properties": {
				"command":    {"type": "string", "description": "Shell command to execute"},
				"timeout_ms": {"type": "integer", "description": "Timeout in milliseconds (default 120000)"}
			}
		}`),
		ReadOnly: false,
		Execute:  executeBash,
	}
}

func executeBash(ctx context.Context, _ string, params map[string]any, onUpdate func(agent.ToolUpdate)) (agent.ToolResult, error) {
	command, err := requireStringParam(params, "command")
	if err != nil {
		return errResult(err), nil
	}

	timeoutMs := intParam(params, "timeout_ms", defaultBashTimeoutMs)
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	result, err := runBashCommand(ctx, command)
	if err != nil {
		return errResult(fmt.Errorf("executing command: %w", err)), nil
	}

	if onUpdate != nil {
		onUpdate(agent.ToolUpdate{Output: result})
	}

	result = truncateOutput(result, maxReadOutput)
	return agent.ToolResult{Content: result}, nil
}

// runBashCommand executes a command string and returns combined stdout+stderr.
func runBashCommand(ctx context.Context, command string) (string, error) {
	bashPath, err := exec.LookPath("bash")
	if err != nil {
		return "", fmt.Errorf("bash not found on PATH: %w", err)
	}
	cmd := exec.CommandContext(ctx, bashPath, "-c", command)

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	err = cmd.Run()

	output := buf.String()
	if err != nil {
		if ctx.Err() != nil {
			return output, fmt.Errorf("command timed out: %w", ctx.Err())
		}
		// Return output even on non-zero exit; include exit error.
		return output + "\n" + err.Error(), nil
	}

	return output, nil
}
