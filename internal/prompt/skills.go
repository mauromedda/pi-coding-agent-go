// ABOUTME: SKILL.md parsing with YAML frontmatter support
// ABOUTME: Loads skills from project, global, and Claude Code compat directories

package prompt

import (
	"fmt"
	"os"
	"path/filepath"
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

func parseSkillFile(path string) (Skill, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Skill{}, fmt.Errorf("reading skill %s: %w", path, err)
	}

	content := string(data)
	skill := Skill{SourcePath: path}

	// Parse YAML frontmatter (--- delimited)
	if strings.HasPrefix(content, "---\n") {
		endIdx := strings.Index(content[4:], "\n---")
		if endIdx >= 0 {
			frontmatter := content[4 : 4+endIdx]
			skill.Content = strings.TrimSpace(content[4+endIdx+4:])
			parseFrontmatter(frontmatter, &skill)
		} else {
			skill.Content = content
		}
	} else {
		skill.Content = content
	}

	// Default name from filename if not in frontmatter
	if skill.Name == "" {
		base := filepath.Base(path)
		skill.Name = strings.TrimSuffix(base, filepath.Ext(base))
	}

	return skill, nil
}

// parseFrontmatter extracts key-value pairs from YAML frontmatter.
// Simplified parser: handles name, description, allowed-tools.
func parseFrontmatter(fm string, skill *Skill) {
	for _, line := range strings.Split(fm, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		switch key {
		case "name":
			skill.Name = value
		case "description":
			skill.Description = value
		case "allowed-tools":
			tools := strings.Split(value, ",")
			for i, t := range tools {
				tools[i] = strings.TrimSpace(t)
			}
			skill.AllowedTools = tools
		}
	}
}
