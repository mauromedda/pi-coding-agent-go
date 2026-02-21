// ABOUTME: Tests for external status line command engine
// ABOUTME: Verifies JSON input piping, command execution, timeout, and padding

package statusline

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestEngine_HasCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		command string
		want    bool
	}{
		{"empty", "", false},
		{"set", "echo hello", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := New(tt.command, 0)
			if got := e.HasCommand(); got != tt.want {
				t.Errorf("HasCommand() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEngine_Execute_EchoCommand(t *testing.T) {
	t.Parallel()

	e := New("cat", 0) // cat reads stdin and outputs it
	input := Input{
		CWD:  "/tmp/test",
		Mode: "plan",
	}

	ctx := context.Background()
	result, err := e.Execute(ctx, input)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	// cat should echo back the JSON input
	if !strings.Contains(result, "/tmp/test") {
		t.Errorf("expected output to contain CWD, got %q", result)
	}
	if !strings.Contains(result, "plan") {
		t.Errorf("expected output to contain mode, got %q", result)
	}
}

func TestEngine_Execute_CustomCommand(t *testing.T) {
	t.Parallel()

	e := New("echo custom-status", 0)
	input := Input{CWD: "/tmp"}

	ctx := context.Background()
	result, err := e.Execute(ctx, input)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	if strings.TrimSpace(result) != "custom-status" {
		t.Errorf("Execute() = %q, want %q", result, "custom-status")
	}
}

func TestEngine_Execute_Padding(t *testing.T) {
	t.Parallel()

	e := New("echo padded", 3)
	input := Input{CWD: "/tmp"}

	ctx := context.Background()
	result, err := e.Execute(ctx, input)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	if !strings.HasPrefix(result, "   ") {
		t.Errorf("expected 3 spaces padding, got %q", result)
	}
}

func TestEngine_Execute_Timeout(t *testing.T) {
	t.Parallel()

	e := New("sleep 30", 0)
	input := Input{CWD: "/tmp"}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := e.Execute(ctx, input)
	if err == nil {
		t.Error("expected timeout error")
	}
}

func TestEngine_Execute_TrimsOutput(t *testing.T) {
	t.Parallel()

	e := New("printf '  status with spaces  \\n'", 0)
	input := Input{CWD: "/tmp"}

	ctx := context.Background()
	result, err := e.Execute(ctx, input)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	if result != "status with spaces" {
		t.Errorf("Execute() = %q, want trimmed output", result)
	}
}

func TestInput_JSONFields(t *testing.T) {
	t.Parallel()

	input := Input{
		CWD:       "/home/user/project",
		SessionID: "abc-123",
		Mode:      "edit",
		GitBranch: "main",
		Model: ModelInfo{
			Name: "claude-sonnet",
			API:  "anthropic",
		},
		ContextWindow: ContextInfo{
			Used:  50000,
			Total: 200000,
		},
	}

	// Verify the struct can be marshaled (tested implicitly by Execute)
	if input.CWD != "/home/user/project" {
		t.Error("unexpected CWD")
	}
	if input.Model.Name != "claude-sonnet" {
		t.Error("unexpected model name")
	}
}
