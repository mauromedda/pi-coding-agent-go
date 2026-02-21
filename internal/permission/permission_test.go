// ABOUTME: Tests for permission checker and sandbox path validation
// ABOUTME: Covers all modes: normal, yolo, plan; and rule matching

package permission

import "testing"

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
