// ABOUTME: Permission prompt overlay for dangerous operations
// ABOUTME: Renders as a modal with three responses: Allow, Always, Deny
// ABOUTME: Updated to match Claude Code's minimal, clean styling

package components

import (
	"context"
	"fmt"

	"github.com/mauromedda/pi-coding-agent-go/pkg/tui"
)

// PermissionResponse represents the user's decision on a permission prompt.
type PermissionResponse int

const (
	PermDeny        PermissionResponse = iota // User denied the operation
	PermAllow                                 // User allowed this one time
	PermAllowAlways                           // User allowed and wants to persist rule
)

// PermissionDialog asks the user to approve a tool operation.
type PermissionDialog struct {
	toolName string
	args     string
	result   chan PermissionResponse
}

// NewPermissionDialog creates a dialog for the given tool.
func NewPermissionDialog(toolName, args string) *PermissionDialog {
	return &PermissionDialog{
		toolName: toolName,
		args:     args,
		result:   make(chan PermissionResponse, 1),
	}
}

// ToolName returns the tool name this dialog is prompting for.
func (d *PermissionDialog) ToolName() string {
	return d.toolName
}

// Render draws the permission dialog with Claude-style minimal styling.
func (d *PermissionDialog) Render(out *tui.RenderBuffer, w int) {
	out.WriteLine("")

	// Claude-style: simple header without box, just "Permission Required"
	out.WriteLine("\x1b[1;33mPermission Required\x1b[0m")
	out.WriteLine("")

	// Tool name with simple indent
	out.WriteLine(fmt.Sprintf("  Tool: \x1b[1m%s\x1b[0m", d.toolName))

	// Args if present, wrapped and indented
	if d.args != "" {
		out.WriteLine("")
		out.WriteLine("  Args:")
		out.WriteLine(fmt.Sprintf("  %s", d.args))
	}

	out.WriteLine("")

	// Options: [y] Allow  [a] Always  [n] Deny (Claude-style minimal)
	out.WriteLine("  \x1b[32m[y]\x1b[0m Allow  \x1b[36m[a]\x1b[0m Always  \x1b[31m[n]\x1b[0m Deny")
	out.WriteLine("")
}

// Invalidate is a no-op for PermissionDialog.
func (d *PermissionDialog) Invalidate() {}

// Allow approves the operation for this invocation.
func (d *PermissionDialog) Allow() {
	select {
	case d.result <- PermAllow:
	default:
	}
}

// AllowAlways approves the operation and signals to persist the rule.
func (d *PermissionDialog) AllowAlways() {
	select {
	case d.result <- PermAllowAlways:
	default:
	}
}

// Deny rejects the operation.
func (d *PermissionDialog) Deny() {
	select {
	case d.result <- PermDeny:
	default:
	}
}

// Wait blocks until the user responds.
func (d *PermissionDialog) Wait() PermissionResponse {
	return <-d.result
}

// WaitContext blocks until the user responds or the context is cancelled.
// Returns PermDeny if the context is cancelled before a response arrives.
func (d *PermissionDialog) WaitContext(ctx context.Context) PermissionResponse {
	select {
	case resp := <-d.result:
		return resp
	case <-ctx.Done():
		return PermDeny
	}
}
