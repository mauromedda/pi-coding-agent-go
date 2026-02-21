// ABOUTME: Tests for the bash tool: command execution, timeout, and stderr capture
// ABOUTME: Uses simple shell commands to validate execution behaviour

package tools

import (
	"bytes"
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

func TestBashTool_OutputExceedsLimit(t *testing.T) {
	t.Parallel()

	tool := NewBashTool()
	// Generate ~11MB of output (exceeds maxBashOutput of 10MB).
	// dd writes 11*1024*1024 bytes of zero, tr converts to 'A'.
	result, err := tool.Execute(context.Background(), "id1", map[string]any{
		"command":    "dd if=/dev/zero bs=1048576 count=11 2>/dev/null | tr '\\0' 'A'",
		"timeout_ms": float64(30000),
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "truncated") {
		t.Error("expected truncation notice in output")
	}
	// Output should be at most maxBashOutput + truncation notice length.
	if len(result.Content) > 10*1024*1024+100 {
		t.Errorf("output too large: %d bytes", len(result.Content))
	}
}

func TestLimitedWriter_Limit(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	lw := &limitedWriter{w: &buf, limit: 10}

	n, err := lw.Write([]byte("hello"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 5 {
		t.Errorf("wrote %d; want 5", n)
	}

	// Write more to exceed limit.
	n, err = lw.Write([]byte("world!!!"))
	if err != errOutputLimitExceeded {
		t.Errorf("expected errOutputLimitExceeded; got %v", err)
	}
	// Only 5 more bytes should be accepted (limit=10, already wrote 5).
	if n != 5 {
		t.Errorf("wrote %d; want 5", n)
	}
	if !lw.exceeded {
		t.Error("expected exceeded = true")
	}
	if buf.String() != "helloworld" {
		t.Errorf("buf = %q; want %q", buf.String(), "helloworld")
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
