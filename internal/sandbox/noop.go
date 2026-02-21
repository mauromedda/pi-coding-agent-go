// ABOUTME: Path-only sandbox fallback when no OS sandbox is available
// ABOUTME: Validates write paths against allowed directories; reads unrestricted

package sandbox

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
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

	abs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolving path %q: %w", path, err)
	}

	// Check work dir
	if n.opts.WorkDir != "" {
		workAbs, _ := filepath.Abs(n.opts.WorkDir)
		if strings.HasPrefix(abs, workAbs) {
			return nil
		}
	}

	// Check additional dirs
	for _, dir := range n.opts.AdditionalDirs {
		dirAbs, _ := filepath.Abs(dir)
		if strings.HasPrefix(abs, dirAbs) {
			return nil
		}
	}

	return fmt.Errorf("write to %q denied: outside allowed directories", path)
}

func (n *noopSandbox) Available() bool { return true }
func (n *noopSandbox) Name() string    { return "noop" }
