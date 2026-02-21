// ABOUTME: Shell command executor for hook definitions
// ABOUTME: Pipes HookInput as JSON to stdin, parses HookOutput from stdout

package hooks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

const hookTimeout = 10 * time.Second

// runHookCommand executes a shell command with HookInput piped to stdin as JSON.
// It enforces a 10s timeout and kills the process group on timeout.
// Non-zero exit code sets Blocked=true in the output.
func runHookCommand(ctx context.Context, command string, input HookInput) (HookOutput, error) {
	ctx, cancel := context.WithTimeout(ctx, hookTimeout)
	defer cancel()

	inputJSON, err := json.Marshal(input)
	if err != nil {
		return HookOutput{}, fmt.Errorf("marshal hook input: %w", err)
	}

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Stdin = bytes.NewReader(inputJSON)
	setProcGroup(cmd)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	cmd.Cancel = func() error {
		return killProcGroup(cmd)
	}

	runErr := cmd.Run()

	// Context deadline exceeded means timeout.
	if ctx.Err() != nil {
		return HookOutput{}, fmt.Errorf("hook command timed out after %v: %w", hookTimeout, ctx.Err())
	}

	var out HookOutput
	if stdout.Len() > 0 {
		if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
			return HookOutput{}, fmt.Errorf("parse hook output (raw: %q): %w", stdout.String(), err)
		}
	}

	// Non-zero exit code means the hook is blocking execution.
	if runErr != nil {
		out.Blocked = true
		if out.Message == "" {
			out.Message = fmt.Sprintf("hook command exited with error: %v", runErr)
		}
	}

	return out, nil
}
