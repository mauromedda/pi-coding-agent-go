// ABOUTME: Glob-based permission rules with tool specifiers for fine-grained access control
// ABOUTME: Supports Bash(cmd *), Edit(/path/**), WebFetch(domain:host) patterns

package permission

import (
	"net/url"
	"path/filepath"
	"strings"
)

// Action represents a permission decision.
type Action int

const (
	ActionNone  Action = iota // No matching rule
	ActionAllow               // Explicitly allowed
	ActionDeny                // Explicitly denied
	ActionAsk                 // Requires user confirmation
)

// GlobRule is a permission rule with optional specifier matching.
type GlobRule struct {
	Tool      string // Tool name or pattern (supports * suffix)
	Specifier string // Optional: "npm run *", "/src/**", "domain:example.com"
	Action    Action
}

// parseGlobRule parses a rule string like "Bash(npm run *)" into a GlobRule.
func parseGlobRule(s string, action Action) GlobRule {
	rule := GlobRule{Action: action}

	// Check for specifier: Tool(specifier)
	if idx := strings.Index(s, "("); idx > 0 && strings.HasSuffix(s, ")") {
		rule.Tool = s[:idx]
		rule.Specifier = s[idx+1 : len(s)-1]
	} else {
		rule.Tool = s
	}

	return rule
}

// matchGlobRule checks if a rule matches a tool name and specifier.
func matchGlobRule(rule GlobRule, toolName, specifier string) bool {
	// Match tool name
	if !matchToolPattern(rule.Tool, toolName) {
		return false
	}

	// If rule has no specifier, it matches all specifiers
	if rule.Specifier == "" {
		return true
	}

	// Match specifier
	if specifier == "" {
		return false
	}

	// Wildcard at end: prefix match (e.g., "rm *" matches "rm -rf /")
	if strings.HasSuffix(rule.Specifier, " *") {
		prefix := rule.Specifier[:len(rule.Specifier)-2]
		if strings.HasPrefix(specifier, prefix) {
			return true
		}
	}

	// Wildcard at end without space: "npm*" matches "npm run test"
	if strings.HasSuffix(rule.Specifier, "*") {
		prefix := rule.Specifier[:len(rule.Specifier)-1]
		if strings.HasPrefix(specifier, prefix) {
			return true
		}
	}

	// Path pattern: "/**" suffix for recursive match
	if strings.HasSuffix(rule.Specifier, "/**") {
		prefix := rule.Specifier[:len(rule.Specifier)-3]
		if strings.HasPrefix(specifier, prefix) {
			return true
		}
	}

	// Try filepath.Match for simple glob patterns
	if matched, _ := filepath.Match(rule.Specifier, specifier); matched {
		return true
	}

	// Exact match
	return rule.Specifier == specifier
}

// matchToolPattern matches a tool name against a pattern.
func matchToolPattern(pattern, name string) bool {
	if pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(name, pattern[:len(pattern)-1])
	}
	return strings.EqualFold(pattern, name)
}

// ExtractSpecifier extracts the relevant specifier from tool arguments.
func ExtractSpecifier(toolName string, args map[string]any) string {
	switch strings.ToLower(toolName) {
	case "bash":
		if cmd, ok := args["command"].(string); ok {
			return cmd
		}
	case "edit", "write", "read":
		if path, ok := args["file_path"].(string); ok {
			return path
		}
	case "webfetch":
		if rawURL, ok := args["url"].(string); ok {
			u, err := url.Parse(rawURL)
			if err == nil {
				return "domain:" + u.Hostname()
			}
		}
	case "grep", "find", "ls":
		if path, ok := args["path"].(string); ok {
			return path
		}
	}
	return ""
}

// evaluateGlobRules evaluates rules in deny-first, ask-second, allow-third order.
func evaluateGlobRules(rules []GlobRule, toolName, specifier string) Action {
	// Pass 1: Deny
	for _, r := range rules {
		if r.Action == ActionDeny && matchGlobRule(r, toolName, specifier) {
			return ActionDeny
		}
	}
	// Pass 2: Ask
	for _, r := range rules {
		if r.Action == ActionAsk && matchGlobRule(r, toolName, specifier) {
			return ActionAsk
		}
	}
	// Pass 3: Allow
	for _, r := range rules {
		if r.Action == ActionAllow && matchGlobRule(r, toolName, specifier) {
			return ActionAllow
		}
	}
	return ActionNone
}

// NewCheckerFromSettings creates a Checker with glob-based rules from settings.
func NewCheckerFromSettings(mode Mode, askFn AskFunc, allow, deny, ask []string) *Checker {
	c := NewChecker(mode, askFn)

	var rules []GlobRule
	for _, s := range deny {
		rules = append(rules, parseGlobRule(s, ActionDeny))
	}
	for _, s := range ask {
		rules = append(rules, parseGlobRule(s, ActionAsk))
	}
	for _, s := range allow {
		rules = append(rules, parseGlobRule(s, ActionAllow))
	}

	c.globRules = rules
	return c
}
