// ABOUTME: Tests for skills validation: name rules, description limits, source detection
// ABOUTME: Covers frontmatter parsing upgrade, collision detection, edge cases

package prompt

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateSkillName(t *testing.T) {
	tests := []struct {
		name      string
		skillName string
		parentDir string
		wantErrs  int
	}{
		{"valid name", "my-skill", "my-skill", 0},
		{"too long", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", 1},
		{"uppercase", "MySkill", "MySkill", 1},
		{"double dash", "my--skill", "my--skill", 1},
		{"starts with dash", "-myskill", "-myskill", 1},
		{"ends with dash", "myskill-", "myskill-", 1},
		{"mismatch parent dir", "my-skill", "other-dir", 1},
		{"empty name", "", "x", 1},
		{"valid with numbers", "skill-123", "skill-123", 0},
		{"special chars", "my_skill!", "my_skill!", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidateSkillName(tt.skillName, tt.parentDir)
			if len(errs) != tt.wantErrs {
				t.Errorf("ValidateSkillName(%q, %q) returned %d errors; want %d: %v",
					tt.skillName, tt.parentDir, len(errs), tt.wantErrs, errs)
			}
		})
	}
}

func TestValidateSkillDescription(t *testing.T) {
	tests := []struct {
		name     string
		desc     string
		wantErrs int
	}{
		{"valid", "A useful skill that does things", 0},
		{"empty", "", 1},
		{"too long", string(make([]byte, 1025)), 1},
		{"max length", string(make([]byte, 1024)), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidateSkillDescription(tt.desc)
			if len(errs) != tt.wantErrs {
				t.Errorf("ValidateSkillDescription returned %d errors; want %d: %v",
					len(errs), tt.wantErrs, errs)
			}
		})
	}
}

func TestSkillSource(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{"user dir", filepath.Join(os.Getenv("HOME"), ".claude", "skills", "test", "SKILL.md"), "user"},
		{"project dir", filepath.Join(".pi-go", "skills", "test", "SKILL.md"), "project"},
		{"other path", "/some/other/path/SKILL.md", "path"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SkillSource(tt.path)
			if got != tt.want {
				t.Errorf("SkillSource(%q) = %q; want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestParseSkillFile_WithYAMLFrontmatter(t *testing.T) {
	// Create a temp skill file with proper YAML frontmatter
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "test-skill")
	os.MkdirAll(skillDir, 0o755)
	skillPath := filepath.Join(skillDir, "SKILL.md")

	content := `---
name: test-skill
description: A test skill for validation
allowed-tools:
  - Read
  - Write
---

# Test Skill

This is the body.
`
	os.WriteFile(skillPath, []byte(content), 0o644)

	skill, err := parseSkillFile(skillPath)
	if err != nil {
		t.Fatalf("parseSkillFile returned error: %v", err)
	}

	if skill.Name != "test-skill" {
		t.Errorf("Name = %q; want %q", skill.Name, "test-skill")
	}
	if skill.Description != "A test skill for validation" {
		t.Errorf("Description = %q; want %q", skill.Description, "A test skill for validation")
	}
	if len(skill.AllowedTools) != 2 {
		t.Errorf("AllowedTools length = %d; want 2", len(skill.AllowedTools))
	}
	if skill.Content == "" {
		t.Error("Content should not be empty")
	}
}

func TestDetectCollisions(t *testing.T) {
	skills := []Skill{
		{Name: "alpha", SourcePath: "/user/skills/alpha/SKILL.md"},
		{Name: "alpha", SourcePath: "/project/skills/alpha/SKILL.md"},
		{Name: "beta", SourcePath: "/user/skills/beta/SKILL.md"},
	}

	warnings := DetectCollisions(skills)
	if len(warnings) != 1 {
		t.Fatalf("DetectCollisions returned %d warnings; want 1", len(warnings))
	}
}
