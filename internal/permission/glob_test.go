// ABOUTME: Tests for glob-based permission rules with tool specifiers
// ABOUTME: Covers rule parsing, matching, specifier extraction, and deny-first evaluation

package permission

import (
	"testing"
)

func TestParseRule_Simple(t *testing.T) {
	tests := []struct {
		input string
		tool  string
		spec  string
	}{
		{"Bash", "Bash", ""},
		{"Edit", "Edit", ""},
		{"mcp__github__*", "mcp__github__*", ""},
		{"Bash(npm run *)", "Bash", "npm run *"},
		{"Edit(/src/**)", "Edit", "/src/**"},
		{"WebFetch(domain:example.com)", "WebFetch", "domain:example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			rule := parseGlobRule(tt.input, ActionAllow)
			if rule.Tool != tt.tool {
				t.Errorf("tool: got %q, want %q", rule.Tool, tt.tool)
			}
			if rule.Specifier != tt.spec {
				t.Errorf("specifier: got %q, want %q", rule.Specifier, tt.spec)
			}
			if rule.Action != ActionAllow {
				t.Errorf("action: got %d, want Allow", rule.Action)
			}
		})
	}
}

func TestMatchGlobRule_ToolOnly(t *testing.T) {
	rule := GlobRule{Tool: "Bash", Action: ActionAllow}
	if !matchGlobRule(rule, "Bash", "") {
		t.Error("Bash should match Bash")
	}
	if matchGlobRule(rule, "Edit", "") {
		t.Error("Bash should not match Edit")
	}
}

func TestMatchGlobRule_Wildcard(t *testing.T) {
	rule := GlobRule{Tool: "mcp__github__*", Action: ActionAllow}
	if !matchGlobRule(rule, "mcp__github__create_issue", "") {
		t.Error("wildcard should match")
	}
	if matchGlobRule(rule, "mcp__slack__send", "") {
		t.Error("wildcard should not match different prefix")
	}
}

func TestMatchGlobRule_WithSpecifier(t *testing.T) {
	rule := GlobRule{Tool: "Bash", Specifier: "npm run *", Action: ActionAllow}
	if !matchGlobRule(rule, "Bash", "npm run test") {
		t.Error("specifier should match")
	}
	if matchGlobRule(rule, "Bash", "rm -rf /") {
		t.Error("specifier should not match different command")
	}
}

func TestMatchGlobRule_PathSpecifier(t *testing.T) {
	rule := GlobRule{Tool: "Edit", Specifier: "/src/*", Action: ActionAllow}
	if !matchGlobRule(rule, "Edit", "/src/main.go") {
		t.Error("path specifier should match")
	}
	if matchGlobRule(rule, "Edit", "/etc/passwd") {
		t.Error("path specifier should not match outside dir")
	}
}

func TestExtractSpecifier(t *testing.T) {
	tests := []struct {
		tool   string
		args   map[string]any
		want   string
	}{
		{"bash", map[string]any{"command": "npm run test"}, "npm run test"},
		{"edit", map[string]any{"file_path": "/src/main.go"}, "/src/main.go"},
		{"write", map[string]any{"file_path": "/src/out.go"}, "/src/out.go"},
		{"read", map[string]any{"file_path": "/src/in.go"}, "/src/in.go"},
		{"webfetch", map[string]any{"url": "https://example.com/page"}, "domain:example.com"},
		{"grep", map[string]any{"path": "/src"}, "/src"},
		{"unknown", map[string]any{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.tool, func(t *testing.T) {
			got := ExtractSpecifier(tt.tool, tt.args)
			if got != tt.want {
				t.Errorf("ExtractSpecifier(%q) = %q, want %q", tt.tool, got, tt.want)
			}
		})
	}
}

func TestEvaluateGlobRules_DenyFirst(t *testing.T) {
	rules := []GlobRule{
		{Tool: "Bash", Specifier: "rm *", Action: ActionDeny},
		{Tool: "Bash", Action: ActionAllow},
	}

	// "rm -rf /" should be denied
	action := evaluateGlobRules(rules, "Bash", "rm -rf /")
	if action != ActionDeny {
		t.Errorf("expected deny, got %d", action)
	}

	// "echo hello" should be allowed
	action = evaluateGlobRules(rules, "Bash", "echo hello")
	if action != ActionAllow {
		t.Errorf("expected allow, got %d", action)
	}
}

func TestEvaluateGlobRules_AskSecond(t *testing.T) {
	rules := []GlobRule{
		{Tool: "Write", Action: ActionAsk},
	}

	action := evaluateGlobRules(rules, "Write", "/src/main.go")
	if action != ActionAsk {
		t.Errorf("expected ask, got %d", action)
	}
}

func TestEvaluateGlobRules_NoMatch(t *testing.T) {
	rules := []GlobRule{
		{Tool: "Bash", Action: ActionAllow},
	}

	// No rule for Edit
	action := evaluateGlobRules(rules, "Edit", "")
	if action != ActionNone {
		t.Errorf("expected none, got %d", action)
	}
}

func TestNewCheckerFromSettings(t *testing.T) {
	allow := []string{"Bash(npm run *)"}
	deny := []string{"Bash(rm *)"}
	ask := []string{"Write"}

	checker := NewCheckerFromSettings(ModeNormal, nil, allow, deny, ask)
	if checker == nil {
		t.Fatal("expected non-nil checker")
	}

	// Denied command
	err := checker.Check("Bash", map[string]any{"command": "rm -rf /"})
	if err == nil {
		t.Error("expected error for denied command")
	}

	// Allowed command
	err = checker.Check("Bash", map[string]any{"command": "npm run test"})
	if err != nil {
		t.Errorf("npm run test should be allowed: %v", err)
	}
}
