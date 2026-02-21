// ABOUTME: Tests for the slash command registry and dispatch
// ABOUTME: Covers all 6 core commands, unknown command error, and IsCommand detection

package commands

import (
	"strings"
	"testing"
)

// testContext creates a CommandContext with callback tracking for test assertions.
func testContext() (*CommandContext, *testCallbacks) {
	cb := &testCallbacks{}
	ctx := &CommandContext{
		Model:       "claude-sonnet",
		Mode:        "EDIT",
		Version:     "0.1.0",
		CWD:         "/tmp/project",
		TotalCost:   1.23,
		TotalTokens: 4567,
		Messages:    12,
		SetModel: func(name string) {
			cb.modelSet = name
		},
		ClearHistory: func() {
			cb.clearCalled = true
		},
		CompactFn: func() string {
			cb.compactCalled = true
			return "Conversation compacted to 3 messages."
		},
	}
	return ctx, cb
}

type testCallbacks struct {
	modelSet      string
	clearCalled   bool
	compactCalled bool
}

func TestRegistry_CoreCommandsRegistered(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()

	expected := []string{"clear", "compact", "config", "help", "model", "status"}
	for _, name := range expected {
		cmd, ok := reg.Get(name)
		if !ok {
			t.Errorf("command %q not found in registry", name)
			continue
		}
		if cmd.Name != name {
			t.Errorf("expected Name=%q, got %q", name, cmd.Name)
		}
		if cmd.Description == "" {
			t.Errorf("command %q has empty description", name)
		}
		if cmd.Execute == nil {
			t.Errorf("command %q has nil Execute", name)
		}
	}

	// Verify List returns exactly 6, sorted.
	all := reg.List()
	if len(all) != len(expected) {
		t.Fatalf("expected %d commands, got %d", len(expected), len(all))
	}
	for i, cmd := range all {
		if cmd.Name != expected[i] {
			t.Errorf("List()[%d]: expected %q, got %q", i, expected[i], cmd.Name)
		}
	}
}

func TestDispatch_Clear(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, cb := testContext()

	result, err := reg.Dispatch(ctx, "/clear")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cb.clearCalled {
		t.Error("ClearHistory was not called")
	}
	if !strings.Contains(result, "cleared") {
		t.Errorf("expected result to contain 'cleared', got %q", result)
	}
}

func TestDispatch_Compact(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, cb := testContext()

	result, err := reg.Dispatch(ctx, "/compact")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cb.compactCalled {
		t.Error("CompactFn was not called")
	}
	if !strings.Contains(result, "compacted") {
		t.Errorf("expected result to contain 'compacted', got %q", result)
	}
}

func TestDispatch_Config(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, _ := testContext()

	result, err := reg.Dispatch(ctx, "/config")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{"claude-sonnet", "EDIT", "/tmp/project", "0.1.0"} {
		if !strings.Contains(result, want) {
			t.Errorf("expected config output to contain %q, got:\n%s", want, result)
		}
	}
}

func TestDispatch_Help(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, _ := testContext()

	result, err := reg.Dispatch(ctx, "/help")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Help must list all commands.
	for _, name := range []string{"/clear", "/compact", "/config", "/help", "/model", "/status"} {
		if !strings.Contains(result, name) {
			t.Errorf("help output missing command %q, got:\n%s", name, result)
		}
	}
}

func TestDispatch_Model_Get(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, cb := testContext()

	result, err := reg.Dispatch(ctx, "/model")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "claude-sonnet") {
		t.Errorf("expected current model in output, got %q", result)
	}
	if cb.modelSet != "" {
		t.Error("SetModel should not have been called without an argument")
	}
}

func TestDispatch_Model_Set(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, cb := testContext()

	result, err := reg.Dispatch(ctx, "/model gpt-4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cb.modelSet != "gpt-4" {
		t.Errorf("expected SetModel called with 'gpt-4', got %q", cb.modelSet)
	}
	if !strings.Contains(result, "gpt-4") {
		t.Errorf("expected confirmation to contain 'gpt-4', got %q", result)
	}
}

func TestDispatch_Status(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, _ := testContext()

	result, err := reg.Dispatch(ctx, "/status")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{"claude-sonnet", "EDIT", "12", "4567", "1.23"} {
		if !strings.Contains(result, want) {
			t.Errorf("expected status output to contain %q, got:\n%s", want, result)
		}
	}
}

func TestDispatch_Unknown(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, _ := testContext()

	_, err := reg.Dispatch(ctx, "/nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown command")
	}
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("expected error to mention command name, got %q", err.Error())
	}
}

func TestIsCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"slash help", "/help", true},
		{"slash with args", "/model gpt-4", true},
		{"slash space", "/ test", true},
		{"plain text", "hello", false},
		{"empty string", "", false},
		{"just slash", "/", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := IsCommand(tt.input); got != tt.want {
				t.Errorf("IsCommand(%q) = %v; want %v", tt.input, got, tt.want)
			}
		})
	}
}
