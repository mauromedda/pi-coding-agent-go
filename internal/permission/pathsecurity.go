// ABOUTME: Enhanced path security validation to prevent directory traversal
// ABOUTME: Validates file paths and prevents access outside allowed directories

package permission

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SecurePathValidator provides secure path validation with allowlist-based access control
type SecurePathValidator struct {
	allowedPrefixes []string // Normalized absolute paths with trailing separator
}

// NewSecurePathValidator creates a new path validator with allowed directories
func NewSecurePathValidator(allowedDirs []string) (*SecurePathValidator, error) {
	normalized := make([]string, 0, len(allowedDirs))
	
	for _, dir := range allowedDirs {
		// Resolve symlinks for consistency
		resolved, err := filepath.EvalSymlinks(dir)
		if err != nil {
			// If directory doesn't exist, use the raw path
			resolved = dir
		}
		
		abs, err := filepath.Abs(resolved)
		if err != nil {
			return nil, fmt.Errorf("resolving path %q: %w", dir, err)
		}
		
		// Ensure trailing separator for correct prefix matching
		if !strings.HasSuffix(abs, string(filepath.Separator)) {
			abs += string(filepath.Separator)
		}
		
		normalized = append(normalized, abs)
	}
	
	return &SecurePathValidator{
		allowedPrefixes: normalized,
	}, nil
}

// DefaultAllowedDirectories returns the default set of allowed directories for pi-go
func DefaultAllowedDirectories() []string {
	home, _ := os.UserHomeDir()
	return []string{
		filepath.Join(home, ".pi-go"),
		filepath.Join(home, ".claude"),
		"/tmp/pi-go-sessions",
		"/var/tmp/pi-go-temp",
		".", // Current working directory
	}
}

// ValidateReadPath validates a path for read operations
func (v *SecurePathValidator) ValidateReadPath(path string) error {
	return v.validatePath(path, "read")
}

// ValidateWritePath validates a path for write operations
func (v *SecurePathValidator) ValidateWritePath(path string) error {
	// Additional restrictions for write operations
	if err := v.validatePath(path, "write"); err != nil {
		return err
	}
	
	// Don't allow writing to system directories.
	// Use trailing separator to prevent false positives: "/etc" must not match "/etc-config/".
	systemPaths := []string{"/etc", "/usr", "/bin", "/sbin", "/boot", "/proc", "/sys"}
	absPath, _ := filepath.Abs(path)

	for _, syspath := range systemPaths {
		if absPath == syspath || strings.HasPrefix(absPath, syspath+string(filepath.Separator)) {
			return fmt.Errorf("write access denied to system directory: %s", path)
		}
	}
	
	return nil
}

// validatePath performs core path validation
func (v *SecurePathValidator) validatePath(path, operation string) error {
	if path == "" {
		return fmt.Errorf("empty path not allowed")
	}
	
	// Check for obvious traversal attempts
	if err := checkTraversalPatterns(path); err != nil {
		return fmt.Errorf("path traversal detected in %s operation: %w", operation, err)
	}
	
	// Resolve path to handle symlinks and relative paths
	resolved, err := resolveSecurePath(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path for %s operation: %w", operation, err)
	}
	
	// Check against allowed prefixes
	if err := v.checkAllowedPrefix(resolved); err != nil {
		return fmt.Errorf("%s access denied: %w", operation, err)
	}
	
	return nil
}

// checkTraversalPatterns checks for various path traversal patterns
func checkTraversalPatterns(path string) error {
	// Normalize path separators for consistent checking
	normalizedPath := filepath.ToSlash(path)
	
	// Check for various traversal patterns
	dangerousPatterns := []string{
		"../",
		"..\\",
		"/..",
		"\\..",
		"..%2f",
		"..%2F",
		"..%5c",
		"..%5C",
		"%2e%2e%2f",
		"%2e%2e%5c",
	}
	
	pathLower := strings.ToLower(normalizedPath)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(pathLower, pattern) {
			return fmt.Errorf("contains dangerous pattern: %s", pattern)
		}
	}
	
	// Check for null bytes
	if strings.Contains(path, "\x00") {
		return fmt.Errorf("null byte in path")
	}
	
	// Check path components
	parts := strings.Split(normalizedPath, "/")
	for _, part := range parts {
		if part == ".." {
			return fmt.Errorf("path contains .. component")
		}
		if part == "." && len(parts) > 1 {
			// Allow single "." for current directory, but not in compound paths
			return fmt.Errorf("path contains . component")
		}
	}
	
	return nil
}

// resolveSecurePath safely resolves a path, handling non-existent files
func resolveSecurePath(path string) (string, error) {
	// Try to resolve the full path
	resolved, err := filepath.EvalSymlinks(path)
	if err == nil {
		// Path exists and symlinks resolved
		return filepath.Abs(resolved)
	}
	
	// Path doesn't exist, resolve parent directory
	if os.IsNotExist(err) {
		parent := filepath.Dir(path)
		resolvedParent, err := filepath.EvalSymlinks(parent)
		if err != nil {
			// Parent also doesn't exist, use absolute path as best effort
			return filepath.Abs(path)
		}
		
		absParent, err := filepath.Abs(resolvedParent)
		if err != nil {
			return "", err
		}
		
		return filepath.Join(absParent, filepath.Base(path)), nil
	}
	
	// Other error resolving symlinks
	return filepath.Abs(path)
}

// checkAllowedPrefix verifies the path is within allowed directories
func (v *SecurePathValidator) checkAllowedPrefix(resolvedPath string) error {
	// Ensure path has trailing separator for comparison
	pathWithSep := resolvedPath
	if !strings.HasSuffix(pathWithSep, string(filepath.Separator)) {
		pathWithSep += string(filepath.Separator)
	}
	
	for _, allowedPrefix := range v.allowedPrefixes {
		// Check if path is within allowed directory
		if strings.HasPrefix(pathWithSep, allowedPrefix) {
			return nil
		}
		
		// Check if path exactly matches allowed directory (without trailing separator)
		trimmedPrefix := strings.TrimSuffix(allowedPrefix, string(filepath.Separator))
		if resolvedPath == trimmedPrefix {
			return nil
		}
	}
	
	return fmt.Errorf("path %q is outside allowed directories", resolvedPath)
}

// ValidateFilename validates a filename for security issues
func ValidateFilename(filename string) error {
	if filename == "" {
		return fmt.Errorf("empty filename")
	}
	
	// Check for path separators (filename shouldn't be a path)
	if strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		return fmt.Errorf("filename cannot contain path separators")
	}
	
	// Check for dangerous filenames
	dangerous := []string{".", "..", "con", "prn", "aux", "nul"}
	filenameLower := strings.ToLower(filename)
	
	for _, danger := range dangerous {
		if filenameLower == danger {
			return fmt.Errorf("reserved filename: %s", filename)
		}
	}
	
	// Check for Windows reserved names
	windowsReserved := []string{"com1", "com2", "com3", "com4", "com5", "com6", "com7", "com8", "com9",
		"lpt1", "lpt2", "lpt3", "lpt4", "lpt5", "lpt6", "lpt7", "lpt8", "lpt9"}
	
	for _, reserved := range windowsReserved {
		if filenameLower == reserved {
			return fmt.Errorf("Windows reserved filename: %s", filename)
		}
	}
	
	// Check for null bytes and control characters
	for _, r := range filename {
		if r == 0 || (r < 32 && r != '\t' && r != '\n' && r != '\r') {
			return fmt.Errorf("filename contains control character")
		}
	}
	
	// Check length
	if len(filename) > 255 {
		return fmt.Errorf("filename too long (max 255 characters)")
	}
	
	return nil
}