// ABOUTME: Tests for sub-agent spawning, tool filtering, and agent definitions
// ABOUTME: Verifies isolated context, max turns, background execution, and definition loading

package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSubAgentConfig_Defaults(t *testing.T) {
	cfg := SubAgentConfig{
		Name: "test",
	}
	if cfg.MaxTurns != 0 {
		t.Errorf("MaxTurns should default to 0, got %d", cfg.MaxTurns)
	}
	if cfg.Background {
		t.Error("Background should default to false")
	}
}

func TestFilterTools_AllowList(t *testing.T) {
	all := []*AgentTool{
		{Name: "read"},
		{Name: "write"},
		{Name: "bash"},
		{Name: "grep"},
	}

	filtered := filterTools(all, []string{"read", "grep"}, nil)
	if len(filtered) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(filtered))
	}

	names := make(map[string]bool)
	for _, t := range filtered {
		names[t.Name] = true
	}
	if !names["read"] || !names["grep"] {
		t.Errorf("unexpected tools: %v", names)
	}
}

func TestFilterTools_DisallowList(t *testing.T) {
	all := []*AgentTool{
		{Name: "read"},
		{Name: "write"},
		{Name: "bash"},
		{Name: "grep"},
	}

	filtered := filterTools(all, nil, []string{"bash", "write"})
	if len(filtered) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(filtered))
	}

	names := make(map[string]bool)
	for _, t := range filtered {
		names[t.Name] = true
	}
	if !names["read"] || !names["grep"] {
		t.Errorf("unexpected tools: %v", names)
	}
}

func TestFilterTools_NilLists(t *testing.T) {
	all := []*AgentTool{
		{Name: "read"},
		{Name: "write"},
	}

	filtered := filterTools(all, nil, nil)
	if len(filtered) != 2 {
		t.Errorf("nil allow/disallow should return all tools, got %d", len(filtered))
	}
}

func TestFilterTools_EmptyResult(t *testing.T) {
	all := []*AgentTool{
		{Name: "read"},
	}

	filtered := filterTools(all, []string{"nonexistent"}, nil)
	if len(filtered) != 0 {
		t.Errorf("expected 0 tools, got %d", len(filtered))
	}
}

func TestBuiltinDefinitions(t *testing.T) {
	defs := BuiltinDefinitions()
	if len(defs) == 0 {
		t.Fatal("expected at least one builtin definition")
	}

	// Check explore agent exists
	explore, ok := defs["explore"]
	if !ok {
		t.Fatal("expected explore agent definition")
	}
	if explore.MaxTurns == 0 {
		t.Error("explore should have MaxTurns set")
	}
	if len(explore.Tools) == 0 {
		t.Error("explore should have tools specified")
	}

	// Check plan agent exists
	plan, ok := defs["plan"]
	if !ok {
		t.Fatal("expected plan agent definition")
	}
	if plan.MaxTurns == 0 {
		t.Error("plan should have MaxTurns set")
	}
}

func TestLoadDefinitions_EmptyDirs(t *testing.T) {
	project := t.TempDir()
	home := t.TempDir()

	defs, err := LoadDefinitions(project, home)
	if err != nil {
		t.Fatalf("LoadDefinitions: %v", err)
	}

	// Should have builtins
	if len(defs) == 0 {
		t.Error("expected at least builtin definitions")
	}
}

func TestLoadDefinitions_CustomAgent(t *testing.T) {
	project := t.TempDir()
	home := t.TempDir()

	agentsDir := filepath.Join(project, ".pi-go", "agents")
	if err := os.MkdirAll(agentsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	content := "---\nname: custom\ndescription: A custom agent\nmodel: fast\nmax-turns: 5\ntools: read, grep\n---\nYou are a custom agent."
	if err := os.WriteFile(filepath.Join(agentsDir, "custom.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	defs, err := LoadDefinitions(project, home)
	if err != nil {
		t.Fatalf("LoadDefinitions: %v", err)
	}

	custom, ok := defs["custom"]
	if !ok {
		t.Fatal("expected custom agent definition")
	}
	if custom.Description != "A custom agent" {
		t.Errorf("unexpected description: %q", custom.Description)
	}
	if custom.MaxTurns != 5 {
		t.Errorf("expected MaxTurns 5, got %d", custom.MaxTurns)
	}
	if custom.Model != "fast" {
		t.Errorf("expected model fast, got %q", custom.Model)
	}
	if len(custom.Tools) != 2 {
		t.Errorf("expected 2 tools, got %d", len(custom.Tools))
	}
	if custom.SystemPrompt != "You are a custom agent." {
		t.Errorf("unexpected system prompt: %q", custom.SystemPrompt)
	}
}

func TestLoadDefinitions_CustomOverridesBuiltin(t *testing.T) {
	project := t.TempDir()
	home := t.TempDir()

	agentsDir := filepath.Join(project, ".pi-go", "agents")
	if err := os.MkdirAll(agentsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	content := "---\nname: explore\ndescription: Custom explore\nmax-turns: 20\n---\nCustom explore prompt."
	if err := os.WriteFile(filepath.Join(agentsDir, "explore.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	defs, err := LoadDefinitions(project, home)
	if err != nil {
		t.Fatalf("LoadDefinitions: %v", err)
	}

	explore := defs["explore"]
	if explore.Description != "Custom explore" {
		t.Errorf("custom should override builtin description, got %q", explore.Description)
	}
	if explore.MaxTurns != 20 {
		t.Errorf("expected MaxTurns 20, got %d", explore.MaxTurns)
	}
}

func TestParseAgentFrontmatter(t *testing.T) {
	content := "---\nname: test\ndescription: Test agent\nmodel: claude-sonnet\nmax-turns: 3\ntools: read, write, grep\ndisallowed-tools: bash\n---\nBody content here."

	def := parseAgentFile(content, "test.md")
	if def.Name != "test" {
		t.Errorf("expected name test, got %q", def.Name)
	}
	if def.MaxTurns != 3 {
		t.Errorf("expected MaxTurns 3, got %d", def.MaxTurns)
	}
	if len(def.Tools) != 3 {
		t.Errorf("expected 3 tools, got %d", len(def.Tools))
	}
	if len(def.DisallowedTools) != 1 {
		t.Errorf("expected 1 disallowed tool, got %d", len(def.DisallowedTools))
	}
	if def.SystemPrompt != "Body content here." {
		t.Errorf("unexpected body: %q", def.SystemPrompt)
	}
}
