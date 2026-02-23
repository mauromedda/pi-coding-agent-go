// ABOUTME: Factual accuracy check for API versions, deprecation warnings, and correctness
// ABOUTME: Reminds agent to verify claims and check documentation

package checks

// FactualCheck analyzes factual accuracy concerns at configurable rigor levels.
type FactualCheck struct {
	level string
}

// NewFactualCheck creates a FactualCheck at the given level.
func NewFactualCheck(level string) *FactualCheck {
	return &FactualCheck{level: level}
}

// Name returns the check name.
func (c *FactualCheck) Name() string { return "factual" }

// Analyze runs factual accuracy analysis and returns instructions based on level and context.
func (c *FactualCheck) Analyze(ctx CheckContext) CheckResult {
	r := CheckResult{
		Name:  "factual",
		Level: c.level,
		Score: 100,
	}

	// Minimal: basic verification
	r.Instructions = append(r.Instructions, "Verify claims against official documentation before stating them")
	r.Score = 80

	if c.level == "minimal" {
		return r
	}

	// Standard: + API version checks, library compatibility
	r.Instructions = append(r.Instructions,
		"Check API version compatibility for all referenced libraries",
		"Verify function signatures match the installed library version",
	)
	r.Score = 70

	if c.level == "standard" {
		return r
	}

	// Strict: + deprecation warnings, changelog review, cross-reference
	r.Instructions = append(r.Instructions,
		"Check for deprecation warnings in all referenced APIs",
		"Review changelogs for breaking changes in dependencies",
		"Cross-reference multiple documentation sources for accuracy",
		"Verify code examples compile and run correctly",
	)
	r.Score = 50

	if c.level == "strict" {
		return r
	}

	// Paranoid: + formal correctness, proof of behavior
	r.Instructions = append(r.Instructions,
		"Verify all numerical claims with reproducible calculations",
		"Cite specific documentation sections for technical claims",
		"Test all code snippets in isolation before recommending",
	)
	r.Score = 40

	return r
}
