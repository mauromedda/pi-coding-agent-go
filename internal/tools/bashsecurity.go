// ABOUTME: Security utilities for bash command validation and sandboxing
// ABOUTME: Prevents dangerous command execution and limits shell access

package tools

import (
	"fmt"
	"regexp"
	"strings"
)

// dangerousCommands lists shell commands that should be blocked or restricted
var dangerousCommands = map[string]bool{
	"rm":         true,
	"rmdir":      true,
	"delete":     true,
	"del":        true,
	"format":     true,
	"fdisk":      true,
	"mkfs":       true,
	// dd: moved to allowedCommands (safe for read operations; dangerous only with of= to devices)
	"shutdown":   true,
	"reboot":     true,
	"halt":       true,
	"poweroff":   true,
	"su":         true,
	"sudo":       true,
	"passwd":     true,
	"chsh":       true,
	"chfn":       true,
	"usermod":    true,
	"useradd":    true,
	"userdel":    true,
	"groupadd":   true,
	"groupdel":   true,
	"chmod":      true,  // Can be dangerous
	"chown":      true,  // Can be dangerous
	"mount":      true,
	"umount":     true,
	"crontab":    true,
	"at":         true,
	"batch":      true,
	"nc":         true,  // netcat
	"netcat":     true,
	"ncat":       true,
	"socat":      true,
	"telnet":     true,
	"ssh":        true,
	"scp":        true,
	"rsync":      true,
	"curl":       true,  // Can be used for data exfiltration
	"wget":       true,  // Can be used for data exfiltration
	"lynx":       true,
	"links":      true,
	"w3m":        true,
}

// dangerousPatterns are regex patterns that indicate dangerous shell constructs.
// NOTE: pipes (|), chaining (&&, ||), fd redirection (>&2), and /dev/null access
// are intentionally NOT blocked: they are standard shell constructs used by
// legitimate commands. Security comes from the command allowlist/blocklist.
var dangerousPatterns = []*regexp.Regexp{
	regexp.MustCompile(`\$\(`),                    // Command substitution $(...)
	regexp.MustCompile("`[^`]*`"),                 // Command substitution `...`
	regexp.MustCompile(`\${[^}]*}`),               // Variable expansion with complex expressions
	regexp.MustCompile(`;\s*\w+`),                 // Command chaining with semicolon
	regexp.MustCompile(`/etc/passwd`),             // Access to sensitive files
	regexp.MustCompile(`/etc/shadow`),             // Access to sensitive files
	regexp.MustCompile(`/etc/hosts`),              // Access to sensitive files
	regexp.MustCompile(`/proc/`),                  // Access to proc filesystem
	regexp.MustCompile(`/sys/`),                   // Access to sys filesystem
	regexp.MustCompile(`\.\./`),                   // Directory traversal
	regexp.MustCompile(`~[^/\s]*/`),               // Access to other users' home dirs
	regexp.MustCompile(`\$HOME/\.\w+`),            // Access to hidden files in home
	regexp.MustCompile(`exec\s+\d*[<>]`),          // File descriptor redirection via exec
	regexp.MustCompile(`\{[^}]*;[^}]*\}`),         // Brace expansion with command execution
}

// allowedCommands lists safe commands that are generally okay to run
var allowedCommands = map[string]bool{
	"echo":     true,
	"printf":   true,
	"cat":      true,
	"head":     true,
	"tail":     true,
	"grep":     true,
	"egrep":    true,
	"fgrep":    true,
	"sed":      true,
	"awk":      true,
	"cut":      true,
	"sort":     true,
	"uniq":     true,
	"wc":       true,
	"tr":       true,
	"basename": true,
	"dirname":  true,
	"pwd":      true,
	"whoami":   true,
	"id":       true,
	"date":     true,
	"uptime":   true,
	"uname":    true,
	"env":      true,
	"printenv": true,
	"which":    true,
	"type":     true,
	"file":     true,
	"stat":     true,
	"ls":       true,
	"find":     true,
	"locate":   true,
	"tree":     true,
	"df":       true,
	"du":       true,
	"ps":       true,
	"top":      true,
	"htop":     true,
	"free":     true,
	"history":  true,
	"alias":    true,
	"sleep":    true,
	"dd":       true,
	"true":     true,
	"false":    true,
	"test":     true,
	"expr":     true,
	"bc":       true,
	"calc":     true,
	"jq":       true,
	"yq":       true,
	"xmllint":  true,
	"git":      true, // Git commands should be safe
	"make":     true, // Build tools
	"cmake":    true,
	"ninja":    true,
	"go":       true, // Programming language tools
	"python":   true,
	"python3":  true,
	"node":     true,
	"npm":      true,
	"yarn":     true,
	"pip":      true,
	"pip3":     true,
	"docker":   true, // Container tools (with restrictions)
	"kubectl":  true, // Kubernetes tools
}

// validateBashCommand validates a bash command for security issues
func validateBashCommand(command string) error {
	if command == "" {
		return fmt.Errorf("empty command")
	}

	// Check command length
	if len(command) > 10000 {
		return fmt.Errorf("command too long (max 10000 characters)")
	}

	// Check for dangerous patterns
	for _, pattern := range dangerousPatterns {
		if pattern.MatchString(command) {
			return fmt.Errorf("command contains dangerous pattern: %s", pattern.String())
		}
	}

	// Extract the primary command (first word)
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return fmt.Errorf("no command found")
	}

	primaryCmd := strings.ToLower(parts[0])

	// Check if the primary command is explicitly dangerous
	if dangerousCommands[primaryCmd] {
		return fmt.Errorf("command not allowed: %s", primaryCmd)
	}

	// If not in allowed list and not a built-in shell construct, be cautious
	if !allowedCommands[primaryCmd] && !isShellBuiltin(primaryCmd) {
		// Allow some flexibility for unknown but potentially safe commands
		if !looksLikeSafeCommand(primaryCmd) {
			return fmt.Errorf("unknown command not in allow list: %s", primaryCmd)
		}
	}

	// Additional validation for specific commands
	switch primaryCmd {
	case "find":
		return validateFindCommand(command)
	case "grep", "egrep", "fgrep":
		return validateGrepCommand(command)
	case "sed":
		return validateSedCommand(command)
	case "awk":
		return validateAwkCommand(command)
	}

	// Validate commands in pipelines and chains (|, &&, ||).
	// The primary command passed above; now check any chained/piped commands.
	if err := validatePipelineCommands(command); err != nil {
		return err
	}

	return nil
}

// validatePipelineCommands splits on |, &&, || and validates each segment's
// primary command against the dangerous/allowed lists.
func validatePipelineCommands(command string) error {
	// Split on pipe and chain operators.
	// Use a simple regex to split on |, &&, || (but not inside quotes).
	segments := pipelineSplitter.Split(command, -1)
	if len(segments) <= 1 {
		return nil // No pipeline, already validated above
	}

	// Skip the first segment (already validated as primary command).
	for _, seg := range segments[1:] {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		parts := strings.Fields(seg)
		if len(parts) == 0 {
			continue
		}
		cmd := strings.ToLower(parts[0])
		if dangerousCommands[cmd] {
			return fmt.Errorf("command contains dangerous pattern: piped/chained command %q is not allowed", cmd)
		}
	}
	return nil
}

var pipelineSplitter = regexp.MustCompile(`\s*(?:\|{1,2}|&&)\s*`)

// isShellBuiltin checks if a command is a shell builtin
func isShellBuiltin(cmd string) bool {
	builtins := []string{
		"cd", "pwd", "echo", "printf", "read", "exit", "return",
		"break", "continue", "if", "then", "else", "elif", "fi",
		"for", "while", "until", "do", "done", "case", "esac",
		"function", "local", "export", "unset", "readonly",
		"declare", "typeset", "let", "eval", "exec", "source",
		".", ":", "true", "false", "test", "[", "[[", "]]",
	}

	for _, builtin := range builtins {
		if cmd == builtin {
			return true
		}
	}
	return false
}

// looksLikeSafeCommand uses heuristics to determine if an unknown command might be safe
func looksLikeSafeCommand(cmd string) bool {
	// Commands starting with certain prefixes are often safe
	safePrefixes := []string{
		"npm-", "node-", "py-", "python-", "go-", "java-",
		"rust-", "cargo-", "gem-", "bundle-", "php-",
	}

	for _, prefix := range safePrefixes {
		if strings.HasPrefix(cmd, prefix) {
			return true
		}
	}

	// Commands ending with certain suffixes are often safe
	safeSuffixes := []string{
		"-config", "-version", "-help", "-info", "-check",
		"-lint", "-fmt", "-test", "-build", "-run",
	}

	for _, suffix := range safeSuffixes {
		if strings.HasSuffix(cmd, suffix) {
			return true
		}
	}

	return false
}

// validateFindCommand validates find command arguments
func validateFindCommand(command string) error {
	// Check for dangerous find operations
	if strings.Contains(command, "-exec") && strings.Contains(command, "rm") {
		return fmt.Errorf("find with -exec rm is not allowed")
	}

	if strings.Contains(command, "-delete") {
		return fmt.Errorf("find with -delete is not allowed")
	}

	return nil
}

// validateGrepCommand validates grep command arguments
func validateGrepCommand(command string) error {
	// Generally safe, but check for file access patterns
	if strings.Contains(command, "/etc/") {
		return fmt.Errorf("grep access to /etc/ directory is restricted")
	}

	if strings.Contains(command, "/proc/") {
		return fmt.Errorf("grep access to /proc/ directory is restricted")
	}

	return nil
}

// validateSedCommand validates sed command arguments
func validateSedCommand(command string) error {
	// Check for dangerous sed operations
	if strings.Contains(command, "-i") {
		return fmt.Errorf("sed in-place editing (-i) is not allowed")
	}

	// Check for file write operations
	if strings.Contains(command, "w ") || strings.Contains(command, "w\t") {
		return fmt.Errorf("sed write operations are not allowed")
	}

	return nil
}

// validateAwkCommand validates awk command arguments
func validateAwkCommand(command string) error {
	// Check for file operations in awk
	if strings.Contains(command, "system(") {
		return fmt.Errorf("awk system() function is not allowed")
	}

	if strings.Contains(command, "print >") {
		return fmt.Errorf("awk file write operations are not allowed")
	}

	return nil
}

// restrictedEnvironment returns environment variables safe for command execution
func restrictedEnvironment() []string {
	// Only include safe environment variables
	safeVars := []string{
		"PATH=/usr/local/bin:/usr/bin:/bin",
		"HOME=/tmp/pi-go-sandbox",
		"USER=pi-go",
		"SHELL=/bin/bash",
		"TERM=xterm",
		"LANG=en_US.UTF-8",
		"TZ=UTC",
	}

	return safeVars
}

// sanitizeBashCommand performs basic sanitization on a bash command
func sanitizeBashCommand(command string) string {
	// Remove null bytes
	command = strings.ReplaceAll(command, "\x00", "")

	// Remove other control characters except newlines and tabs
	var result strings.Builder
	for _, r := range command {
		if r >= 32 || r == '\n' || r == '\t' {
			result.WriteRune(r)
		}
	}

	// Trim whitespace
	return strings.TrimSpace(result.String())
}