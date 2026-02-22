// ABOUTME: Tests for the slash command registry and dispatch
// ABOUTME: Covers all slash commands, unknown command error, nil callback safety, and IsCommand detection

package commands

import (
	"fmt"
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
		ClearTUI: func() {
			cb.clearTUICalled = true
		},
		CompactFn: func() string {
			cb.compactCalled = true
			return "Conversation compacted to 3 messages."
		},
		MemoryEntries: []string{"project uses Go 1.22", "prefer table-driven tests"},
		ToggleMode: func() {
			cb.toggleModeCalled = true
		},
		GetMode: func() string {
			return "PLAN"
		},
		RenameSession: func(name string) {
			cb.renameArg = name
		},
		ResumeSession: func(id string) error {
			cb.resumeArg = id
			return nil
		},
		SandboxStatus: func() string {
			cb.sandboxCalled = true
			return "Sandbox: enabled (container)"
		},
		ToggleVim: func() {
			cb.toggleVimCalled = true
		},
		VimEnabled: func() bool {
			return true
		},
		MCPServers: func() []string {
			cb.mcpCalled = true
			return []string{"filesystem", "github", "slack"}
		},
		ExportConversation: func(path string) error {
			cb.exportArg = path
			return nil
		},
		ExitFn: func() {
			cb.exitCalled = true
		},
		CopyLastMessageFn: func() (string, error) {
			cb.copyCalled = true
			return "Copied to clipboard.", nil
		},
		NewSessionFn: func() {
			cb.newSessionCalled = true
		},
		ForkSessionFn: func() (string, error) {
			cb.forkSessionCalled = true
			return "fork-id-123", nil
		},
	}
	return ctx, cb
}

type testCallbacks struct {
	modelSet           string
	clearCalled        bool
	clearTUICalled     bool
	compactCalled      bool
	toggleModeCalled   bool
	renameArg          string
	resumeArg          string
	sandboxCalled      bool
	toggleVimCalled    bool
	mcpCalled          bool
	exportArg          string
	exitCalled         bool
	copyCalled         bool
	newSessionCalled   bool
	forkSessionCalled  bool
}

func TestRegistry_AllCommandsRegistered(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()

	expected := []string{
		"changelog", "clear", "compact", "config", "context", "copy", "cost",
		"exit", "export", "fork", "help", "hooks", "hotkeys", "init", "mcp", "memory",
		"model", "new", "permissions", "plan", "quit", "reload", "rename", "resume", "sandbox",
		"scoped-models", "settings", "share", "status", "tree", "vim",
	}
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

	// Verify List returns all commands, sorted.
	all := reg.List()
	if len(all) != len(expected) {
		t.Fatalf("expected %d commands, got %d", len(expected), len(all))
	}
	for i, cmd := range all {
		if cmd.Name != expected[i] {
			t.Errorf("List()[%d]: expected %q, got %q", i, expected[i], cmd.Name)
		}
		if cmd.Category == "" {
			t.Errorf("command %q has empty Category", cmd.Name)
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
	if !cb.clearTUICalled {
		t.Error("ClearTUI was not called")
	}
	if !strings.Contains(result, "cleared") {
		t.Errorf("expected result to contain 'cleared', got %q", result)
	}
}

func TestDispatch_Clear_NilClearTUI(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, cb := testContext()
	ctx.ClearTUI = nil

	result, err := reg.Dispatch(ctx, "/clear")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cb.clearCalled {
		t.Error("ClearHistory was not called")
	}
	// Should not panic even with nil ClearTUI
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
	for _, name := range []string{
		"/clear", "/compact", "/config", "/context", "/cost",
		"/exit", "/export", "/help", "/init", "/mcp", "/memory",
		"/model", "/plan", "/rename", "/resume", "/sandbox",
		"/status", "/vim",
	} {
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

func TestDispatch_Model_NilSetModel(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, _ := testContext()
	ctx.SetModel = nil

	result, err := reg.Dispatch(ctx, "/model gpt-4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(strings.ToLower(result), "not available") {
		t.Errorf("expected 'not available' for nil SetModel, got %q", result)
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

func TestDispatch_Init(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, _ := testContext()

	result, err := reg.Dispatch(ctx, "/init")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(strings.ToLower(result), "initialized") {
		t.Errorf("expected result to contain 'initialized', got %q", result)
	}
}

func TestDispatch_Memory(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, _ := testContext()

	result, err := reg.Dispatch(ctx, "/memory")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, entry := range []string{"project uses Go 1.22", "prefer table-driven tests"} {
		if !strings.Contains(result, entry) {
			t.Errorf("expected memory output to contain %q, got:\n%s", entry, result)
		}
	}
}

func TestDispatch_Memory_Empty(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, _ := testContext()
	ctx.MemoryEntries = nil

	result, err := reg.Dispatch(ctx, "/memory")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(strings.ToLower(result), "no memory entries") {
		t.Errorf("expected 'no memory entries' message, got %q", result)
	}
}

func TestDispatch_Context(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, _ := testContext()

	result, err := reg.Dispatch(ctx, "/context")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{"/tmp/project", "claude-sonnet"} {
		if !strings.Contains(result, want) {
			t.Errorf("expected context output to contain %q, got:\n%s", want, result)
		}
	}
}

func TestDispatch_Cost(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, _ := testContext()

	result, err := reg.Dispatch(ctx, "/cost")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "$1.2300") {
		t.Errorf("expected cost '$1.2300' in output, got:\n%s", result)
	}
	if !strings.Contains(result, "4567") {
		t.Errorf("expected token count '4567' in output, got:\n%s", result)
	}
}

func TestDispatch_Plan(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, cb := testContext()

	result, err := reg.Dispatch(ctx, "/plan")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cb.toggleModeCalled {
		t.Error("ToggleMode was not called")
	}
	if !strings.Contains(result, "PLAN") {
		t.Errorf("expected result to contain mode name, got %q", result)
	}
}

func TestDispatch_Plan_NilCallback(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, _ := testContext()
	ctx.ToggleMode = nil
	ctx.GetMode = nil

	result, err := reg.Dispatch(ctx, "/plan")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(strings.ToLower(result), "not available") {
		t.Errorf("expected 'not available' for nil callback, got %q", result)
	}
}

func TestDispatch_Rename(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, cb := testContext()

	result, err := reg.Dispatch(ctx, "/rename my-session")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cb.renameArg != "my-session" {
		t.Errorf("expected RenameSession called with 'my-session', got %q", cb.renameArg)
	}
	if !strings.Contains(result, "my-session") {
		t.Errorf("expected confirmation to contain session name, got %q", result)
	}
}

func TestDispatch_Rename_NilCallback(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, _ := testContext()
	ctx.RenameSession = nil

	result, err := reg.Dispatch(ctx, "/rename test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(strings.ToLower(result), "not available") {
		t.Errorf("expected 'not available' for nil callback, got %q", result)
	}
}

func TestDispatch_Rename_NoArgs(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, cb := testContext()

	result, err := reg.Dispatch(ctx, "/rename")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cb.renameArg != "" {
		t.Error("RenameSession should not have been called without an argument")
	}
	if !strings.Contains(result, "Usage:") {
		t.Errorf("expected usage message, got %q", result)
	}
}

func TestDispatch_Resume(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, cb := testContext()

	result, err := reg.Dispatch(ctx, "/resume abc-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cb.resumeArg != "abc-123" {
		t.Errorf("expected ResumeSession called with 'abc-123', got %q", cb.resumeArg)
	}
	if !strings.Contains(result, "abc-123") {
		t.Errorf("expected confirmation to contain session id, got %q", result)
	}
}

func TestDispatch_Resume_NoArgs(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, cb := testContext()

	result, err := reg.Dispatch(ctx, "/resume")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cb.resumeArg != "" {
		t.Error("ResumeSession should not have been called without an argument")
	}
	if !strings.Contains(result, "Usage:") {
		t.Errorf("expected usage message, got %q", result)
	}
}

func TestDispatch_Resume_Error(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, _ := testContext()
	ctx.ResumeSession = func(_ string) error {
		return fmt.Errorf("session not found")
	}

	_, err := reg.Dispatch(ctx, "/resume bad-id")
	if err == nil {
		t.Fatal("expected error from ResumeSession")
	}
	if !strings.Contains(err.Error(), "session not found") {
		t.Errorf("expected error to contain 'session not found', got %q", err.Error())
	}
}

func TestDispatch_Sandbox(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, cb := testContext()

	result, err := reg.Dispatch(ctx, "/sandbox")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cb.sandboxCalled {
		t.Error("SandboxStatus was not called")
	}
	if !strings.Contains(result, "enabled") {
		t.Errorf("expected sandbox status in output, got %q", result)
	}
}

func TestDispatch_Sandbox_NilCallback(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, _ := testContext()
	ctx.SandboxStatus = nil

	result, err := reg.Dispatch(ctx, "/sandbox")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(strings.ToLower(result), "not available") {
		t.Errorf("expected 'not available' for nil callback, got %q", result)
	}
}

func TestDispatch_Vim(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, cb := testContext()

	result, err := reg.Dispatch(ctx, "/vim")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cb.toggleVimCalled {
		t.Error("ToggleVim was not called")
	}
	if !strings.Contains(strings.ToLower(result), "enabled") {
		t.Errorf("expected vim status in output, got %q", result)
	}
}

func TestDispatch_Vim_NilCallback(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, _ := testContext()
	ctx.ToggleVim = nil
	ctx.VimEnabled = nil

	result, err := reg.Dispatch(ctx, "/vim")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(strings.ToLower(result), "not available") {
		t.Errorf("expected 'not available' for nil callback, got %q", result)
	}
}

func TestDispatch_MCP(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, cb := testContext()

	result, err := reg.Dispatch(ctx, "/mcp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cb.mcpCalled {
		t.Error("MCPServers was not called")
	}
	for _, server := range []string{"filesystem", "github", "slack"} {
		if !strings.Contains(result, server) {
			t.Errorf("expected MCP output to contain %q, got:\n%s", server, result)
		}
	}
}

func TestDispatch_MCP_NilCallback(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, _ := testContext()
	ctx.MCPServers = nil

	result, err := reg.Dispatch(ctx, "/mcp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(strings.ToLower(result), "not available") {
		t.Errorf("expected 'not available' for nil callback, got %q", result)
	}
}

func TestDispatch_Export(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, cb := testContext()

	result, err := reg.Dispatch(ctx, "/export /tmp/chat.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cb.exportArg != "/tmp/chat.md" {
		t.Errorf("expected ExportConversation called with '/tmp/chat.md', got %q", cb.exportArg)
	}
	if !strings.Contains(result, "/tmp/chat.md") {
		t.Errorf("expected confirmation to contain path, got %q", result)
	}
}

func TestDispatch_Export_NoArgs(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, cb := testContext()

	result, err := reg.Dispatch(ctx, "/export")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cb.exportArg != "" {
		t.Error("ExportConversation should not have been called without an argument")
	}
	if !strings.Contains(result, "Usage:") {
		t.Errorf("expected usage message, got %q", result)
	}
}

func TestDispatch_Export_Error(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, _ := testContext()
	ctx.ExportConversation = func(_ string) error {
		return fmt.Errorf("permission denied")
	}

	_, err := reg.Dispatch(ctx, "/export /root/chat.md")
	if err == nil {
		t.Fatal("expected error from ExportConversation")
	}
	if !strings.Contains(err.Error(), "permission denied") {
		t.Errorf("expected error to contain 'permission denied', got %q", err.Error())
	}
}

func TestDispatch_Export_NilCallback(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, _ := testContext()
	ctx.ExportConversation = nil

	result, err := reg.Dispatch(ctx, "/export /tmp/chat.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(strings.ToLower(result), "not available") {
		t.Errorf("expected 'not available' for nil callback, got %q", result)
	}
}

func TestDispatch_Exit(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, cb := testContext()

	result, err := reg.Dispatch(ctx, "/exit")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cb.exitCalled {
		t.Error("ExitFn was not called")
	}
	if !strings.Contains(strings.ToLower(result), "goodbye") {
		t.Errorf("expected goodbye message, got %q", result)
	}
}

func TestDispatch_Exit_NilCallback(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, _ := testContext()
	ctx.ExitFn = nil

	result, err := reg.Dispatch(ctx, "/exit")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(strings.ToLower(result), "not available") {
		t.Errorf("expected 'not available' for nil callback, got %q", result)
	}
}

func TestDispatch_Hooks_NilCallback(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, _ := testContext()
	ctx.HookManagerFn = nil

	result, err := reg.Dispatch(ctx, "/hooks")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(strings.ToLower(result), "not available") {
		t.Errorf("expected 'not available' for nil HookManagerFn, got %q", result)
	}
}

func TestDispatch_Permissions_NilCallback(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, _ := testContext()
	ctx.PermissionManagerFn = nil

	result, err := reg.Dispatch(ctx, "/permissions")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(strings.ToLower(result), "not available") {
		t.Errorf("expected 'not available' for nil PermissionManagerFn, got %q", result)
	}
}

func TestDispatch_Hotkeys_Default(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, _ := testContext()
	ctx.KeybindingsFn = nil

	result, err := reg.Dispatch(ctx, "/hotkeys")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should return the default hotkeys table.
	for _, want := range []string{"Ctrl+C", "Ctrl+D", "Shift+Tab", "Enter"} {
		if !strings.Contains(result, want) {
			t.Errorf("expected default hotkeys to contain %q, got:\n%s", want, result)
		}
	}
}

func TestDispatch_Tree_NilCallback(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, _ := testContext()
	ctx.SessionTreeFn = nil

	result, err := reg.Dispatch(ctx, "/tree")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(strings.ToLower(result), "not available") {
		t.Errorf("expected 'not available' for nil SessionTreeFn, got %q", result)
	}
}

func TestDispatch_ScopedModels_NilCallback(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, _ := testContext()
	ctx.ScopedModelsFn = nil

	result, err := reg.Dispatch(ctx, "/scoped-models")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(strings.ToLower(result), "not available") {
		t.Errorf("expected 'not available' for nil ScopedModelsFn, got %q", result)
	}
}

func TestDispatch_Resume_NoArgsWithListFn(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, _ := testContext()
	listCalled := false
	ctx.ListSessionsFn = func() string {
		listCalled = true
		return "session-1\nsession-2\nsession-3"
	}

	result, err := reg.Dispatch(ctx, "/resume")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !listCalled {
		t.Error("ListSessionsFn was not called")
	}
	if !strings.Contains(result, "session-1") {
		t.Errorf("expected result to contain session list, got %q", result)
	}
}

func TestHelp_HasCategories(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, _ := testContext()

	result, err := reg.Dispatch(ctx, "/help")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, header := range []string{"## Session", "## Config"} {
		if !strings.Contains(result, header) {
			t.Errorf("expected help output to contain category header %q, got:\n%s", header, result)
		}
	}
}

func TestDispatch_Settings(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, _ := testContext()
	settingsCalled := false
	ctx.GetSettings = func() string {
		settingsCalled = true
		return "theme: dark\neditor: vim"
	}

	result, err := reg.Dispatch(ctx, "/settings")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !settingsCalled {
		t.Error("GetSettings was not called")
	}
	if !strings.Contains(result, "theme: dark") {
		t.Errorf("expected settings output, got %q", result)
	}
}

func TestDispatch_Settings_NilCallback(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, _ := testContext()
	ctx.GetSettings = nil

	result, err := reg.Dispatch(ctx, "/settings")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(strings.ToLower(result), "not available") {
		t.Errorf("expected 'not available' for nil GetSettings, got %q", result)
	}
}

func TestDispatch_Changelog(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, _ := testContext()

	result, err := reg.Dispatch(ctx, "/changelog")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "Changelog") {
		t.Errorf("expected changelog content, got %q", result)
	}
}

func TestDispatch_Share(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, _ := testContext()
	shareCalled := false
	ctx.ShareFn = func() string {
		shareCalled = true
		return "Share link: https://example.com/share/abc"
	}

	result, err := reg.Dispatch(ctx, "/share")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !shareCalled {
		t.Error("ShareFn was not called")
	}
	if !strings.Contains(result, "https://example.com/share/abc") {
		t.Errorf("expected share link in output, got %q", result)
	}
}

func TestDispatch_Share_NilCallback(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, _ := testContext()
	ctx.ShareFn = nil

	result, err := reg.Dispatch(ctx, "/share")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(strings.ToLower(result), "not available") {
		t.Errorf("expected 'not available' for nil ShareFn, got %q", result)
	}
}

func TestDispatch_Export_HTMLDetection(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, _ := testContext()
	var exportedPath string
	ctx.ExportConversation = func(path string) error {
		exportedPath = path
		return nil
	}

	// Regular markdown export should still work.
	result, err := reg.Dispatch(ctx, "/export /tmp/chat.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exportedPath != "/tmp/chat.md" {
		t.Errorf("expected export path '/tmp/chat.md', got %q", exportedPath)
	}
	if !strings.Contains(result, "/tmp/chat.md") {
		t.Errorf("expected path in result, got %q", result)
	}
}

func TestDispatch_Export_HTMLWithCallback(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, _ := testContext()
	var htmlExportedPath string
	ctx.ExportHTMLFn = func(path string) error {
		htmlExportedPath = path
		return nil
	}

	result, err := reg.Dispatch(ctx, "/export /tmp/chat.html")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if htmlExportedPath != "/tmp/chat.html" {
		t.Errorf("expected ExportHTMLFn called with '/tmp/chat.html', got %q", htmlExportedPath)
	}
	if !strings.Contains(result, "HTML") {
		t.Errorf("expected result to mention HTML, got %q", result)
	}
}

func TestDispatch_Export_HTMLNilCallback(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, _ := testContext()
	ctx.ExportHTMLFn = nil

	result, err := reg.Dispatch(ctx, "/export /tmp/chat.html")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(strings.ToLower(result), "not available") {
		t.Errorf("expected 'not available' for nil ExportHTMLFn, got %q", result)
	}
}

func TestDispatch_Export_HTMExtension(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, _ := testContext()
	var htmlExportedPath string
	ctx.ExportHTMLFn = func(path string) error {
		htmlExportedPath = path
		return nil
	}

	result, err := reg.Dispatch(ctx, "/export /tmp/chat.htm")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if htmlExportedPath != "/tmp/chat.htm" {
		t.Errorf("expected ExportHTMLFn called with '/tmp/chat.htm', got %q", htmlExportedPath)
	}
	if !strings.Contains(result, "HTML") {
		t.Errorf("expected result to mention HTML, got %q", result)
	}
}

func TestDispatch_Export_HTMLError(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, _ := testContext()
	ctx.ExportHTMLFn = func(_ string) error {
		return fmt.Errorf("html render failed")
	}

	_, err := reg.Dispatch(ctx, "/export /tmp/chat.html")
	if err == nil {
		t.Fatal("expected error from ExportHTMLFn")
	}
	if !strings.Contains(err.Error(), "html render failed") {
		t.Errorf("expected error to contain 'html render failed', got %q", err.Error())
	}
}

func TestDispatch_Copy(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, cb := testContext()

	result, err := reg.Dispatch(ctx, "/copy")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cb.copyCalled {
		t.Error("CopyLastMessageFn was not called")
	}
	if !strings.Contains(strings.ToLower(result), "copied") {
		t.Errorf("expected result to contain 'copied', got %q", result)
	}
}

func TestDispatch_Copy_NilCallback(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, _ := testContext()
	ctx.CopyLastMessageFn = nil

	result, err := reg.Dispatch(ctx, "/copy")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(strings.ToLower(result), "not available") {
		t.Errorf("expected 'not available' for nil callback, got %q", result)
	}
}

func TestDispatch_New(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, cb := testContext()

	result, err := reg.Dispatch(ctx, "/new")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cb.newSessionCalled {
		t.Error("NewSessionFn was not called")
	}
	if !strings.Contains(strings.ToLower(result), "new session") {
		t.Errorf("expected result to contain 'new session', got %q", result)
	}
}

func TestDispatch_New_NilCallback(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, _ := testContext()
	ctx.NewSessionFn = nil

	result, err := reg.Dispatch(ctx, "/new")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(strings.ToLower(result), "not available") {
		t.Errorf("expected 'not available' for nil callback, got %q", result)
	}
}

func TestDispatch_Fork(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, cb := testContext()

	result, err := reg.Dispatch(ctx, "/fork")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cb.forkSessionCalled {
		t.Error("ForkSessionFn was not called")
	}
	if !strings.Contains(result, "fork-id-123") {
		t.Errorf("expected result to contain fork ID, got %q", result)
	}
}

func TestDispatch_Fork_NilCallback(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, _ := testContext()
	ctx.ForkSessionFn = nil

	result, err := reg.Dispatch(ctx, "/fork")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(strings.ToLower(result), "not available") {
		t.Errorf("expected 'not available' for nil callback, got %q", result)
	}
}

func TestDispatch_Fork_Error(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, _ := testContext()
	ctx.ForkSessionFn = func() (string, error) {
		return "", fmt.Errorf("session not saved")
	}

	_, err := reg.Dispatch(ctx, "/fork")
	if err == nil {
		t.Fatal("expected error from ForkSessionFn")
	}
	if !strings.Contains(err.Error(), "session not saved") {
		t.Errorf("expected error to contain 'session not saved', got %q", err.Error())
	}
}

func TestDispatch_Quit(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, cb := testContext()

	result, err := reg.Dispatch(ctx, "/quit")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cb.exitCalled {
		t.Error("ExitFn was not called (quit should alias exit)")
	}
	if !strings.Contains(strings.ToLower(result), "goodbye") {
		t.Errorf("expected goodbye message, got %q", result)
	}
}

func TestDispatch_Quit_NilCallback(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ctx, _ := testContext()
	ctx.ExitFn = nil

	result, err := reg.Dispatch(ctx, "/quit")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(strings.ToLower(result), "not available") {
		t.Errorf("expected 'not available' for nil callback, got %q", result)
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
