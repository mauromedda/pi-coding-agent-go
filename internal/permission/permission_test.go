// ABOUTME: Tests for permission checker and sandbox path validation
// ABOUTME: Covers all modes: normal, yolo, plan; rule matching; sandbox prefix/symlink

package permission

import (
	"os"
	"path/filepath"
	"testing"
)

func TestChecker_PlanMode(t *testing.T) {
	t.Parallel()

	c := NewChecker(ModePlan, nil)

	if err := c.Check("read", nil); err != nil {
		t.Errorf("read should be allowed in plan mode: %v", err)
	}
	if err := c.Check("grep", nil); err != nil {
		t.Errorf("grep should be allowed in plan mode: %v", err)
	}
	if err := c.Check("write", nil); err == nil {
		t.Error("write should be blocked in plan mode")
	}
	if err := c.Check("bash", nil); err == nil {
		t.Error("bash should be blocked in plan mode")
	}
}

func TestChecker_YoloMode(t *testing.T) {
	t.Parallel()

	c := NewChecker(ModeYolo, nil)

	if err := c.Check("bash", nil); err != nil {
		t.Errorf("bash should be allowed in yolo mode: %v", err)
	}
	if err := c.Check("write", nil); err != nil {
		t.Errorf("write should be allowed in yolo mode: %v", err)
	}
}

func TestChecker_NormalMode_AskFn(t *testing.T) {
	t.Parallel()

	asked := false
	askFn := func(tool string, args map[string]any) (bool, error) {
		asked = true
		return true, nil
	}

	c := NewChecker(ModeNormal, askFn)

	if err := c.Check("bash", nil); err != nil {
		t.Errorf("bash should be allowed after user approval: %v", err)
	}
	if !asked {
		t.Error("askFn should have been called")
	}
}

func TestChecker_DenyRule(t *testing.T) {
	t.Parallel()

	c := NewChecker(ModeYolo, nil)
	c.AddDenyRule(Rule{Tool: "bash", Message: "bash is blocked"})

	if err := c.Check("bash", nil); err == nil {
		t.Error("bash should be blocked by deny rule")
	}
}

func TestChecker_AllowRule(t *testing.T) {
	t.Parallel()

	c := NewChecker(ModeNormal, nil)
	c.AddAllowRule(Rule{Tool: "bash"})

	if err := c.Check("bash", nil); err != nil {
		t.Errorf("bash should be allowed by allow rule: %v", err)
	}
}

func TestMatchTool(t *testing.T) {
	t.Parallel()

	tests := []struct {
		pattern string
		tool    string
		want    bool
	}{
		{"*", "anything", true},
		{"bash", "bash", true},
		{"bash", "read", false},
		{"bash*", "bash_exec", true},
	}

	for _, tt := range tests {
		got := matchTool(tt.pattern, tt.tool)
		if got != tt.want {
			t.Errorf("matchTool(%q, %q) = %v, want %v", tt.pattern, tt.tool, got, tt.want)
		}
	}
}

func TestSandbox_ValidPath(t *testing.T) {
	t.Parallel()

	sb, err := NewSandbox([]string{"/tmp"})
	if err != nil {
		t.Fatal(err)
	}

	if err := sb.ValidatePath("/tmp/test.txt"); err != nil {
		t.Errorf("expected valid: %v", err)
	}
}

func TestSandbox_Traversal(t *testing.T) {
	t.Parallel()

	sb, err := NewSandbox([]string{"/tmp"})
	if err != nil {
		t.Fatal(err)
	}

	if err := sb.ValidatePath("/tmp/../etc/passwd"); err == nil {
		t.Error("expected traversal to be rejected")
	}
}

func TestSandbox_PrefixBypass(t *testing.T) {
	t.Parallel()

	sb, err := NewSandbox([]string{"/tmp"})
	if err != nil {
		t.Fatal(err)
	}

	// "/tmpevil" shares the prefix "/tmp" but is NOT inside "/tmp/"
	if err := sb.ValidatePath("/tmpevil/secret.txt"); err == nil {
		t.Error("expected /tmpevil to be rejected: prefix bypass without separator boundary")
	}
}

func TestSandbox_SymlinkResolution(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	allowed := filepath.Join(dir, "allowed")
	secret := filepath.Join(dir, "secret")
	link := filepath.Join(allowed, "escape")

	if err := os.MkdirAll(allowed, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(secret, 0o755); err != nil {
		t.Fatal(err)
	}
	// Create a symlink inside allowed that points outside
	if err := os.Symlink(secret, link); err != nil {
		t.Fatal(err)
	}

	sb, err := NewSandbox([]string{allowed})
	if err != nil {
		t.Fatal(err)
	}

	// The symlink resolves to "secret", which is outside "allowed"
	if err := sb.ValidatePath(filepath.Join(link, "data.txt")); err == nil {
		t.Error("expected symlink escape to be rejected")
	}
}

func TestMode_String_AllModes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		mode Mode
		want string
	}{
		{ModeNormal, "normal"},
		{ModeAcceptEdits, "accept-edits"},
		{ModePlan, "plan"},
		{ModeDontAsk, "dont-ask"},
		{ModeYolo, "yolo"},
		{Mode(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.mode.String(); got != tt.want {
			t.Errorf("Mode(%d).String() = %q, want %q", tt.mode, got, tt.want)
		}
	}
}

func TestParseMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input   string
		want    Mode
		wantErr bool
	}{
		{"default", ModeNormal, false},
		{"acceptEdits", ModeAcceptEdits, false},
		{"plan", ModePlan, false},
		{"dontAsk", ModeDontAsk, false},
		{"bypassPermissions", ModeYolo, false},
		{"", ModeNormal, false},
		{"normal", ModeNormal, false},
		{"unknown", 0, true},
		{"YOLO", 0, true},
	}

	for _, tt := range tests {
		got, err := ParseMode(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseMode(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && got != tt.want {
			t.Errorf("ParseMode(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestChecker_AcceptEditsMode_AllowsEditWrite(t *testing.T) {
	t.Parallel()

	askCalled := false
	askFn := func(tool string, args map[string]any) (bool, error) {
		askCalled = true
		return true, nil
	}

	c := NewChecker(ModeAcceptEdits, askFn)

	// edit, write, notebook_edit should pass without asking
	for _, tool := range []string{"edit", "write", "notebook_edit"} {
		askCalled = false
		if err := c.Check(tool, nil); err != nil {
			t.Errorf("%s should be allowed in accept-edits mode: %v", tool, err)
		}
		if askCalled {
			t.Errorf("askFn should NOT have been called for %s in accept-edits mode", tool)
		}
	}

	// read-only tools should also pass without asking
	askCalled = false
	if err := c.Check("read", nil); err != nil {
		t.Errorf("read should be allowed: %v", err)
	}
	if askCalled {
		t.Error("askFn should NOT have been called for read")
	}
}

func TestChecker_AcceptEditsMode_PromptsBash(t *testing.T) {
	t.Parallel()

	askCalled := false
	askFn := func(tool string, args map[string]any) (bool, error) {
		askCalled = true
		return true, nil
	}

	c := NewChecker(ModeAcceptEdits, askFn)

	// bash should trigger askFn (not an edit tool)
	if err := c.Check("bash", nil); err != nil {
		t.Errorf("bash should be allowed after user approval: %v", err)
	}
	if !askCalled {
		t.Error("askFn should have been called for bash in accept-edits mode")
	}
}

func TestChecker_DontAskMode_DeniesUnlessAllowed(t *testing.T) {
	t.Parallel()

	c := NewChecker(ModeDontAsk, nil)

	// read-only tools should be allowed
	for _, tool := range []string{"read", "grep", "find", "ls"} {
		if err := c.Check(tool, nil); err != nil {
			t.Errorf("%s should be allowed in dont-ask mode: %v", tool, err)
		}
	}

	// non-allowed tools should be denied
	for _, tool := range []string{"bash", "write", "edit"} {
		if err := c.Check(tool, nil); err == nil {
			t.Errorf("%s should be denied in dont-ask mode", tool)
		}
	}
}

func TestChecker_DontAskMode_AllowRulePermits(t *testing.T) {
	t.Parallel()

	c := NewChecker(ModeDontAsk, nil)
	c.AddAllowRule(Rule{Tool: "bash"})

	// bash should be allowed via explicit rule
	if err := c.Check("bash", nil); err != nil {
		t.Errorf("bash should be allowed by allow rule in dont-ask mode: %v", err)
	}

	// write should still be denied
	if err := c.Check("write", nil); err == nil {
		t.Error("write should be denied in dont-ask mode without allow rule")
	}
}

func TestChecker_NilAskFn_DeniesWriteInNormalMode(t *testing.T) {
	t.Parallel()

	// M6: When askFn is nil in ModeNormal, write tools should be denied, not silently allowed
	c := NewChecker(ModeNormal, nil)

	if err := c.Check("write", nil); err == nil {
		t.Error("write should be denied when askFn is nil in normal mode")
	}
	if err := c.Check("bash", nil); err == nil {
		t.Error("bash should be denied when askFn is nil in normal mode")
	}
	// Read-only tools should still be allowed
	if err := c.Check("read", nil); err != nil {
		t.Errorf("read should be allowed: %v", err)
	}
}
