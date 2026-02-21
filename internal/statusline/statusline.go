// ABOUTME: External command engine for custom status line content
// ABOUTME: Pipes JSON input to shell command, captures stdout, applies padding

package statusline

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Input contains the data piped to the external status line command as JSON.
type Input struct {
	CWD           string      `json:"cwd"`
	SessionID     string      `json:"session_id,omitempty"`
	Model         ModelInfo   `json:"model"`
	Workspace     string      `json:"workspace,omitempty"`
	Mode          string      `json:"mode"`
	GitBranch     string      `json:"git_branch,omitempty"`
	ContextWindow ContextInfo `json:"context_window"`
}

// ModelInfo describes the active model.
type ModelInfo struct {
	Name string `json:"name"`
	API  string `json:"api,omitempty"`
}

// ContextInfo tracks context window usage.
type ContextInfo struct {
	Used  int `json:"used"`
	Total int `json:"total"`
}

// Engine executes an external command to produce status line content.
type Engine struct {
	command string
	padding int
}

// New creates a status line engine with the given shell command and padding.
func New(command string, padding int) *Engine {
	return &Engine{
		command: command,
		padding: padding,
	}
}

// HasCommand reports whether an external command is configured.
func (e *Engine) HasCommand() bool {
	return e.command != ""
}

// Execute runs the configured command, piping the Input as JSON to stdin.
// Returns the trimmed stdout output with padding applied.
// Respects the provided context for cancellation; applies a 5-second default timeout.
func (e *Engine) Execute(ctx context.Context, input Input) (string, error) {
	if e.command == "" {
		return "", fmt.Errorf("no command configured")
	}

	// Apply default timeout if context has no deadline
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
	}

	data, err := json.Marshal(input)
	if err != nil {
		return "", fmt.Errorf("marshaling input: %w", err)
	}

	cmd := exec.CommandContext(ctx, "sh", "-c", e.command)
	cmd.Stdin = bytes.NewReader(data)

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("running status line command: %w", err)
	}

	result := strings.TrimSpace(stdout.String())

	if e.padding > 0 {
		result = strings.Repeat(" ", e.padding) + result
	}

	return result, nil
}
