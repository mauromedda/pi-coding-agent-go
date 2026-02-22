// ABOUTME: Agent definition registry with builtins and custom agent loading
// ABOUTME: Loads from .pi-go/agents/, ~/.pi-go/agents/, .claude/agents/ directories

package agent

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Definition describes a reusable agent configuration.
type Definition struct {
	Name            string
	Description     string
	Model           string
	SystemPrompt    string
	Tools           []string
	DisallowedTools []string
	AllowedTools    []string
	MaxTurns        int
}

// ResolveAgentModel maps shorthand names to full model IDs.
func ResolveAgentModel(shorthand string) string {
	switch shorthand {
	case "fast":
		return "claude-haiku-4-5-20251001"
	case "default", "":
		return "claude-sonnet-4-6"
	case "powerful":
		return "claude-opus-4-6"
	default:
		return shorthand // assume it's already a full model ID
	}
}

// BuiltinDefinitions returns the built-in agent definitions.
func BuiltinDefinitions() map[string]Definition {
	return map[string]Definition{
		"explore": {
			Name:        "explore",
			Description: "Fast agent for exploring codebases: find files, search code, read files.",
			Model:       "fast",
			Tools:       []string{"read", "grep", "find", "ls"},
			MaxTurns:    10,
			SystemPrompt: "You are an exploration agent. Search the codebase to answer questions. " +
				"Use grep and find to locate relevant files, then read them. " +
				"Be thorough but efficient. Report your findings clearly.",
		},
		"plan": {
			Name:        "plan",
			Description: "Software architect agent for designing implementation plans.",
			Model:       "default",
			Tools:       []string{"read", "grep", "find", "ls"},
			MaxTurns:    15,
			SystemPrompt: "You are a planning agent. Analyze the codebase and design implementation plans. " +
				"Read existing code to understand patterns and architecture. " +
				"Produce step-by-step plans with file locations and trade-offs.",
		},
		"bash_agent": {
			Name:        "bash_agent",
			Description: "Command execution specialist for running bash commands.",
			Model:       "fast",
			Tools:       []string{"bash", "read", "ls"},
			MaxTurns:    5,
			SystemPrompt: "You are a command execution agent. Run commands as requested. " +
				"Report results clearly. Be cautious with destructive operations.",
		},
	}
}

// LoadDefinitions loads agent definitions from all sources, merging with builtins.
// Custom definitions override builtins with the same name.
func LoadDefinitions(projectDir, homeDir string) (map[string]Definition, error) {
	defs := BuiltinDefinitions()

	dirs := []string{
		filepath.Join(homeDir, ".pi-go", "agents"),
		filepath.Join(homeDir, ".claude", "agents"),
		filepath.Join(projectDir, ".pi-go", "agents"),
		filepath.Join(projectDir, ".claude", "agents"),
	}

	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}

			data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
			if err != nil {
				continue
			}

			def := parseAgentFile(string(data), entry.Name())
			if def.Name != "" {
				defs[def.Name] = def
			}
		}
	}

	return defs, nil
}

// parseAgentFile parses a markdown agent definition with YAML frontmatter.
func parseAgentFile(content, filename string) Definition {
	def := Definition{}

	// Default name from filename
	def.Name = strings.TrimSuffix(filename, filepath.Ext(filename))

	if !strings.HasPrefix(content, "---\n") {
		def.SystemPrompt = content
		return def
	}

	endIdx := strings.Index(content[4:], "\n---")
	if endIdx < 0 {
		def.SystemPrompt = content
		return def
	}

	fm := content[4 : 4+endIdx]
	def.SystemPrompt = strings.TrimSpace(content[4+endIdx+4:])

	for line := range strings.SplitSeq(fm, "\n") {
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
			def.Name = value
		case "description":
			def.Description = value
		case "model":
			def.Model = value
		case "max-turns":
			if n, err := strconv.Atoi(value); err == nil {
				def.MaxTurns = n
			}
		case "tools":
			def.Tools = splitTrimCSV(value)
		case "disallowed-tools":
			def.DisallowedTools = splitTrimCSV(value)
		case "allowed-tools":
			def.AllowedTools = splitTrimCSV(value)
		}
	}

	return def
}

// splitTrimCSV splits a comma-separated string and trims whitespace.
func splitTrimCSV(s string) []string {
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
