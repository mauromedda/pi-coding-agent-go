// ABOUTME: SKILL.md parsing with YAML frontmatter support and validation
// ABOUTME: Loads skills from project, global, and Claude Code compat directories

package prompt

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mauromedda/pi-coding-agent-go/internal/config"
)

// Skill represents a loaded skill definition.
type Skill struct {
	Name         string   `yaml:"name"`
	Description  string   `yaml:"description"`
	AllowedTools []string `yaml:"allowed-tools"`
	Content      string   // Markdown body after frontmatter
	SourcePath   string   // File path this was loaded from
}

// LoadSkills loads skills from all resolution paths, merging by name.
// Project-local overrides user-global overrides Claude Code compat.
func LoadSkills(projectDir string) ([]Skill, error) {
	dirs := config.SkillsDirs(projectDir)
	byName := make(map[string]Skill)

	// Load in reverse order so higher-priority dirs override
	for i := len(dirs) - 1; i >= 0; i-- {
		skills, err := loadSkillsFromDir(dirs[i])
		if err != nil {
			continue // Skip inaccessible directories
		}
		for _, s := range skills {
			byName[s.Name] = s
		}
	}

	result := make([]Skill, 0, len(byName))
	for _, s := range byName {
		result = append(result, s)
	}
	return result, nil
}

func loadSkillsFromDir(dir string) ([]Skill, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading skills dir %s: %w", dir, err)
	}

	var skills []Skill
	for _, entry := range entries {
		if entry.IsDir() {
			// Check for SKILL.md inside directory
			skillPath := filepath.Join(dir, entry.Name(), "SKILL.md")
			skill, err := parseSkillFile(skillPath)
			if err != nil {
				continue
			}
			skills = append(skills, skill)
		} else if strings.HasSuffix(entry.Name(), ".md") {
			skillPath := filepath.Join(dir, entry.Name())
			skill, err := parseSkillFile(skillPath)
			if err != nil {
				continue
			}
			skills = append(skills, skill)
		}
	}
	return skills, nil
}

// skillFrontmatter is the typed structure for YAML frontmatter in skill files.
type skillFrontmatter struct {
	Name         string   `yaml:"name"`
	Description  string   `yaml:"description"`
	AllowedTools []string `yaml:"allowed-tools"`
}

func parseSkillFile(path string) (Skill, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Skill{}, fmt.Errorf("reading skill %s: %w", path, err)
	}

	content := string(data)
	skill := Skill{SourcePath: path}

	fm, body, err := config.ParseFrontmatter[skillFrontmatter](content)
	if err != nil {
		// Frontmatter parse error; use content as-is
		skill.Content = content
	} else {
		skill.Name = fm.Name
		skill.Description = fm.Description
		skill.AllowedTools = fm.AllowedTools
		skill.Content = strings.TrimSpace(body)
	}

	// Default name from parent directory or filename if not in frontmatter
	if skill.Name == "" {
		dir := filepath.Dir(path)
		base := filepath.Base(dir)
		if base == "." || base == "/" {
			base = filepath.Base(path)
			base = strings.TrimSuffix(base, filepath.Ext(base))
		}
		skill.Name = base
	}

	return skill, nil
}

// skillNameRe matches valid skill names: lowercase alphanumeric with single hyphens.
var skillNameRe = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// ValidateSkillName checks a skill name against naming rules.
// Returns a list of validation error messages (empty = valid).
func ValidateSkillName(name, parentDir string) []string {
	var errs []string

	if name == "" {
		return []string{"skill name is required"}
	}
	if len(name) > 64 {
		errs = append(errs, fmt.Sprintf("skill name %q exceeds 64 characters", name))
	}
	if !skillNameRe.MatchString(name) {
		errs = append(errs, fmt.Sprintf("skill name %q must match ^[a-z0-9]+(-[a-z0-9]+)*$", name))
	}
	if name != parentDir {
		errs = append(errs, fmt.Sprintf("skill name %q must match parent directory %q", name, parentDir))
	}
	return errs
}

// ValidateSkillDescription checks a skill description against rules.
func ValidateSkillDescription(desc string) []string {
	var errs []string
	if desc == "" {
		errs = append(errs, "skill description is required")
	}
	if len(desc) > 1024 {
		errs = append(errs, fmt.Sprintf("skill description exceeds 1024 characters (%d)", len(desc)))
	}
	return errs
}

// SkillSource identifies where a skill was loaded from.
// Returns "user", "project", or "path".
func SkillSource(sourcePath string) string {
	home, _ := os.UserHomeDir()
	if home != "" && strings.Contains(sourcePath, filepath.Join(home, ".claude", "skills")) {
		return "user"
	}
	if strings.Contains(sourcePath, filepath.Join(".pi-go", "skills")) {
		return "project"
	}
	return "path"
}

// DetectCollisions returns warnings for skills loaded from multiple sources.
func DetectCollisions(skills []Skill) []string {
	byName := make(map[string][]string) // name -> list of source paths
	for _, s := range skills {
		byName[s.Name] = append(byName[s.Name], s.SourcePath)
	}

	var warnings []string
	for name, paths := range byName {
		if len(paths) > 1 {
			warnings = append(warnings, fmt.Sprintf("skill %q loaded from multiple sources: %v", name, paths))
		}
	}
	return warnings
}
