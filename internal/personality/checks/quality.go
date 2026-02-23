// ABOUTME: Code quality check for test coverage, error handling, and lint compliance
// ABOUTME: Scales instructions from basic hygiene to comprehensive quality gates

package checks

// QualityCheck analyzes code quality at configurable rigor levels.
type QualityCheck struct {
	level string
}

// NewQualityCheck creates a QualityCheck at the given level.
func NewQualityCheck(level string) *QualityCheck {
	return &QualityCheck{level: level}
}

// Name returns the check name.
func (c *QualityCheck) Name() string { return "quality" }

// Analyze runs quality analysis and returns instructions based on level and context.
func (c *QualityCheck) Analyze(ctx CheckContext) CheckResult {
	r := CheckResult{
		Name:  "quality",
		Level: c.level,
		Score: 100,
	}

	// Minimal: basic hygiene
	r.Instructions = append(r.Instructions, "Run linter before committing")
	r.Score = 80

	if c.level == "minimal" {
		if !ctx.HasTests && ctx.LinesChanged > 0 {
			r.Warnings = append(r.Warnings, "No tests detected for changed code")
		}
		return r
	}

	// Standard: + error handling, test coverage
	r.Instructions = append(r.Instructions,
		"Handle all errors explicitly; do not ignore return values",
		"Ensure test coverage for new and modified code paths",
	)
	r.Score = 70

	if !ctx.HasTests && ctx.LinesChanged > 0 {
		r.Warnings = append(r.Warnings, "No tests detected for changed code")
	}
	if !ctx.HasErrorHandling {
		r.Warnings = append(r.Warnings, "Error handling gaps detected")
	}

	if c.level == "standard" {
		return r
	}

	// Strict: + comprehensive coverage, code review, documentation
	r.Instructions = append(r.Instructions,
		"Target 80%+ test coverage for all packages",
		"Add documentation for all exported functions and types",
		"Run static analysis and address all findings",
		"Ensure consistent code style across the codebase",
	)
	r.Score = 50

	if !ctx.HasTests {
		r.Warnings = append(r.Warnings, "Tests are mandatory at strict level")
	}

	if c.level == "strict" {
		return r
	}

	// Paranoid: + mutation testing, property-based tests
	r.Instructions = append(r.Instructions,
		"Consider mutation testing to verify test effectiveness",
		"Add property-based tests for core business logic",
		"Review all edge cases and boundary conditions",
	)
	r.Score = 40

	return r
}
