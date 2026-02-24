// ABOUTME: Security utilities for git command validation
// ABOUTME: Prevents command injection by validating git subcommands and arguments

package git

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"
)

// allowedGitCommands defines safe git subcommands that can be executed
var allowedGitCommands = map[string]bool{
	"status":           true,
	"diff":             true,
	"log":              true,
	"rev-parse":        true,
	"ls-files":         true,
	"show-toplevel":    true,
	"worktree":         true,
	"branch":           true,
	"checkout":         true,
	"stash":            true,
	"init":             true,
	"add":              true,
	"commit":           true,
	"push":             true,
	"pull":             true,
	"fetch":            true,
	"merge":            true,
	"rebase":           true,
	"reset":            true,
	"clean":            true,
	"config":           true,
	"remote":           true,
	"tag":              true,
	"describe":         true,
	"reflog":           true,
	"gc":               true,
}

// allowedGitOptions defines safe git options and flags
var allowedGitOptions = map[string]bool{
	"--porcelain":         true,
	"--cached":            true,
	"--others":            true,
	"--exclude-standard":  true,
	"--abbrev-ref":        true,
	"--short":             true,
	"--stat":              true,
	"--oneline":           true,
	"--graph":             true,
	"--decorate":          true,
	"--all":               true,
	"-C":                  true, // Change directory
	"--git-dir":           true,
	"--work-tree":         true,
	"--bare":              true,
	"--no-pager":            true,
	"--show-toplevel":       true,
	"--is-inside-work-tree": true,
	"--force":               true,
	"--no-edit":             true, // Merge without editor
	"-b":                    true, // Branch flag for worktree/checkout
	"-d":                    true, // Delete branch
	"-D":                    true, // Force delete branch
	"-m":                    true, // Message flag for commit/tag
	"-u":                    true, // Set upstream for push
	"--help":                true,
	"--version":             true,
	"--":                    true, // End of options marker
}

// sanitizeGitArgs validates and sanitizes git command arguments to prevent injection
func sanitizeGitArgs(args []string) ([]string, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("no git command specified")
	}

	// Validate the main git subcommand
	subcommand := args[0]
	if !allowedGitCommands[subcommand] {
		return nil, fmt.Errorf("git subcommand not allowed: %q", subcommand)
	}

	sanitized := make([]string, 0, len(args))
	sanitized = append(sanitized, subcommand)

	// Process remaining arguments
	for i := 1; i < len(args); i++ {
		arg := args[i]
		
		// Skip empty arguments
		if arg == "" {
			continue
		}

		if err := validateGitArg(arg); err != nil {
			return nil, fmt.Errorf("invalid git argument %q: %w", arg, err)
		}

		sanitized = append(sanitized, arg)
	}

	return sanitized, nil
}

// validateGitArg validates a single git argument
func validateGitArg(arg string) error {
	// Check for command substitution patterns (before general dangerous chars
	// so the error message is specific)
	if strings.Contains(arg, "$(") || strings.Contains(arg, "`") {
		return fmt.Errorf("command substitution not allowed")
	}

	// Check for dangerous shell metacharacters
	dangerousChars := []string{";", "|", "&", "$", "(", ")", "{", "}", "<", ">", "\\"}
	for _, char := range dangerousChars {
		if strings.Contains(arg, char) {
			return fmt.Errorf("contains dangerous character: %s", char)
		}
	}

	// If it starts with -, validate as option
	if strings.HasPrefix(arg, "-") {
		return validateGitOption(arg)
	}

	// If it looks like a path, validate it
	if strings.Contains(arg, "/") || strings.Contains(arg, "\\") {
		return validateGitPath(arg)
	}

	// Validate as general string (branch names, commit hashes, etc.)
	return validateGitString(arg)
}

// validateGitOption validates git command line options
func validateGitOption(option string) error {
	// Split option and value (e.g., "--format=oneline")
	parts := strings.SplitN(option, "=", 2)
	optionName := parts[0]

	if !allowedGitOptions[optionName] {
		// Allow some common patterns
		if strings.HasPrefix(optionName, "--format") ||
		   strings.HasPrefix(optionName, "--pretty") ||
		   strings.HasPrefix(optionName, "--grep") {
			return nil // These are generally safe
		}
		return fmt.Errorf("git option not allowed: %s", optionName)
	}

	// If option has a value, validate it
	if len(parts) > 1 {
		return validateGitString(parts[1])
	}

	return nil
}

// validateGitPath validates file/directory paths used in git commands
func validateGitPath(path string) error {
	// Check for path traversal
	if strings.Contains(path, "..") {
		return fmt.Errorf("path traversal not allowed")
	}

	// Absolute paths are allowed; the dangerous-character check above already
	// prevents shell injection. Restricting to specific prefixes would break
	// worktree operations that use repo-relative absolute paths.

	// Clean the path to normalize it
	cleaned := filepath.Clean(path)
	if cleaned != path && !strings.HasSuffix(path, "/") {
		return fmt.Errorf("path contains unnecessary elements")
	}

	return nil
}

// validateGitString validates general string arguments (branch names, commit hashes, etc.)
func validateGitString(s string) error {
	// Check length
	if len(s) > 255 {
		return fmt.Errorf("string too long (max 255 characters)")
	}

	// Check for non-printable characters
	for _, r := range s {
		if !unicode.IsPrint(r) && !unicode.IsSpace(r) {
			return fmt.Errorf("contains non-printable character")
		}
	}

	// Additional validation for specific patterns
	if strings.HasPrefix(s, "-") {
		return fmt.Errorf("string cannot start with dash (use -- separator)")
	}

	return nil
}

// isValidWorktreeName validates worktree names to prevent injection
func isValidWorktreeName(name string) bool {
	if name == "" || len(name) > 64 {
		return false
	}

	// Only allow alphanumeric, dash, underscore, dot
	for _, r := range name {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '_' && r != '.' {
			return false
		}
	}

	// Don't allow consecutive dots or starting/ending with dot
	if strings.Contains(name, "..") || strings.HasPrefix(name, ".") || strings.HasSuffix(name, ".") {
		return false
	}

	return true
}