// ABOUTME: Path validation to restrict file access to allowed directories
// ABOUTME: Rejects path traversal, symlink escapes, and enforces separator-aware prefix matching

package permission

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Sandbox validates file paths against a set of allowed directories.
type Sandbox struct {
	allowed []string // Normalized absolute paths with trailing separator
}

// NewSandbox creates a Sandbox with the given allowed directories.
// Resolves symlinks in allowed dirs for consistent comparison.
func NewSandbox(allowedDirs []string) (*Sandbox, error) {
	normalized := make([]string, 0, len(allowedDirs))
	for _, dir := range allowedDirs {
		// Resolve symlinks for consistency (e.g. /tmp → /private/tmp on macOS)
		resolved, err := filepath.EvalSymlinks(dir)
		if err != nil {
			resolved = dir // Fall back to raw path if dir doesn't exist yet
		}
		abs, err := filepath.Abs(resolved)
		if err != nil {
			return nil, fmt.Errorf("resolving path %q: %w", dir, err)
		}
		// Ensure trailing separator for correct prefix matching:
		// "/tmp" + "/" prevents "/tmpevil" from matching
		if !strings.HasSuffix(abs, string(filepath.Separator)) {
			abs += string(filepath.Separator)
		}
		normalized = append(normalized, abs)
	}
	return &Sandbox{allowed: normalized}, nil
}

// ValidatePath checks that a path is within the allowed directories
// and does not contain path traversal components.
// Resolves symlinks to prevent sandbox escapes.
func (s *Sandbox) ValidatePath(path string) error {
	if containsTraversal(path) {
		return fmt.Errorf("path %q contains traversal components", path)
	}

	resolved, err := resolvePathForValidation(path)
	if err != nil {
		return fmt.Errorf("resolving path %q: %w", path, err)
	}

	// Check prefix against allowed directories with separator boundary
	for _, allowed := range s.allowed {
		if strings.HasPrefix(resolved+string(filepath.Separator), allowed) || resolved == strings.TrimSuffix(allowed, string(filepath.Separator)) {
			return nil
		}
	}

	return fmt.Errorf("path %q is outside allowed directories", path)
}

// resolvePathForValidation resolves symlinks where possible.
// For non-existent paths (e.g. writes), resolves the parent directory.
func resolvePathForValidation(path string) (string, error) {
	resolved, err := filepath.EvalSymlinks(path)
	if err == nil {
		return filepath.Abs(resolved)
	}

	// Path doesn't exist yet: resolve parent, keep base name
	if os.IsNotExist(err) {
		parent := filepath.Dir(path)
		resolvedParent, err := filepath.EvalSymlinks(parent)
		if err != nil {
			// Parent also doesn't exist: fall back to Abs
			return filepath.Abs(path)
		}
		absParent, err := filepath.Abs(resolvedParent)
		if err != nil {
			return "", err
		}
		return filepath.Join(absParent, filepath.Base(path)), nil
	}

	return filepath.Abs(path)
}

// containsTraversal checks for .. components in the path.
func containsTraversal(path string) bool {
	for _, part := range strings.Split(filepath.ToSlash(path), "/") {
		if part == ".." {
			return true
		}
	}
	return false
}
