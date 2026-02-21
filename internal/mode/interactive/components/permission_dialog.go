// ABOUTME: Permission prompt overlay for dangerous operations
// ABOUTME: Renders as a modal asking user to allow/deny tool execution

package components

import (
	"fmt"

	"github.com/mauromedda/pi-coding-agent-go/pkg/tui"
)

// PermissionDialog asks the user to approve a tool operation.
type PermissionDialog struct {
	toolName string
	args     string
	result   chan bool
}

// NewPermissionDialog creates a dialog for the given tool.
func NewPermissionDialog(toolName, args string) *PermissionDialog {
	return &PermissionDialog{
		toolName: toolName,
		args:     args,
		result:   make(chan bool, 1),
	}
}

// Render draws the permission dialog.
func (d *PermissionDialog) Render(out *tui.RenderBuffer, w int) {
	out.WriteLine("")
	out.WriteLine("\x1b[1;33m  Permission Required  \x1b[0m")
	out.WriteLine("")
	out.WriteLine(fmt.Sprintf("  Tool: \x1b[1m%s\x1b[0m", d.toolName))
	if d.args != "" {
		out.WriteLine(fmt.Sprintf("  Args: %s", d.args))
	}
	out.WriteLine("")
	out.WriteLine("  \x1b[32m[y]\x1b[0m Allow  \x1b[31m[n]\x1b[0m Deny")
	out.WriteLine("")
}

// Invalidate is a no-op for PermissionDialog.
func (d *PermissionDialog) Invalidate() {}

// Allow approves the operation.
func (d *PermissionDialog) Allow() {
	select {
	case d.result <- true:
	default:
	}
}

// Deny rejects the operation.
func (d *PermissionDialog) Deny() {
	select {
	case d.result <- false:
	default:
	}
}

// Wait blocks until the user responds.
func (d *PermissionDialog) Wait() bool {
	return <-d.result
}
