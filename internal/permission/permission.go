// ABOUTME: Permission checker supporting normal, yolo, and plan modes
// ABOUTME: Controls tool execution access based on mode and allow/deny rules

package permission

import (
	"fmt"
	"strings"
)

// Mode determines the permission checking behavior.
type Mode int

const (
	ModeNormal Mode = iota // Prompt user for dangerous operations
	ModeYolo               // Skip all prompts (--yolo)
	ModePlan               // Read-only: block write/bash
)

// String returns the mode name.
func (m Mode) String() string {
	switch m {
	case ModeNormal:
		return "normal"
	case ModeYolo:
		return "yolo"
	case ModePlan:
		return "plan"
	default:
		return "unknown"
	}
}

// Rule defines a permission rule for a specific tool pattern.
type Rule struct {
	Tool    string // Tool name pattern (supports *)
	Allow   bool   // true = allow, false = deny
	Message string // Custom message for deny
}

// AskFunc is called when user confirmation is needed in normal mode.
type AskFunc func(tool string, args map[string]any) (bool, error)

// Checker validates tool execution permissions.
type Checker struct {
	mode       Mode
	allowRules []Rule
	denyRules  []Rule
	globRules  []GlobRule
	askFn      AskFunc
}

// NewChecker creates a Checker with the given mode and ask function.
func NewChecker(mode Mode, askFn AskFunc) *Checker {
	return &Checker{
		mode:  mode,
		askFn: askFn,
	}
}

// SetMode updates the permission mode.
func (c *Checker) SetMode(mode Mode) {
	c.mode = mode
}

// SetAskFn sets or replaces the interactive ask function.
func (c *Checker) SetAskFn(fn AskFunc) {
	c.askFn = fn
}

// Mode returns the current permission mode.
func (c *Checker) Mode() Mode {
	return c.mode
}

// AddAllowRule adds a rule that permits a tool.
func (c *Checker) AddAllowRule(rule Rule) {
	rule.Allow = true
	c.allowRules = append(c.allowRules, rule)
}

// AddDenyRule adds a rule that blocks a tool.
func (c *Checker) AddDenyRule(rule Rule) {
	rule.Allow = false
	c.denyRules = append(c.denyRules, rule)
}

// readOnlyTools lists tools that are always allowed in plan mode.
var readOnlyTools = map[string]bool{
	"read": true, "grep": true, "find": true, "ls": true,
}

// Check validates whether a tool can execute.
// Returns nil if allowed, error with reason if blocked.
func (c *Checker) Check(tool string, args map[string]any) error {
	// Plan mode: only read-only tools
	if c.mode == ModePlan && !readOnlyTools[tool] {
		return fmt.Errorf("tool %q blocked in plan mode; switch to edit mode (Shift+Tab)", tool)
	}

	// Check deny rules first
	for _, rule := range c.denyRules {
		if matchTool(rule.Tool, tool) {
			msg := rule.Message
			if msg == "" {
				msg = fmt.Sprintf("tool %q denied by rule", tool)
			}
			return fmt.Errorf("%s", msg)
		}
	}

	// Check allow rules
	for _, rule := range c.allowRules {
		if matchTool(rule.Tool, tool) {
			return nil
		}
	}

	// Evaluate glob rules (deny -> ask -> allow)
	if len(c.globRules) > 0 {
		specifier := ExtractSpecifier(tool, args)
		switch evaluateGlobRules(c.globRules, tool, specifier) {
		case ActionDeny:
			return fmt.Errorf("tool %q with specifier %q denied by glob rule", tool, specifier)
		case ActionAllow:
			return nil
		case ActionAsk:
			if c.askFn != nil {
				allowed, err := c.askFn(tool, args)
				if err != nil {
					return fmt.Errorf("permission check failed: %w", err)
				}
				if !allowed {
					return fmt.Errorf("tool %q denied by user", tool)
				}
				return nil
			}
		}
	}

	// Yolo mode: allow everything
	if c.mode == ModeYolo {
		return nil
	}

	// Normal mode for write tools: ask user or deny if no askFn
	if c.mode == ModeNormal && !readOnlyTools[tool] {
		if c.askFn == nil {
			return fmt.Errorf("tool %q denied: no interactive approval available", tool)
		}
		allowed, err := c.askFn(tool, args)
		if err != nil {
			return fmt.Errorf("permission check failed: %w", err)
		}
		if !allowed {
			return fmt.Errorf("tool %q denied by user", tool)
		}
	}

	return nil
}

// matchTool checks if a pattern matches a tool name.
// Supports * as wildcard.
func matchTool(pattern, tool string) bool {
	if pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(tool, pattern[:len(pattern)-1])
	}
	return pattern == tool
}
