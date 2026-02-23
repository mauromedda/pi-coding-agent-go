// ABOUTME: System prompt construction with tools, context files, skills, date/cwd
// ABOUTME: Assembles the system prompt dynamically based on session state

package prompt

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mauromedda/pi-coding-agent-go/internal/prompts"
)

// BuildSystem constructs the system prompt for the agent.
func BuildSystem(opts SystemOpts) string {
	var b strings.Builder

	// Lean mode: hardcoded header + tool list only
	if opts.Lean {
		writeHardcodedHeader(&b, opts.CWD)
		if len(opts.ToolNames) > 0 {
			b.WriteString("Available tools: ")
			b.WriteString(strings.Join(opts.ToolNames, ", "))
			b.WriteString("\n\n")
		}
		return b.String()
	}

	// Base prompt: versioned loader or hardcoded fallback
	if opts.PromptVersion != "" {
		loader := prompts.NewLoader("prompts", "prompts/overrides")
		vars := map[string]string{
			"DATE":      time.Now().Format("2006-01-02"),
			"CWD":       opts.CWD,
			"TOOL_LIST": strings.Join(opts.ToolNames, ", "),
			"MODE":      modeForVersion(opts),
		}
		if composed, err := loader.Compose(opts.PromptVersion, vars); err == nil {
			b.WriteString(composed)
			b.WriteString("\n\n")
		} else {
			// Fallback to hardcoded on error
			writeHardcodedHeader(&b, opts.CWD)
		}
	} else {
		writeHardcodedHeader(&b, opts.CWD)
	}

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

	// Personality prompt (after skills, before context files)
	if opts.PersonalityPrompt != "" {
		b.WriteString("# Personality\n")
		b.WriteString(opts.PersonalityPrompt)
		b.WriteString("\n\n")
	}

	// Memory entries
	if opts.MemorySection != "" {
		b.WriteString(opts.MemorySection)
	}

	// Context files (.pi-go/context or CLAUDE.md)
	for _, ctx := range opts.ContextFiles {
		b.WriteString(fmt.Sprintf("# Context: %s\n%s\n\n", ctx.Name, ctx.Content))
	}

	// Output style instructions
	if si := StyleInstructions(opts.Style); si != "" {
		b.WriteString(si)
	}

	return b.String()
}

// writeHardcodedHeader writes the default header when no versioned prompt is active.
func writeHardcodedHeader(b *strings.Builder, cwd string) {
	b.WriteString("You are pi-go, an elité AI coding assistant.\n\n")
	b.WriteString(fmt.Sprintf("Current date: %s\n", time.Now().Format("2006-01-02")))
	b.WriteString(fmt.Sprintf("Working directory: %s\n\n", cwd))
}

// modeForVersion maps SystemOpts to a mode string for prompt variable substitution.
func modeForVersion(opts SystemOpts) string {
	if opts.PlanMode {
		return "plan"
	}
	return "execute"
}

// SystemOpts configures the system prompt.
type SystemOpts struct {
	CWD           string
	PlanMode      bool
	Lean          bool // minimal prompt: header + tools only
	ToolNames     []string
	Skills        []SkillRef
	ContextFiles  []ContextFile
	MemorySection string
	Style         string

	// PromptVersion delegates base prompt to prompts.Loader when set.
	// Empty string preserves the hardcoded default header.
	PromptVersion string

	// PersonalityPrompt is an injected personality prompt fragment.
	PersonalityPrompt string
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

// StyleInstructions returns style-specific instructions to append to the system prompt.
// Returns an empty string for unrecognised or empty style values.
func StyleInstructions(style string) string {
	switch style {
	case "concise":
		return "\n\nIMPORTANT: Be extremely concise. Use short sentences. Omit unnecessary words. Prefer bullet points over paragraphs."
	case "verbose":
		return "\n\nProvide detailed, thorough explanations. Include context, examples, and edge cases. Be comprehensive in your responses."
	case "formal":
		return "\n\nUse formal, professional language. Avoid contractions, slang, and casual expressions. Structure responses clearly with proper grammar."
	case "casual":
		return "\n\nBe casual and conversational. Use contractions, simple language, and a friendly tone. Feel free to use informal expressions."
	default:
		return ""
	}
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
