// ABOUTME: Path validation to restrict file access to allowed directories
// ABOUTME: Rejects path traversal attempts and enforces allowed directory prefixes

package permission

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Sandbox validates file paths against a set of allowed directories.
type Sandbox struct {
	allowed []string // Normalized absolute paths
}

// NewSandbox creates a Sandbox with the given allowed directories.
func NewSandbox(allowedDirs []string) (*Sandbox, error) {
	normalized := make([]string, 0, len(allowedDirs))
	for _, dir := range allowedDirs {
		abs, err := filepath.Abs(dir)
		if err != nil {
			return nil, fmt.Errorf("resolving path %q: %w", dir, err)
		}
		normalized = append(normalized, abs)
	}
	return &Sandbox{allowed: normalized}, nil
}

// ValidatePath checks that a path is within the allowed directories
// and does not contain path traversal components.
func (s *Sandbox) ValidatePath(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolving path %q: %w", path, err)
	}

	// Check for path traversal
	if containsTraversal(path) {
		return fmt.Errorf("path %q contains traversal components", path)
	}

	// Check prefix against allowed directories
	for _, allowed := range s.allowed {
		if strings.HasPrefix(abs, allowed) {
			return nil
		}
	}

	return fmt.Errorf("path %q is outside allowed directories", path)
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
