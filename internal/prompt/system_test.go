// ABOUTME: Tests for system prompt construction and output style instructions
// ABOUTME: Covers StyleInstructions mapping and BuildSystem style integration

package prompt

import (
	"os"
	"strings"
	"testing"
)

func TestStyleInstructions_Concise(t *testing.T) {
	got := StyleInstructions("concise")
	if got == "" {
		t.Fatal("StyleInstructions(\"concise\") returned empty string")
	}
	if !strings.Contains(strings.ToLower(got), "concise") {
		t.Errorf("expected text containing \"concise\", got %q", got)
	}
}

func TestStyleInstructions_Verbose(t *testing.T) {
	got := StyleInstructions("verbose")
	if got == "" {
		t.Fatal("StyleInstructions(\"verbose\") returned empty string")
	}
	if !strings.Contains(strings.ToLower(got), "detailed") {
		t.Errorf("expected text containing \"detailed\", got %q", got)
	}
}

func TestStyleInstructions_Formal(t *testing.T) {
	got := StyleInstructions("formal")
	if got == "" {
		t.Fatal("StyleInstructions(\"formal\") returned empty string")
	}
	if !strings.Contains(strings.ToLower(got), "formal") {
		t.Errorf("expected text containing \"formal\", got %q", got)
	}
}

func TestStyleInstructions_Casual(t *testing.T) {
	got := StyleInstructions("casual")
	if got == "" {
		t.Fatal("StyleInstructions(\"casual\") returned empty string")
	}
	if !strings.Contains(strings.ToLower(got), "casual") {
		t.Errorf("expected text containing \"casual\", got %q", got)
	}
}

func TestStyleInstructions_Empty(t *testing.T) {
	got := StyleInstructions("")
	if got != "" {
		t.Errorf("StyleInstructions(\"\") = %q; want empty string", got)
	}
}

func TestStyleInstructions_Unknown(t *testing.T) {
	got := StyleInstructions("unknown-style")
	if got != "" {
		t.Errorf("StyleInstructions(\"unknown-style\") = %q; want empty string", got)
	}
}

func TestBuildSystem_WithStyle(t *testing.T) {
	opts := SystemOpts{
		CWD:   "/tmp/test",
		Style: "concise",
	}
	result := BuildSystem(opts)
	if !strings.Contains(strings.ToLower(result), "concise") {
		t.Errorf("BuildSystem with style \"concise\" should contain style text, got:\n%s", result)
	}
}

func TestBuildSystem_WithoutStyle(t *testing.T) {
	opts := SystemOpts{
		CWD: "/tmp/test",
	}
	result := BuildSystem(opts)

	// Should not contain any style instruction keywords
	styleKeywords := []string{
		"Be extremely concise",
		"detailed, thorough",
		"formal, professional",
		"casual and conversational",
	}
	for _, kw := range styleKeywords {
		if strings.Contains(result, kw) {
			t.Errorf("BuildSystem without style should not contain %q", kw)
		}
	}
}

func TestBuildSystem_PersonalityPrompt(t *testing.T) {
	opts := SystemOpts{
		CWD:               "/tmp/test",
		PersonalityPrompt: "Be thorough and verify your work.",
	}
	result := BuildSystem(opts)

	if !strings.Contains(result, "# Personality") {
		t.Error("expected personality section header")
	}
	if !strings.Contains(result, "Be thorough and verify your work.") {
		t.Error("expected personality prompt text in output")
	}
}

func TestBuildSystem_PersonalityPrompt_Empty(t *testing.T) {
	opts := SystemOpts{
		CWD: "/tmp/test",
	}
	result := BuildSystem(opts)

	if strings.Contains(result, "# Personality") {
		t.Error("empty personality prompt should not produce personality section")
	}
}

func TestBuildSystem_PersonalityAfterSkills(t *testing.T) {
	opts := SystemOpts{
		CWD:               "/tmp/test",
		Skills:            []SkillRef{{Name: "test-skill", Content: "skill content"}},
		PersonalityPrompt: "personality text",
		ContextFiles:      []ContextFile{{Name: "ctx", Content: "context content"}},
	}
	result := BuildSystem(opts)

	skillIdx := strings.Index(result, "# Skill: test-skill")
	personalityIdx := strings.Index(result, "# Personality")
	contextIdx := strings.Index(result, "# Context: ctx")

	if skillIdx < 0 || personalityIdx < 0 || contextIdx < 0 {
		t.Fatalf("missing sections: skill=%d, personality=%d, context=%d", skillIdx, personalityIdx, contextIdx)
	}
	if personalityIdx < skillIdx {
		t.Error("personality section should appear after skills")
	}
	if personalityIdx > contextIdx {
		t.Error("personality section should appear before context files")
	}
}

func TestBuildSystem_PromptVersionFallback(t *testing.T) {
	// When PromptVersion is set but no prompts directory exists,
	// it should fall back to the hardcoded header.
	opts := SystemOpts{
		CWD:           "/tmp/test",
		PromptVersion: "v99.99.99", // Non-existent version
	}
	result := BuildSystem(opts)

	// Fallback should produce the hardcoded header
	if !strings.Contains(result, "pi-go") {
		t.Error("expected fallback header to contain 'pi-go'")
	}
}

func TestBuildSystem_NoPromptVersion(t *testing.T) {
	// Empty PromptVersion should use the hardcoded header
	opts := SystemOpts{
		CWD: "/tmp/test",
	}
	result := BuildSystem(opts)

	if !strings.Contains(result, "You are pi-go") {
		t.Error("expected hardcoded header when PromptVersion is empty")
	}
}

func TestBuildSystem_Lean(t *testing.T) {
	opts := SystemOpts{
		CWD:               "/tmp/test",
		Lean:              true,
		ToolNames:         []string{"read", "write", "bash"},
		Skills:            []SkillRef{{Name: "test-skill", Content: "skill content"}},
		PersonalityPrompt: "Be thorough.",
		MemorySection:     "# Memory\nSome memory.",
		ContextFiles:      []ContextFile{{Name: "ctx", Content: "context content"}},
		Style:             "concise",
		PromptVersion:     "v1.0.0",
		PlanMode:          true,
	}
	result := BuildSystem(opts)

	// Lean MUST include: hardcoded header, tools
	if !strings.Contains(result, "You are pi-go") {
		t.Error("lean prompt must include hardcoded header")
	}
	if !strings.Contains(result, "read, write, bash") {
		t.Error("lean prompt must include tool list")
	}

	// Lean MUST NOT include: skills, personality, memory, context, style, plan mode
	excluded := []struct {
		label string
		text  string
	}{
		{"skills", "# Skill:"},
		{"personality", "# Personality"},
		{"memory", "# Memory"},
		{"context files", "# Context:"},
		{"style", "concise"},
		{"plan mode", "PLAN mode"},
	}
	for _, ex := range excluded {
		if strings.Contains(result, ex.text) {
			t.Errorf("lean prompt must not include %s section (found %q)", ex.label, ex.text)
		}
	}
}

func TestBuildSystem_LeanFalse_Unchanged(t *testing.T) {
	// Lean=false should behave exactly as before (no regression)
	opts := SystemOpts{
		CWD:               "/tmp/test",
		Lean:              false,
		ToolNames:         []string{"read"},
		PersonalityPrompt: "Be thorough.",
		MemorySection:     "# Memory\nSome memory.",
		Style:             "concise",
	}
	result := BuildSystem(opts)

	if !strings.Contains(result, "# Personality") {
		t.Error("non-lean prompt should include personality section")
	}
	if !strings.Contains(result, "# Memory") {
		t.Error("non-lean prompt should include memory section")
	}
}

func TestLoadContextFiles_NoDuplicateCLAUDEMD(t *testing.T) {
	dir := t.TempDir()

	// Create both CLAUDE.md and .pi-go/CONTEXT.md
	if err := os.WriteFile(dir+"/CLAUDE.md", []byte("claude content"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir+"/.pi-go", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dir+"/.pi-go/CONTEXT.md", []byte("context content"), 0o644); err != nil {
		t.Fatal(err)
	}

	files := LoadContextFiles(dir)

	// CLAUDE.md should NOT be in context files (it's loaded via memory.Load)
	for _, f := range files {
		if f.Name == "claude-md" {
			t.Error("LoadContextFiles should not include claude-md; it is already loaded by memory.Load")
		}
	}

	// .pi-go/CONTEXT.md should still be loaded
	found := false
	for _, f := range files {
		if f.Name == "project-context" {
			found = true
		}
	}
	if !found {
		t.Error("LoadContextFiles should still include project-context")
	}
}

func TestBuildSystem_LoaderSingleton(t *testing.T) {
	// getDefaultLoader should return the same instance on repeated calls.
	l1 := getDefaultLoader()
	l2 := getDefaultLoader()
	if l1 != l2 {
		t.Error("getDefaultLoader() returned different instances; expected singleton")
	}
	if l1.Cache == nil {
		t.Error("default loader should have a Cache attached")
	}
}

func TestModeForVersion(t *testing.T) {
	tests := []struct {
		name string
		opts SystemOpts
		want string
	}{
		{"plan mode", SystemOpts{PlanMode: true}, "plan"},
		{"execute mode", SystemOpts{PlanMode: false}, "execute"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := modeForVersion(tt.opts); got != tt.want {
				t.Errorf("modeForVersion() = %q; want %q", got, tt.want)
			}
		})
	}
}
