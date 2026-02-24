// ABOUTME: Security tests for git command validation
// ABOUTME: Tests command injection prevention and argument sanitization

package git

import (
	"testing"
)

func TestSanitizeGitArgs(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid git status",
			args:        []string{"status"},
			expectError: false,
		},
		{
			name:        "valid git diff with options",
			args:        []string{"diff", "--stat"},
			expectError: false,
		},
		{
			name:        "command injection attempt with semicolon",
			args:        []string{"status", "; rm -rf /"},
			expectError: true,
			errorMsg:    "dangerous character",
		},
		{
			name:        "command injection attempt with pipe",
			args:        []string{"log", "| cat /etc/passwd"},
			expectError: true,
			errorMsg:    "dangerous character",
		},
		{
			name:        "command injection attempt with backtick",
			args:        []string{"status", "`whoami`"},
			expectError: true,
			errorMsg:    "command substitution",
		},
		{
			name:        "command injection attempt with dollar",
			args:        []string{"log", "$(whoami)"},
			expectError: true,
			errorMsg:    "command substitution",
		},
		{
			name:        "invalid git subcommand",
			args:        []string{"rm", "-rf", "/"},
			expectError: true,
			errorMsg:    "not allowed",
		},
		{
			name:        "empty args",
			args:        []string{},
			expectError: true,
			errorMsg:    "no git command",
		},
		{
			name:        "valid worktree command",
			args:        []string{"worktree", "list", "--porcelain"},
			expectError: false,
		},
		{
			name:        "valid rev-parse with options",
			args:        []string{"rev-parse", "--abbrev-ref", "HEAD"},
			expectError: false,
		},
		{
			name:        "path traversal attempt",
			args:        []string{"log", "../../../etc/passwd"},
			expectError: true,
			errorMsg:    "path traversal",
		},
		{
			name:        "valid file path",
			args:        []string{"add", "src/main.go"},
			expectError: false,
		},
		{
			name:        "dangerous option not in allowlist",
			args:        []string{"config", "--global", "user.email", "evil@example.com"},
			expectError: true,
			errorMsg:    "not allowed",
		},
		{
			name:        "valid config read",
			args:        []string{"config", "user.name"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := sanitizeGitArgs(tt.args)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing %q, but got no error", tt.errorMsg)
					return
				}
				if tt.errorMsg != "" && !containsString(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got: %v", tt.errorMsg, err)
				}
				if result != nil {
					t.Errorf("Expected nil result on error, got: %v", result)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
					return
				}
				if result == nil {
					t.Error("Expected non-nil result")
					return
				}
				if len(result) == 0 {
					t.Error("Expected non-empty result")
				}
			}
		})
	}
}

func TestIsValidWorktreeName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid simple name", "feature", true},
		{"valid with dash", "feature-branch", true},
		{"valid with underscore", "feature_branch", true},
		{"valid with dot", "v1.0", true},
		{"empty string", "", false},
		{"too long", "this-is-a-very-long-worktree-name-that-exceeds-the-maximum-allowed-length-of-64-characters", false},
		{"starts with dot", ".hidden", false},
		{"ends with dot", "feature.", false},
		{"consecutive dots", "feature..branch", false},
		{"contains slash", "feature/branch", false},
		{"contains backslash", "feature\\branch", false},
		{"contains space", "feature branch", false},
		{"contains special chars", "feature@branch", false},
		{"starts with dash", "-feature", true}, // This should be allowed
		{"only dots", "...", false},
		{"single dot", ".", false},
		{"double dot", "..", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidWorktreeName(tt.input)
			if result != tt.expected {
				t.Errorf("isValidWorktreeName(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestValidateGitArg(t *testing.T) {
	tests := []struct {
		name        string
		arg         string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "normal filename",
			arg:         "main.go",
			expectError: false,
		},
		{
			name:        "branch name",
			arg:         "feature-branch",
			expectError: false,
		},
		{
			name:        "commit hash",
			arg:         "abc123def456",
			expectError: false,
		},
		{
			name:        "semicolon injection",
			arg:         "file.txt; rm -rf /",
			expectError: true,
			errorMsg:    "dangerous character",
		},
		{
			name:        "pipe injection",
			arg:         "file.txt | cat",
			expectError: true,
			errorMsg:    "dangerous character",
		},
		{
			name:        "command substitution",
			arg:         "$(whoami)",
			expectError: true,
			errorMsg:    "command substitution",
		},
		{
			name:        "backtick substitution",
			arg:         "`id`",
			expectError: true,
			errorMsg:    "command substitution",
		},
		{
			name:        "valid option",
			arg:         "--stat",
			expectError: false,
		},
		{
			name:        "invalid option",
			arg:         "--exec=evil",
			expectError: true,
			errorMsg:    "not allowed",
		},
		{
			name:        "path traversal",
			arg:         "../../../etc/passwd",
			expectError: true,
			errorMsg:    "path traversal",
		},
		{
			name:        "absolute path in safe location",
			arg:         "/tmp/safe-file.txt",
			expectError: false,
		},
		{
			name:        "absolute path in other location",
			arg:         "/etc/passwd",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateGitArg(tt.arg)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing %q, but got no error", tt.errorMsg)
					return
				}
				if tt.errorMsg != "" && !containsString(err.Error(), tt.errorMsg) {
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

// containsString checks if a string contains a substring (case-insensitive)
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || 
			len(s) > len(substr) && 
			stringContainsIgnoreCase(s, substr))
}

func stringContainsIgnoreCase(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if toLowerCase(s[i+j]) != toLowerCase(substr[j]) {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func toLowerCase(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b + 32
	}
	return b
}