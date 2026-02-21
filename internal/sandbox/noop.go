// ABOUTME: Path-only sandbox fallback when no OS sandbox is available
// ABOUTME: Validates write paths against allowed directories; reads unrestricted

package sandbox

import (
	"os/exec"
)

// noopSandbox does path validation only; no OS-level isolation.
type noopSandbox struct {
	opts Opts
}

func (n *noopSandbox) WrapCommand(cmd *exec.Cmd, _ Opts) (*exec.Cmd, error) {
	return cmd, nil
}

func (n *noopSandbox) ValidatePath(path string, write bool) error {
	if !write {
		return nil // Reads are unrestricted
	}
	return validateWritePath(path, n.opts)
}

func (n *noopSandbox) Available() bool { return true }
func (n *noopSandbox) Name() string    { return "noop" }
