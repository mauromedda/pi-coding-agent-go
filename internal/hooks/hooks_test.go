// ABOUTME: Tests for hook engine: allow, block, matcher filtering, timeout
// ABOUTME: Uses real shell commands to exercise the full execution path

package hooks

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/mauromedda/pi-coding-agent-go/internal/config"
)

func newEngine(t *testing.T, hooks map[string][]config.HookDef) *Engine {
	t.Helper()
	e, err := NewEngine(hooks)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	return e
}

func TestEngine_Fire_AllowsExecution(t *testing.T) {
	t.Parallel()

	// Hook script exits 0 and returns valid JSON with blocked=false.
	engine := newEngine(t, map[string][]config.HookDef{
		"PreToolUse": {
			{
				Matcher: ".*",
				Type:    "command",
				Command: `echo '{"blocked":false,"message":"ok"}'`,
			},
		},
	})

	out, err := engine.Fire(context.Background(), HookInput{
		Event: PreToolUse,
		Tool:  "bash",
	})
	if err != nil {
		t.Fatalf("Fire returned error: %v", err)
	}
	if out.Blocked {
		t.Error("expected Blocked=false for exit-0 hook")
	}
	if out.Message != "ok" {
		t.Errorf("Message = %q, want %q", out.Message, "ok")
	}
}

func TestEngine_Fire_BlocksExecution(t *testing.T) {
	t.Parallel()

	// Hook script exits 1; engine should set Blocked=true.
	engine := newEngine(t, map[string][]config.HookDef{
		"PreToolUse": {
			{
				Matcher: ".*",
				Type:    "command",
				Command: `echo '{"message":"denied"}' && exit 1`,
			},
		},
	})

	out, err := engine.Fire(context.Background(), HookInput{
		Event: PreToolUse,
		Tool:  "bash",
	})
	if err != nil {
		t.Fatalf("Fire returned error: %v", err)
	}
	if !out.Blocked {
		t.Error("expected Blocked=true for exit-1 hook")
	}
	if out.Message != "denied" {
		t.Errorf("Message = %q, want %q", out.Message, "denied")
	}
}

func TestEngine_Fire_MatcherFilter(t *testing.T) {
	t.Parallel()

	// Hook matcher only matches "bash"; firing with "read" should not trigger it.
	engine := newEngine(t, map[string][]config.HookDef{
		"PreToolUse": {
			{
				Matcher: "^bash$",
				Type:    "command",
				Command: `echo '{"blocked":true,"message":"bash blocked"}'  && exit 1`,
			},
		},
	})

	// "read" should NOT match the "^bash$" pattern.
	out, err := engine.Fire(context.Background(), HookInput{
		Event: PreToolUse,
		Tool:  "read",
	})
	if err != nil {
		t.Fatalf("Fire returned error: %v", err)
	}
	if out.Blocked {
		t.Error("expected Blocked=false when tool does not match matcher")
	}

	// "bash" SHOULD match.
	out, err = engine.Fire(context.Background(), HookInput{
		Event: PreToolUse,
		Tool:  "bash",
	})
	if err != nil {
		t.Fatalf("Fire returned error: %v", err)
	}
	if !out.Blocked {
		t.Error("expected Blocked=true when tool matches matcher")
	}
}

func TestEngine_Fire_Timeout(t *testing.T) {
	t.Parallel()

	// Hook sleeps for 30s; should be killed by the 10s timeout.
	engine := newEngine(t, map[string][]config.HookDef{
		"PreToolUse": {
			{
				Matcher: ".*",
				Type:    "command",
				Command: "sleep 30",
			},
		},
	})

	start := time.Now()
	_, err := engine.Fire(context.Background(), HookInput{
		Event: PreToolUse,
		Tool:  "bash",
	})

	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("expected error from timed-out hook")
	}
	// Should have been killed well before 30s; allow generous 15s window.
	if elapsed > 15*time.Second {
		t.Errorf("hook took %v, expected timeout around 10s", elapsed)
	}
}

func TestNewEngine_InvalidRegex(t *testing.T) {
	t.Parallel()

	_, err := NewEngine(map[string][]config.HookDef{
		"PreToolUse": {
			{
				Matcher: "[invalid",
				Type:    "command",
				Command: "echo ok",
			},
		},
	})
	if err == nil {
		t.Fatal("expected error for invalid regex matcher")
	}
	if !strings.Contains(err.Error(), "invalid hook matcher") {
		t.Errorf("error = %q, want it to mention 'invalid hook matcher'", err)
	}
}

func TestEngine_Fire_GarbageOutput(t *testing.T) {
	t.Parallel()

	// Hook outputs non-JSON text to stdout.
	engine := newEngine(t, map[string][]config.HookDef{
		"PreToolUse": {
			{
				Matcher: ".*",
				Type:    "command",
				Command: `echo "this is not json"`,
			},
		},
	})

	_, err := engine.Fire(context.Background(), HookInput{
		Event: PreToolUse,
		Tool:  "bash",
	})
	if err == nil {
		t.Fatal("expected error for non-JSON hook output")
	}
	if !strings.Contains(err.Error(), "parse hook output") {
		t.Errorf("error = %q, want it to mention 'parse hook output'", err)
	}
}
