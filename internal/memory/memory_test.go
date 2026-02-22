// ABOUTME: Tests for memory hierarchy loading, import expansion, and prompt formatting
// ABOUTME: Table-driven tests covering 5-level resolution, cycles, depth limits, globs

package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLevelOrdering(t *testing.T) {
	if ProjectRules >= ClaudeCompat {
		t.Error("ProjectRules must be lower than ClaudeCompat")
	}
	if ClaudeCompat >= ClaudeRules {
		t.Error("ClaudeCompat must be lower than ClaudeRules")
	}
	if ClaudeRules >= UserClaudeCompat {
		t.Error("ClaudeRules must be lower than UserClaudeCompat")
	}
	if UserClaudeCompat >= AutoMemory {
		t.Error("UserClaudeCompat must be lower than AutoMemory")
	}
}

func TestLoad_EmptyDirs(t *testing.T) {
	project := t.TempDir()
	home := t.TempDir()

	entries, err := Load(project, home)
	if err != nil {
		t.Fatalf("Load empty dirs: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestLoad_PIGOMDIgnored(t *testing.T) {
	project := t.TempDir()
	home := t.TempDir()

	// PIGOMD.md files should be ignored entirely
	writeFile(t, filepath.Join(project, "PIGOMD.md"), "should be ignored")
	mkdirAll(t, filepath.Join(project, ".pi-go"))
	writeFile(t, filepath.Join(project, ".pi-go", "PIGOMD.md"), "also ignored")
	mkdirAll(t, filepath.Join(home, ".pi-go"))
	writeFile(t, filepath.Join(home, ".pi-go", "PIGOMD.md"), "user ignored")
	writeFile(t, filepath.Join(project, "PIGOMD.local.md"), "local ignored")

	entries, err := Load(project, home)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	for _, e := range entries {
		if strings.Contains(e.Content, "ignored") {
			t.Errorf("PIGOMD file should not be loaded, got entry from %s", e.Source)
		}
	}
}

func TestLoad_CLAUDEMDCompat(t *testing.T) {
	project := t.TempDir()
	home := t.TempDir()

	writeFile(t, filepath.Join(project, "CLAUDE.md"), "claude project")
	mkdirAll(t, filepath.Join(home, ".claude"))
	writeFile(t, filepath.Join(home, ".claude", "CLAUDE.md"), "claude user")

	entries, err := Load(project, home)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	projectEntry := findLevel(entries, ClaudeCompat)
	if projectEntry == nil {
		t.Fatal("no ClaudeCompat entry")
	}
	if projectEntry.Content != "claude project" {
		t.Errorf("project CLAUDE.md mismatch: got %q", projectEntry.Content)
	}

	userEntry := findLevel(entries, UserClaudeCompat)
	if userEntry == nil {
		t.Fatal("no UserClaudeCompat entry")
	}
	if userEntry.Content != "claude user" {
		t.Errorf("user CLAUDE.md mismatch: got %q", userEntry.Content)
	}
}

func TestLoad_FullHierarchy(t *testing.T) {
	project := t.TempDir()
	home := t.TempDir()

	// Level 0: Project rules
	rulesDir := filepath.Join(project, ".pi-go", "rules")
	mkdirAll(t, rulesDir)
	writeFile(t, filepath.Join(rulesDir, "rule1.md"), "project rule 1")

	// Level 1: Claude compat (project)
	writeFile(t, filepath.Join(project, "CLAUDE.md"), "claude project")

	// Level 2: Claude rules
	claudeRulesDir := filepath.Join(project, ".claude", "rules")
	mkdirAll(t, claudeRulesDir)
	writeFile(t, filepath.Join(claudeRulesDir, "r1.md"), "claude rule 1")

	// Level 3: User Claude compat
	mkdirAll(t, filepath.Join(home, ".claude"))
	writeFile(t, filepath.Join(home, ".claude", "CLAUDE.md"), "user claude")

	entries, err := Load(project, home)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// Should have 4 entries (no auto-memory yet)
	if len(entries) < 4 {
		t.Errorf("expected at least 4 entries, got %d", len(entries))
		for _, e := range entries {
			t.Logf("  level=%d source=%s", e.Level, e.Source)
		}
	}

	// Verify ordering: entries should be sorted by level
	for i := 1; i < len(entries); i++ {
		if entries[i].Level < entries[i-1].Level {
			t.Errorf("entries not sorted: [%d].Level=%d < [%d].Level=%d",
				i, entries[i].Level, i-1, entries[i-1].Level)
		}
	}
}

func TestExpandImports_Simple(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "included.md"), "included content")

	input := "before\n@included.md\nafter"
	got, err := expandImports(input, dir, nil, 0)
	if err != nil {
		t.Fatalf("expandImports: %v", err)
	}

	if !strings.Contains(got, "included content") {
		t.Errorf("expected expanded content, got %q", got)
	}
	if !strings.Contains(got, "before") || !strings.Contains(got, "after") {
		t.Errorf("expected surrounding content preserved, got %q", got)
	}
}

func TestExpandImports_Nested(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.md"), "content-a\n@b.md")
	writeFile(t, filepath.Join(dir, "b.md"), "content-b")

	input := "@a.md"
	got, err := expandImports(input, dir, nil, 0)
	if err != nil {
		t.Fatalf("expandImports: %v", err)
	}

	if !strings.Contains(got, "content-a") || !strings.Contains(got, "content-b") {
		t.Errorf("expected nested expansion, got %q", got)
	}
}

func TestExpandImports_CycleDetection(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.md"), "@b.md")
	writeFile(t, filepath.Join(dir, "b.md"), "@a.md")

	_, err := expandImports("@a.md", dir, nil, 0)
	if err == nil {
		t.Error("expected cycle detection error")
	}
	if !strings.Contains(err.Error(), "cycle") {
		t.Errorf("expected cycle error, got %v", err)
	}
}

func TestExpandImports_MaxDepth(t *testing.T) {
	dir := t.TempDir()

	// Create chain: d0.md -> d1.md -> ... -> d6.md (depth > 5)
	for i := range 7 {
		var content string
		if i < 6 {
			content = strings.ReplaceAll("@dNEXT.md", "NEXT", strings.Repeat("x", i+1))
			// Simpler: just use numbered files
		}
		_ = content
	}

	// Simpler: create a chain that's too deep
	writeFile(t, filepath.Join(dir, "d0.md"), "@d1.md")
	writeFile(t, filepath.Join(dir, "d1.md"), "@d2.md")
	writeFile(t, filepath.Join(dir, "d2.md"), "@d3.md")
	writeFile(t, filepath.Join(dir, "d3.md"), "@d4.md")
	writeFile(t, filepath.Join(dir, "d4.md"), "@d5.md")
	writeFile(t, filepath.Join(dir, "d5.md"), "@d6.md")
	writeFile(t, filepath.Join(dir, "d6.md"), "end")

	_, err := expandImports("@d0.md", dir, nil, 0)
	if err == nil {
		t.Error("expected max depth error")
	}
	if !strings.Contains(err.Error(), "depth") {
		t.Errorf("expected depth error, got %v", err)
	}
}

func TestExpandImports_MissingFile(t *testing.T) {
	dir := t.TempDir()

	// Missing files should be skipped with a comment, not fail
	input := "before\n@missing.md\nafter"
	got, err := expandImports(input, dir, nil, 0)
	if err != nil {
		t.Fatalf("expandImports should not fail on missing file: %v", err)
	}
	if !strings.Contains(got, "before") || !strings.Contains(got, "after") {
		t.Errorf("surrounding content should be preserved, got %q", got)
	}
}

func TestExpandImports_NoImports(t *testing.T) {
	dir := t.TempDir()

	input := "plain content\nno imports here"
	got, err := expandImports(input, dir, nil, 0)
	if err != nil {
		t.Fatalf("expandImports: %v", err)
	}
	if got != input {
		t.Errorf("content should be unchanged, got %q", got)
	}
}

func TestFormatForPrompt_Basic(t *testing.T) {
	entries := []Entry{
		{Source: "rule.md", Content: "project rule", Level: ProjectRules},
		{Source: "CLAUDE.md", Content: "claude compat", Level: ClaudeCompat},
	}

	result := FormatForPrompt(entries, nil)
	if !strings.Contains(result, "project rule") {
		t.Error("expected project rule in output")
	}
	if !strings.Contains(result, "claude compat") {
		t.Error("expected claude compat in output")
	}
}

func TestFormatForPrompt_PathFiltering(t *testing.T) {
	entries := []Entry{
		{Source: "general.md", Content: "always included", Level: ProjectRules},
		{Source: "go-only.md", Content: "go specific", Level: ProjectRules, Paths: []string{"*.go"}},
		{Source: "js-only.md", Content: "js specific", Level: ProjectRules, Paths: []string{"*.js"}},
	}

	result := FormatForPrompt(entries, []string{"main.go"})
	if !strings.Contains(result, "always included") {
		t.Error("expected general entry always included")
	}
	if !strings.Contains(result, "go specific") {
		t.Error("expected go-specific entry included for .go file")
	}
	if strings.Contains(result, "js specific") {
		t.Error("js-specific entry should be excluded for .go file")
	}
}

func TestFormatForPrompt_Empty(t *testing.T) {
	result := FormatForPrompt(nil, nil)
	if result != "" {
		t.Errorf("expected empty string for nil entries, got %q", result)
	}
}

func TestParseFrontmatter_Paths(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{
			name:    "bracket syntax",
			content: "---\npaths: [\"*.go\", \"*.mod\"]\n---\nBody",
			want:    []string{"*.go", "*.mod"},
		},
		{
			name:    "comma separated",
			content: "---\npaths: *.go, *.mod\n---\nBody",
			want:    []string{"*.go", "*.mod"},
		},
		{
			name:    "single value",
			content: "---\npaths: *.go\n---\nBody",
			want:    []string{"*.go"},
		},
		{
			name:    "no frontmatter",
			content: "Just body content",
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, paths := parseFrontmatter(tt.content)
			if len(paths) != len(tt.want) {
				t.Errorf("paths length: got %d, want %d", len(paths), len(tt.want))
				return
			}
			for i, p := range paths {
				if p != tt.want[i] {
					t.Errorf("paths[%d]: got %q, want %q", i, p, tt.want[i])
				}
			}
			if len(tt.want) > 0 && !strings.Contains(tt.content, body) {
				// body should not contain frontmatter
				if strings.Contains(body, "paths:") {
					t.Error("body should not contain frontmatter")
				}
			}
		})
	}
}

func TestLoadRules(t *testing.T) {
	project := t.TempDir()

	rulesDir := filepath.Join(project, ".pi-go", "rules")
	mkdirAll(t, rulesDir)
	writeFile(t, filepath.Join(rulesDir, "coding.md"), "# Coding rules\nUse TDD")
	writeFile(t, filepath.Join(rulesDir, "style.md"), "---\npaths: [\"*.go\"]\n---\n# Style\nGo style")

	entries, err := loadRulesDir(rulesDir, ProjectRules)
	if err != nil {
		t.Fatalf("loadRulesDir: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(entries))
	}

	// Find the one with paths
	var styled *Entry
	for i := range entries {
		if len(entries[i].Paths) > 0 {
			styled = &entries[i]
		}
	}
	if styled == nil {
		t.Fatal("expected one entry with paths")
	}
	if styled.Paths[0] != "*.go" {
		t.Errorf("expected *.go path, got %q", styled.Paths[0])
	}
}

func TestAutoMemoryDir(t *testing.T) {
	home := t.TempDir()
	project := "/some/project/path"

	dir := AutoMemoryDir(project, home)
	if dir == "" {
		t.Fatal("expected non-empty auto memory dir")
	}
	if !strings.HasPrefix(dir, home) {
		t.Errorf("auto memory dir should be under home, got %q", dir)
	}

	// Same project should yield same dir
	dir2 := AutoMemoryDir(project, home)
	if dir != dir2 {
		t.Errorf("same project should yield same dir: %q != %q", dir, dir2)
	}

	// Different project should yield different dir
	dir3 := AutoMemoryDir("/other/project", home)
	if dir == dir3 {
		t.Error("different projects should yield different dirs")
	}
}

func TestMatchPath(t *testing.T) {
	tests := []struct {
		pattern string
		path    string
		want    bool
	}{
		{"*.go", "main.go", true},
		{"*.go", "main.js", false},
		{"src/*.go", "src/main.go", true},
		{"src/*.go", "pkg/main.go", false},
		{"*", "anything", true},
		{"*.md", "README.md", true},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.path, func(t *testing.T) {
			got := matchPath([]string{tt.pattern}, tt.path)
			if got != tt.want {
				t.Errorf("matchPath(%q, %q) = %v, want %v", tt.pattern, tt.path, got, tt.want)
			}
		})
	}
}

// Helpers

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writeFile %s: %v", path, err)
	}
}

func mkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdirAll %s: %v", path, err)
	}
}

func findLevel(entries []Entry, level Level) *Entry {
	for i := range entries {
		if entries[i].Level == level {
			return &entries[i]
		}
	}
	return nil
}
