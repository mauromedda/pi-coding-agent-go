// ABOUTME: Tests for the bash tool: command execution, timeout, and stderr capture
// ABOUTME: Uses simple shell commands to validate execution behaviour

package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/mauromedda/pi-coding-agent-go/internal/agent"
)

func TestBashTool_SimpleCommand(t *testing.T) {
	t.Parallel()

	tool := NewBashTool()
	result, err := tool.Execute(context.Background(), "id1", map[string]any{
		"command": "echo hello",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %s", result.Content)
	}
	if strings.TrimSpace(result.Content) != "hello" {
		t.Errorf("got %q, want %q", result.Content, "hello\n")
	}
}

func TestBashTool_StderrCapture(t *testing.T) {
	t.Parallel()

	tool := NewBashTool()
	result, err := tool.Execute(context.Background(), "id1", map[string]any{
		"command": "echo error_output >&2",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.Content, "error_output") {
		t.Errorf("expected stderr in output, got %q", result.Content)
	}
}

func TestBashTool_NonZeroExit(t *testing.T) {
	t.Parallel()

	tool := NewBashTool()
	result, err := tool.Execute(context.Background(), "id1", map[string]any{
		"command": "echo 'some output' && exit 1",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Non-zero exit is captured but not treated as an Execute error.
	if !strings.Contains(result.Content, "some output") {
		t.Errorf("expected output in result, got %q", result.Content)
	}
	if !strings.Contains(result.Content, "exit status 1") {
		t.Errorf("expected exit status in result, got %q", result.Content)
	}
}

func TestBashTool_Timeout(t *testing.T) {
	t.Parallel()

	tool := NewBashTool()
	result, err := tool.Execute(context.Background(), "id1", map[string]any{
		"command":    "sleep 10",
		"timeout_ms": float64(200), // 200ms timeout
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError for timed-out command")
	}
	if !strings.Contains(result.Content, "timed out") {
		t.Errorf("expected 'timed out' in error, got %q", result.Content)
	}
}

func TestBashTool_MissingCommand(t *testing.T) {
	t.Parallel()

	tool := NewBashTool()
	result, err := tool.Execute(context.Background(), "id1", map[string]any{}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError for missing command")
	}
}

func TestBashTool_UpdateCallback(t *testing.T) {
	t.Parallel()

	var received string
	tool := NewBashTool()
	_, err := tool.Execute(context.Background(), "id1", map[string]any{
		"command": "echo callback_test",
	}, func(u agent.ToolUpdate) {
		received = u.Output
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(received, "callback_test") {
		t.Errorf("expected update callback with output, got %q", received)
	}
}
