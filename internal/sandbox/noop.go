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

func (n *noopSandbox) WrapCommand(cmd *exec.Cmd, perCall Opts) (*exec.Cmd, error) {
	merged := mergeOpts(n.opts, perCall)
	if merged.WorkDir != "" && cmd.Dir == "" {
		cmd.Dir = merged.WorkDir
	}
	return cmd, nil
}

// mergeOpts returns a copy of base with non-zero fields from override applied.
func mergeOpts(base, override Opts) Opts {
	result := base
	if override.WorkDir != "" {
		result.WorkDir = override.WorkDir
	}
	if len(override.AdditionalDirs) > 0 {
		result.AdditionalDirs = override.AdditionalDirs
	}
	if override.AllowNetwork {
		result.AllowNetwork = true
	}
	if len(override.AllowedDomains) > 0 {
		result.AllowedDomains = override.AllowedDomains
	}
	if len(override.ExcludedCmds) > 0 {
		result.ExcludedCmds = override.ExcludedCmds
	}
	return result
}

func (n *noopSandbox) ValidatePath(path string, write bool) error {
	if !write {
		return nil // Reads are unrestricted
	}
	return validateWritePath(path, n.opts)
}

func (n *noopSandbox) Available() bool { return true }
func (n *noopSandbox) Name() string    { return "noop" }
