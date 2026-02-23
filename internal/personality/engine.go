// ABOUTME: Personality engine orchestrating profile loading and check composition
// ABOUTME: Composes prompt injections from active profile and check results

package personality

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/mauromedda/pi-coding-agent-go/internal/personality/checks"
)

// Engine manages personality profiles and runs checks.
// It is safe for concurrent use.
type Engine struct {
	mu       sync.RWMutex
	profiles map[string]*Profile
	active   *Profile
}

// NewEngine creates an engine with built-in profiles.
// If profilesDir is non-empty, loads additional profiles from disk.
func NewEngine(profilesDir string) (*Engine, error) {
	e := &Engine{
		profiles: builtinProfiles(),
	}

	if profilesDir != "" {
		extra, err := LoadProfiles(profilesDir)
		if err != nil {
			return nil, fmt.Errorf("load profiles: %w", err)
		}
		for name, p := range extra {
			e.profiles[name] = p
		}
	}

	e.active = e.profiles["base"]
	return e, nil
}

// SetProfile activates a profile by name.
func (e *Engine) SetProfile(name string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	p, ok := e.profiles[name]
	if !ok {
		return fmt.Errorf("unknown profile %q", name)
	}
	e.active = p
	return nil
}

// ActiveProfile returns the currently active profile.
func (e *Engine) ActiveProfile() *Profile {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.active
}

// ProfileNames returns all available profile names sorted alphabetically.
func (e *Engine) ProfileNames() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	names := make([]string, 0, len(e.profiles))
	for name := range e.profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ComposePrompt generates the prompt injection for the current personality.
// Runs all enabled checks and combines their instructions.
// Checks not explicitly configured in the active profile default to "standard" level.
func (e *Engine) ComposePrompt(ctx checks.CheckContext) string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if e.active == nil {
		return ""
	}

	var sections []string

	// Header with personality context
	sections = append(sections, fmt.Sprintf("## Personality: %s", e.active.Name))
	if e.active.Description != "" {
		sections = append(sections, e.active.Description)
	}

	// Trait instructions
	sections = append(sections, e.composeTraitInstructions())

	// Run checks and collect results
	for _, checkName := range checks.AllCheckNames() {
		level, ok := e.active.Checks[checkName]
		if !ok {
			level = "standard" // default check level
		}

		c := checks.NewCheck(checkName, level)
		if c == nil {
			continue
		}

		result := c.Analyze(ctx)
		if len(result.Instructions) > 0 {
			section := fmt.Sprintf("### %s (%s)", result.Name, result.Level)
			for _, instr := range result.Instructions {
				section += "\n- " + instr
			}
			if len(result.Warnings) > 0 {
				section += "\n**Warnings:**"
				for _, w := range result.Warnings {
					section += "\n- ⚠ " + w
				}
			}
			sections = append(sections, section)
		}
	}

	return strings.Join(sections, "\n\n")
}

func (e *Engine) composeTraitInstructions() string {
	t := e.active.Traits
	var lines []string

	switch t.Verbosity {
	case "terse":
		lines = append(lines, "Be concise; omit unnecessary explanation.")
	case "verbose":
		lines = append(lines, "Provide detailed explanations and reasoning.")
	default:
		lines = append(lines, "Balance brevity with clarity.")
	}

	switch t.RiskTolerance {
	case "cautious":
		lines = append(lines, "Prefer safe, well-tested approaches over novel ones.")
	case "aggressive":
		lines = append(lines, "Favor speed and pragmatism; accept calculated risks.")
	default:
		lines = append(lines, "Balance safety with pragmatism.")
	}

	switch t.Explanation {
	case "minimal":
		lines = append(lines, "Explain only when asked or when the choice is non-obvious.")
	case "detailed":
		lines = append(lines, "Explain the reasoning behind every significant decision.")
	default:
		lines = append(lines, "Explain important decisions; skip the obvious.")
	}

	switch t.AutoPlan {
	case "never":
		lines = append(lines, "Do not generate plans unless explicitly asked.")
	case "auto":
		lines = append(lines, "Automatically generate implementation plans for complex tasks.")
	default:
		lines = append(lines, "Suggest plans for complex tasks but wait for approval.")
	}

	return "### Behavioral Traits\n- " + strings.Join(lines, "\n- ")
}

func builtinProfiles() map[string]*Profile {
	return map[string]*Profile{
		"base": {
			Name:        "base",
			Description: "Balanced defaults for general-purpose development.",
			Traits:      DefaultTraitSet(),
			Checks: map[string]string{
				"security":     "standard",
				"performance":  "standard",
				"quality":      "standard",
				"architecture": "standard",
				"factual":      "standard",
			},
		},
		"security-focused": {
			Name:        "security-focused",
			Description: "Paranoid security posture with standard checks elsewhere.",
			Traits: TraitSet{
				Verbosity:     "balanced",
				RiskTolerance: "cautious",
				Explanation:   "detailed",
				AutoPlan:      "suggest",
			},
			Checks: map[string]string{
				"security":     "paranoid",
				"performance":  "standard",
				"quality":      "standard",
				"architecture": "standard",
				"factual":      "standard",
			},
		},
		"speed-focused": {
			Name:        "speed-focused",
			Description: "Minimal checks for rapid iteration.",
			Traits: TraitSet{
				Verbosity:     "terse",
				RiskTolerance: "aggressive",
				Explanation:   "minimal",
				AutoPlan:      "never",
			},
			Checks: map[string]string{
				"security":     "minimal",
				"performance":  "minimal",
				"quality":      "minimal",
				"architecture": "minimal",
				"factual":      "minimal",
			},
		},
		"mentor": {
			Name:        "mentor",
			Description: "Verbose explanations with strict quality expectations.",
			Traits: TraitSet{
				Verbosity:     "verbose",
				RiskTolerance: "cautious",
				Explanation:   "detailed",
				AutoPlan:      "suggest",
			},
			Checks: map[string]string{
				"security":     "standard",
				"performance":  "standard",
				"quality":      "strict",
				"architecture": "standard",
				"factual":      "strict",
			},
		},
		"architect": {
			Name:        "architect",
			Description: "Strict architectural enforcement with automatic planning.",
			Traits: TraitSet{
				Verbosity:     "balanced",
				RiskTolerance: "cautious",
				Explanation:   "standard",
				AutoPlan:      "auto",
			},
			Checks: map[string]string{
				"security":     "standard",
				"performance":  "standard",
				"quality":      "standard",
				"architecture": "strict",
				"factual":      "standard",
			},
		},
	}
}
