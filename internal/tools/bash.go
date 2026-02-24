// ABOUTME: Bash tool: executes shell commands via /bin/bash -c
// ABOUTME: Captures combined stdout+stderr; respects configurable timeout

package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"time"

	"github.com/mauromedda/pi-coding-agent-go/internal/agent"
)

const (
	defaultBashTimeoutMs = 120_000
	maxBashOutput        = 10 * 1024 * 1024 // 10MB
)

var errOutputLimitExceeded = errors.New("output limit exceeded")

// limitedWriter wraps an io.Writer and stops accepting data after limit bytes.
// When the limit is hit, Write returns errOutputLimitExceeded.
type limitedWriter struct {
	w        io.Writer
	limit    int
	written  int
	exceeded bool
}

func (lw *limitedWriter) Write(p []byte) (int, error) {
	remaining := lw.limit - lw.written
	if remaining <= 0 {
		lw.exceeded = true
		return 0, errOutputLimitExceeded
	}

	if len(p) > remaining {
		n, err := lw.w.Write(p[:remaining])
		lw.written += n
		lw.exceeded = true
		if err != nil {
			return n, err
		}
		return n, errOutputLimitExceeded
	}

	n, err := lw.w.Write(p)
	lw.written += n
	return n, err
}

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

	// Sanitize and validate the command for security
	sanitizedCommand := sanitizeBashCommand(command)
	if err := validateBashCommand(sanitizedCommand); err != nil {
		return errResult(fmt.Errorf("command validation failed: %w", err)), nil
	}

	timeoutMs := intParam(params, "timeout_ms", defaultBashTimeoutMs)
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	result, err := runBashCommand(ctx, sanitizedCommand)
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
// Output is capped at maxBashOutput bytes; the process is killed if exceeded.
func runBashCommand(ctx context.Context, command string) (string, error) {
	bashPath, err := exec.LookPath("bash")
	if err != nil {
		return "", fmt.Errorf("bash not found on PATH: %w", err)
	}
	cmd := exec.CommandContext(ctx, bashPath, "-c", command)

	// Set restricted environment for security
	cmd.Env = restrictedEnvironment()

	var buf bytes.Buffer
	lw := &limitedWriter{w: &buf, limit: maxBashOutput}
	cmd.Stdout = lw
	cmd.Stderr = lw

	err = cmd.Run()

	output := buf.String()

	if lw.exceeded {
		output += "\n... [output truncated: exceeded 10MB limit]"
	}

	if err != nil {
		if ctx.Err() != nil {
			return output, fmt.Errorf("command timed out: %w", ctx.Err())
		}
		// Output limit exceeded kills the process, which is expected.
		if lw.exceeded {
			return output, nil
		}
		// Return output even on non-zero exit; include exit error.
		return output + "\n" + err.Error(), nil
	}

	return output, nil
}
