// ABOUTME: Permission checker supporting normal, accept-edits, plan, dont-ask, and yolo modes
// ABOUTME: Controls tool execution access based on mode, allow/deny rules, and glob patterns

package permission

import (
	"errors"
	"fmt"
	"strings"
	"sync"
)

// ErrNeedsApproval is returned by Check when the tool requires interactive
// user approval but no AskFunc is configured. Callers (e.g. the TUI) can
// detect this with IsNeedsApproval and present an approval dialog.
var ErrNeedsApproval = errors.New("tool requires interactive approval")

// IsNeedsApproval reports whether err (or any error in its chain) is
// ErrNeedsApproval.
func IsNeedsApproval(err error) bool {
	return errors.Is(err, ErrNeedsApproval)
}

// Mode determines the permission checking behavior.
type Mode int

const (
	ModeNormal      Mode = iota // Prompt user for dangerous operations
	ModeAcceptEdits             // Auto-allow edit/write; prompt for bash
	ModePlan                    // Read-only: block write/bash
	ModeDontAsk                 // Deny all non-allowed tools without prompting
	ModeYolo                    // Skip all prompts (--yolo)
)

// String returns the mode name.
func (m Mode) String() string {
	switch m {
	case ModeNormal:
		return "normal"
	case ModeAcceptEdits:
		return "accept-edits"
	case ModePlan:
		return "plan"
	case ModeDontAsk:
		return "dont-ask"
	case ModeYolo:
		return "yolo"
	default:
		return "unknown"
	}
}

// ParseMode converts a settings string to a Mode constant.
// Recognized values: "default"/"normal"/""→Normal, "acceptEdits"→AcceptEdits,
// "plan"→Plan, "dontAsk"→DontAsk, "bypassPermissions"→Yolo.
func ParseMode(s string) (Mode, error) {
	switch s {
	case "", "default", "normal":
		return ModeNormal, nil
	case "acceptEdits":
		return ModeAcceptEdits, nil
	case "plan":
		return ModePlan, nil
	case "dontAsk":
		return ModeDontAsk, nil
	case "bypassPermissions":
		return ModeYolo, nil
	default:
		return 0, fmt.Errorf("unknown permission mode: %q", s)
	}
}

// editWriteTools lists tools that are auto-allowed in accept-edits mode.
var editWriteTools = map[string]bool{
	"edit": true, "write": true, "notebook_edit": true,
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
// All exported methods are safe for concurrent use.
type Checker struct {
	mu         sync.RWMutex
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
	c.mu.Lock()
	defer c.mu.Unlock()
	c.mode = mode
}

// SetAskFn sets or replaces the interactive ask function.
func (c *Checker) SetAskFn(fn AskFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.askFn = fn
}

// Mode returns the current permission mode.
func (c *Checker) Mode() Mode {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.mode
}

// AddAllowRule adds a rule that permits a tool.
func (c *Checker) AddAllowRule(rule Rule) {
	c.mu.Lock()
	defer c.mu.Unlock()
	rule.Allow = true
	c.allowRules = append(c.allowRules, rule)
}

// AddGlobAllowRule adds a glob-based allow rule for a tool with specifier.
func (c *Checker) AddGlobAllowRule(tool, specifier string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.globRules = append(c.globRules, GlobRule{
		Tool:      tool,
		Specifier: specifier,
		Action:    ActionAllow,
	})
}

// AddDenyRule adds a rule that blocks a tool.
func (c *Checker) AddDenyRule(rule Rule) {
	c.mu.Lock()
	defer c.mu.Unlock()
	rule.Allow = false
	c.denyRules = append(c.denyRules, rule)
}

// readOnlyTools lists tools that are always allowed in plan mode.
var readOnlyTools = map[string]bool{
	"read": true, "grep": true, "find": true, "ls": true,
}

// Check validates whether a tool can execute.
// Returns nil if allowed, error with reason if blocked.
// Returns an error wrapping ErrNeedsApproval when the tool requires
// interactive approval but no AskFunc is configured.
func (c *Checker) Check(tool string, args map[string]any) error {
	// Evaluate rules under read lock; capture askFn for potential callback.
	verdict, askFn := c.evaluate(tool, args)
	switch verdict {
	case verdictAllow:
		return nil
	case verdictAsk:
		if askFn == nil {
			return fmt.Errorf("tool %q: %w", tool, ErrNeedsApproval)
		}
		allowed, err := askFn(tool, args)
		if err != nil {
			return fmt.Errorf("permission check failed: %w", err)
		}
		if !allowed {
			return fmt.Errorf("tool %q denied by user", tool)
		}
		return nil
	default:
		// verdictDeny carries the error message
		return verdict.err
	}
}

// verdict is the result of rule evaluation.
type verdict struct {
	kind int // 0=deny, 1=allow, 2=ask
	err  error
}

var (
	verdictAllow = verdict{kind: 1}
	verdictAsk   = verdict{kind: 2}
)

func denyVerdict(err error) verdict { return verdict{kind: 0, err: err} }

// evaluate checks rules under RLock and returns a verdict plus the askFn.
// The caller invokes askFn (if needed) outside the lock to avoid blocking
// while holding the mutex.
func (c *Checker) evaluate(tool string, args map[string]any) (verdict, AskFunc) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Plan mode: only read-only tools
	if c.mode == ModePlan && !readOnlyTools[tool] {
		return denyVerdict(fmt.Errorf("tool %q blocked in plan mode; switch to edit mode (Shift+Tab)", tool)), nil
	}

	// Check deny rules first
	for _, rule := range c.denyRules {
		if matchTool(rule.Tool, tool) {
			msg := rule.Message
			if msg == "" {
				msg = fmt.Sprintf("tool %q denied by rule", tool)
			}
			return denyVerdict(fmt.Errorf("%s", msg)), nil
		}
	}

	// Check allow rules
	for _, rule := range c.allowRules {
		if matchTool(rule.Tool, tool) {
			return verdictAllow, nil
		}
	}

	// Evaluate glob rules (deny -> ask -> allow)
	if len(c.globRules) > 0 {
		specifier := ExtractSpecifier(tool, args)
		switch evaluateGlobRules(c.globRules, tool, specifier) {
		case ActionDeny:
			return denyVerdict(fmt.Errorf("tool %q with specifier %q denied by glob rule", tool, specifier)), nil
		case ActionAllow:
			return verdictAllow, nil
		case ActionAsk:
			return verdictAsk, c.askFn
		}
	}

	// Yolo mode: allow everything
	if c.mode == ModeYolo {
		return verdictAllow, nil
	}

	// AcceptEdits mode: auto-allow edit/write tools, ask for others
	if c.mode == ModeAcceptEdits && editWriteTools[tool] {
		return verdictAllow, nil
	}

	// DontAsk mode: deny non-read-only tools without prompting
	if c.mode == ModeDontAsk && !readOnlyTools[tool] {
		return denyVerdict(fmt.Errorf("tool %q denied in dont-ask mode", tool)), nil
	}

	// Normal + AcceptEdits for non-edit tools: ask user
	if (c.mode == ModeNormal || c.mode == ModeAcceptEdits) && !readOnlyTools[tool] {
		return verdictAsk, c.askFn
	}

	return verdictAllow, nil
}

// Rules returns a combined slice of allow and deny rules.
func (c *Checker) Rules() []Rule {
	c.mu.RLock()
	defer c.mu.RUnlock()
	rules := make([]Rule, 0, len(c.allowRules)+len(c.denyRules))
	rules = append(rules, c.allowRules...)
	rules = append(rules, c.denyRules...)
	return rules
}

// RemoveRule removes the first rule matching the given tool name from
// either the allow or deny lists. Returns true if a rule was removed.
func (c *Checker) RemoveRule(tool string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i, r := range c.allowRules {
		if r.Tool == tool {
			c.allowRules = append(c.allowRules[:i], c.allowRules[i+1:]...)
			return true
		}
	}
	for i, r := range c.denyRules {
		if r.Tool == tool {
			c.denyRules = append(c.denyRules[:i], c.denyRules[i+1:]...)
			return true
		}
	}
	return false
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
