// ABOUTME: Sandbox interface and factory for OS-level command isolation
// ABOUTME: Auto-detects seatbelt (macOS), bwrap (Linux), or noop fallback

package sandbox

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
)

// Sandbox wraps commands with OS-level isolation.
type Sandbox interface {
	// WrapCommand wraps an exec.Cmd with sandbox constraints.
	WrapCommand(cmd *exec.Cmd, opts Opts) (*exec.Cmd, error)
	// ValidatePath checks if a path is accessible (write=true for write access).
	ValidatePath(path string, write bool) error
	// Available reports whether this sandbox backend is usable.
	Available() bool
	// Name returns the sandbox backend name.
	Name() string
}

// Opts configures sandbox behavior.
type Opts struct {
	WorkDir        string
	AdditionalDirs []string
	AllowNetwork   bool
	AllowedDomains []string
	ExcludedCmds   []string
}

// New auto-detects the best sandbox for the current OS.
func New(defaultOpts Opts) Sandbox {
	switch runtime.GOOS {
	case "darwin":
		s := &seatbeltSandbox{opts: defaultOpts}
		if s.Available() {
			return s
		}
	case "linux":
		b := &bwrapSandbox{opts: defaultOpts}
		if b.Available() {
			return b
		}
	}
	return &noopSandbox{opts: defaultOpts}
}

// isExcludedCmd checks if the first token of a command is in the excluded list.
func isExcludedCmd(command string, excluded []string) bool {
	if command == "" || len(excluded) == 0 {
		return false
	}
	first := strings.Fields(command)[0]
	return slices.Contains(excluded, first)
}

// validateWritePath checks if a write path is within allowed dirs.
// Uses separator boundary to prevent prefix bypass (e.g. /tmp vs /tmpevil).
func validateWritePath(path string, opts Opts) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolving path %q: %w", path, err)
	}

	allowed := []string{opts.WorkDir}
	allowed = append(allowed, opts.AdditionalDirs...)

	for _, dir := range allowed {
		dirAbs, _ := filepath.Abs(dir)
		if dirAbs == "" {
			continue
		}
		// Add separator boundary to prevent /tmpevil matching /tmp
		dirWithSep := dirAbs + string(filepath.Separator)
		if strings.HasPrefix(abs+string(filepath.Separator), dirWithSep) || abs == dirAbs {
			return nil
		}
	}

	return fmt.Errorf("write to %q denied: outside allowed directories", path)
}
