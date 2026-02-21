// ABOUTME: System prompt construction with tools, context files, skills, date/cwd
// ABOUTME: Assembles the system prompt dynamically based on session state

package prompt

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// BuildSystem constructs the system prompt for the agent.
func BuildSystem(opts SystemOpts) string {
	var b strings.Builder

	b.WriteString("You are pi-go, an AI coding assistant.\n\n")

	// Date and working directory
	b.WriteString(fmt.Sprintf("Current date: %s\n", time.Now().Format("2006-01-02")))
	b.WriteString(fmt.Sprintf("Working directory: %s\n\n", opts.CWD))

	// Mode
	if opts.PlanMode {
		b.WriteString("You are in PLAN mode. You can only read files and analyze code.\n")
		b.WriteString("You cannot modify files or execute commands.\n")
		b.WriteString("Suggest changes but do not make them.\n\n")
	}

	// Available tools
	if len(opts.ToolNames) > 0 {
		b.WriteString("Available tools: ")
		b.WriteString(strings.Join(opts.ToolNames, ", "))
		b.WriteString("\n\n")
	}

	// Skills
	for _, skill := range opts.Skills {
		b.WriteString(fmt.Sprintf("# Skill: %s\n%s\n\n", skill.Name, skill.Content))
	}

	// Context files (.pi-go/context or CLAUDE.md)
	for _, ctx := range opts.ContextFiles {
		b.WriteString(fmt.Sprintf("# Context: %s\n%s\n\n", ctx.Name, ctx.Content))
	}

	return b.String()
}

// SystemOpts configures the system prompt.
type SystemOpts struct {
	CWD          string
	PlanMode     bool
	ToolNames    []string
	Skills       []SkillRef
	ContextFiles []ContextFile
}

// SkillRef is a reference to a loaded skill.
type SkillRef struct {
	Name    string
	Content string
}

// ContextFile is a loaded context file.
type ContextFile struct {
	Name    string
	Content string
}

// LoadContextFiles reads context files from standard locations.
func LoadContextFiles(projectRoot string) []ContextFile {
	var files []ContextFile

	// Check for .pi-go/context
	paths := []struct {
		path string
		name string
	}{
		{projectRoot + "/.pi-go/CONTEXT.md", "project-context"},
		{projectRoot + "/CLAUDE.md", "claude-md"},
	}

	for _, p := range paths {
		data, err := os.ReadFile(p.path)
		if err != nil {
			continue
		}
		files = append(files, ContextFile{
			Name:    p.name,
			Content: string(data),
		})
	}

	return files
}
