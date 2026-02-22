// ABOUTME: Tests for PermissionDialog three-way response (Allow, Deny, AllowAlways)
// ABOUTME: Verifies channel-based response mechanism and render output

package components

import (
	"context"
	"strings"
	"testing"

	"github.com/mauromedda/pi-coding-agent-go/pkg/tui"
)

func TestPermissionDialog_Allow(t *testing.T) {
	t.Parallel()

	d := NewPermissionDialog("bash", `{"command":"ls"}`)

	go d.Allow()

	resp := d.Wait()
	if resp != PermAllow {
		t.Errorf("Wait() = %v, want PermAllow", resp)
	}
}

func TestPermissionDialog_Deny(t *testing.T) {
	t.Parallel()

	d := NewPermissionDialog("bash", "")

	go d.Deny()

	resp := d.Wait()
	if resp != PermDeny {
		t.Errorf("Wait() = %v, want PermDeny", resp)
	}
}

func TestPermissionDialog_AllowAlways(t *testing.T) {
	t.Parallel()

	d := NewPermissionDialog("edit", `{"file_path":"/tmp/test.go"}`)

	go d.AllowAlways()

	resp := d.Wait()
	if resp != PermAllowAlways {
		t.Errorf("Wait() = %v, want PermAllowAlways", resp)
	}
}

func TestPermissionDialog_ToolName(t *testing.T) {
	t.Parallel()

	d := NewPermissionDialog("write", "")
	if d.ToolName() != "write" {
		t.Errorf("ToolName() = %q, want %q", d.ToolName(), "write")
	}
}

func TestPermissionDialog_WaitContext_Allow(t *testing.T) {
	t.Parallel()

	d := NewPermissionDialog("bash", `{"command":"ls"}`)

	go d.Allow()

	resp := d.WaitContext(context.Background())
	if resp != PermAllow {
		t.Errorf("WaitContext() = %v, want PermAllow", resp)
	}
}

func TestPermissionDialog_WaitContext_CancelledContext(t *testing.T) {
	t.Parallel()

	d := NewPermissionDialog("bash", `{"command":"rm -rf /"}`)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately before any response

	resp := d.WaitContext(ctx)
	if resp != PermDeny {
		t.Errorf("WaitContext() with cancelled context = %v, want PermDeny", resp)
	}
}

func TestPermissionDialog_Render_ShowsAllOptions(t *testing.T) {
	t.Parallel()

	d := NewPermissionDialog("bash", `{"command":"rm -rf /tmp/test"}`)
	buf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(buf)

	d.Render(buf, 80)

	output := strings.Join(buf.Lines, "\n")
	if !strings.Contains(output, "bash") {
		t.Error("render should show tool name")
	}
	if !strings.Contains(output, "[y]") {
		t.Error("render should show [y] Allow")
	}
	if !strings.Contains(output, "[a]") {
		t.Error("render should show [a] Always")
	}
	if !strings.Contains(output, "[n]") {
		t.Error("render should show [n] Deny")
	}
}
